#!/usr/bin/env bash
set -euo pipefail

mock_download() {
  local readonly directory="$1" mode=0755
  cat > "$directory/curl" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail
main() {
  local readonly empty='' zero=0 two=2 failure=22
  local readonly latest_suffix=/latest manifest_suffix=/checksums.txt format='%s\n'
  local readonly tag="$SAVEME_TEST_VERSION" missing=missing
  local output="$empty" url="${!#}"
  while [[ "$#" -gt "$zero" ]]; do
    case "$1" in -o) output="$2"; shift "$two" ;; *) shift ;; esac
  done
  [[ "${SAVEME_TEST_MODE:-$empty}" != "$missing" ]] || return "$failure"
  case "$url" in
    *"$latest_suffix") printf "$format" "${url%$latest_suffix}/tag/$tag" ;;
    *"$manifest_suffix") cp -- "$SAVEME_TEST_BUNDLES/checksums.txt" "$output" ;;
    *) cp -- "$SAVEME_TEST_BUNDLES/${url##*/}" "$output" ;;
  esac
}
main "$@"
SCRIPT
  chmod "$mode" "$directory/curl"
}

mock_deployment() {
  local readonly directory="$1" mode=0755
  cat > "$directory/sudo" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail
main() {
  local readonly shell_name=bash install_name=install.sh update_name=update.sh format='%s\n'
  local readonly script="$2" owner="$3" base="$(dirname -- "$(dirname -- "$2")")"
  [[ "$1" == "$shell_name" && -f "$script" ]]
  [[ "${script##*/}" == "$install_name" || "${script##*/}" == "$update_name" ]]
  [[ -x "$base/bin/saveme" && -x "$base/bin/saved" ]]
  printf "$format" "$(cat "$base/VERSION")" "$owner" "${script##*/}" > "$SAVEME_TEST_REPORT"
}
main "$@"
SCRIPT
  cat > "$directory/uname" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail
main() {
  local readonly machine_flag=-m empty='' format='%s\n'
  if [[ "$1" == "$machine_flag" && -n "${SAVEME_TEST_ARCH:-$empty}" ]]; then
    printf "$format" "$SAVEME_TEST_ARCH"
  else
    /usr/bin/uname "$@"
  fi
}
main "$@"
SCRIPT
  chmod "$mode" "$directory/sudo" "$directory/uname"
}

check_bootstrap() (
  local readonly script_index=0 default_tag=v0.1.0 zero=0 failure=1
  local readonly root="$(cd -- "$(dirname -- "${BASH_SOURCE[$script_index]}")/.." && pwd)"
  local readonly bundles="${1:-$root/dist/$default_tag}" template=saveme-bootstrap.XXXXXXXX
  local readonly tag="${bundles##*/}" cleanup='rm -rf -- "$work"' marker=deployment output=output
  local readonly format='%s\n'
  local readonly mismatch='Checksum mismatch' unavailable='No published release' unsupported='Unsupported architecture'
  [[ "$EUID" -ne "$zero" ]] || { printf "$format" 'Run bootstrap tests as an ordinary user.' >&2; return "$failure"; }
  local work
  work="$(mktemp -d "${TMPDIR:-/tmp}/$template")"
  readonly work
  trap "$cleanup" EXIT
  mkdir -p "$work/tools" "$work/tmp"
  mock_download "$work/tools"
  mock_deployment "$work/tools"
  export PATH="$work/tools:$PATH" TMPDIR="$work/tmp"
  export SAVEME_TEST_BUNDLES="$bundles" SAVEME_TEST_REPORT="$work/$marker" SAVEME_TEST_VERSION="$tag"
  bash "$root/install.sh" --user "$USER" > "$work/$output"
  grep -Fxq "$tag" "$work/$marker"
  grep -qx "$USER" "$work/$marker"
  rm -- "$work/$marker"
  bash "$root/install.sh" --update --version "$tag" --user "$USER" > "$work/$output"
  grep -qx update.sh "$work/$marker"
  rm -- "$work/$marker"
  if SAVEME_TEST_MODE=missing bash "$root/install.sh" > "$work/$output" 2>&1; then return "$failure"; fi
  grep -q "$unavailable" "$work/$output"
  [[ ! -e "$work/$marker" ]]
  if SAVEME_TEST_ARCH=unsupported bash "$root/install.sh" --version "$tag" > "$work/$output" 2>&1; then return "$failure"; fi
  grep -q "$unsupported" "$work/$output"
  local readonly corrupt="$work/corrupt.tar.gz" archive="$(find "$bundles" -name '*_amd64.tar.gz' -print -quit)"
  cp -- "$archive" "$corrupt"
  printf "$format" 'corrupt' >> "$corrupt"
  if bash "$root/install.sh" --version "$tag" --archive "$corrupt" --checksums "$bundles/checksums.txt" > "$work/$output" 2>&1; then return "$failure"; fi
  grep -q "$mismatch" "$work/$output"
  [[ ! -e "$work/$marker" ]]
  [[ -z "$(find "$work/tmp" -mindepth 1 -print -quit)" ]]
  printf "$format" 'PASS bootstrap: latest and pinned downloads, update delegation, missing release, unsupported architecture, checksum refusal, temporary cleanup'
)

check_bootstrap "$@"
