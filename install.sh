#!/usr/bin/env bash
set -euo pipefail

fail() {
  local readonly message="$1" format='%s\n' failure=1
  printf "$format" "$message" >&2
  exit "$failure"
}

help() {
  local readonly text='Usage: bash install.sh [--user ACCOUNT] [--version vX.Y.Z] [--update]
       bash install.sh --version vX.Y.Z --archive FILE --checksums FILE

Downloads a verified Linux binary bundle. Run as an ordinary user; sudo is
used only for deployment. --update keeps configuration, state and policies.
Updates require no stored lock or daily schedule. Offline bundles require
both --archive and --checksums. No shell profile is modified.'
  local readonly format='%s\n'
  printf "$format" "$text"
}

architecture() {
  local readonly system="$(uname -s)" machine="$(uname -m)" linux=Linux
  local readonly amd=amd64 arm=arm64 format='%s\n'
  local readonly unsupported='saveme requires Linux with systemd.'
  [[ "$system" == "$linux" ]] || fail "$unsupported"
  case "$machine" in
    x86_64|amd64) printf "$format" "$amd" ;;
    aarch64|arm64) printf "$format" "$arm" ;;
    *) fail "Unsupported architecture: $machine" ;;
  esac
}

latest() {
  local readonly base="$1" suffix=/latest format='%{url_effective}'
  local readonly connection_timeout=15 timeout=120
  local readonly unavailable='No published release found. Build from source instead.'
  local resolved
  resolved="$(curl --proto '=https' --proto-redir '=https' --tlsv1.2 -fsSL \
    --connect-timeout "$connection_timeout" --max-time "$timeout" -o /dev/null -w "$format" "$base$suffix")" || fail "$unavailable"
  local readonly output_format='%s\n'
  printf "$output_format" "${resolved##*/}"
}

download() {
  local readonly url="$1" destination="$2"
  local readonly retries=3 connection_timeout=15 timeout=300
  curl --proto '=https' --proto-redir '=https' --tlsv1.2 -fSL --retry "$retries" \
    --connect-timeout "$connection_timeout" --max-time "$timeout" -o "$destination" "$url"
}

verify() {
  local readonly directory="$1" asset="$2" checksums="$3" entry=selected.sha256
  local readonly absent='Missing or duplicate archive checksum.' mismatch='Checksum mismatch; nothing installed.'
  local readonly selection='$2 == file { print; count++ } END { if (count != 1) exit 1 }'
  awk -v file="$asset" "$selection" "$checksums" > "$directory/$entry" || fail "$absent"
  (cd "$directory" && sha256sum --check --status "$entry") || fail "$mismatch"
}

extract() {
  local readonly archive="$1" destination="$2"
  local readonly members=(bin/saveme bin/saved deploy/install.sh deploy/update.sh
    deploy/check.sh deploy/config.json deploy/saved.service deploy/chrome.json
    deploy/firefox.json LICENSE VERSION)
  mkdir -p "$destination"
  tar --no-same-owner --no-same-permissions -xzf "$archive" -C "$destination" -- "${members[@]}"
  local member
  for member in "${members[@]}"; do
    [[ -f "$destination/$member" && ! -L "$destination/$member" ]] || fail "Invalid bundle file: $member"
  done
  chmod +x "$destination/bin/saveme" "$destination/bin/saved"
}

main() (
  local readonly base=https://github.com/Catherine1401/saveme/releases
  local readonly empty='' zero=0 one=1 two=2 tag_pattern='^v?[0-9]+\.[0-9]+\.[0-9]+$'
  local readonly checksum_name=checksums.txt install_name=install.sh update_name=update.sh
  local readonly temporary_pattern=saveme-install.XXXXXXXX payload_name=payload
  local readonly cleanup='rm -rf -- "$workspace"' format='%s\n'
  local readonly root_message='Run this script as an ordinary user, without sudo.'
  local readonly owner_message='Select a non-root Linux account with --user.'
  local readonly offline_message='Offline install needs --archive, --checksums and --version.'
  local readonly curl_message='Missing command: curl' tag_message='Expected a stable version such as 0.1.0 or v0.1.0.'
  local readonly version_message='Bundle version does not match the requested version.'
  local version="$empty" archive="$empty" checksums="$empty" update="$zero"
  local owner="${USER:-$empty}"
  while [[ "$#" -gt "$zero" ]]; do
    case "$1" in
      --help|-h) help; return ;;
      --update) update="$one"; shift ;;
      --user|--version|--archive|--checksums)
        [[ "$#" -ge "$two" && -n "$2" ]] || fail "Missing value for $1"
        case "$1" in
          --user) owner="$2" ;; --version) version="$2" ;;
          --archive) archive="$2" ;; --checksums) checksums="$2" ;;
        esac
        shift "$two" ;;
      *) fail "Unknown option: $1" ;;
    esac
  done
  [[ "$EUID" -ne "$zero" ]] || fail "$root_message"
  [[ -n "$owner" && "$(id -u "$owner")" -ne "$zero" ]] || fail "$owner_message"
  local tool
  for tool in tar sha256sum awk mktemp sudo; do
    command -v "$tool" >/dev/null || fail "Missing command: $tool"
  done
  if [[ -n "$archive" || -n "$checksums" ]]; then
    [[ -f "$archive" && -f "$checksums" && -n "$version" ]] || fail "$offline_message"
  else
    command -v curl >/dev/null || fail "$curl_message"
  fi
  local arch workspace
  arch="$(architecture)"
  readonly arch
  [[ -n "$version" ]] || version="$(latest "$base")"
  [[ "$version" =~ $tag_pattern ]] || fail "$tag_message"
  workspace="$(mktemp -d "${TMPDIR:-/tmp}/$temporary_pattern")"
  readonly workspace
  trap "$cleanup" EXIT
  local readonly asset="saveme_${version#v}_linux_$arch.tar.gz"
  local readonly package="$workspace/$payload_name"
  if [[ -n "$archive" ]]; then
    cp -- "$archive" "$workspace/$asset"
    cp -- "$checksums" "$workspace/$checksum_name"
  else
    printf "$format" "Downloading saveme $version for linux/$arch"
    download "$base/download/$version/$asset" "$workspace/$asset"
    download "$base/download/$version/$checksum_name" "$workspace/$checksum_name"
  fi
  verify "$workspace" "$asset" "$workspace/$checksum_name"
  extract "$workspace/$asset" "$package"
  [[ "$(cat "$package/VERSION")" == "$version" ]] || fail "$version_message"
  local action="$install_name"
  [[ "$update" -eq "$zero" ]] || action="$update_name"
  sudo bash "$package/deploy/$action" "$owner"
)

main "$@"
