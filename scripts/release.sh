#!/usr/bin/env bash
set -euo pipefail

package() (
  local readonly version="${1:?Specify a stable version such as v0.1.0.}"
  local readonly script_index=0 pattern='^v?[0-9]+\.[0-9]+\.[0-9]+$' failure=1 format='%s\n'
  local readonly root="$(cd -- "$(dirname -- "${BASH_SOURCE[$script_index]}")/.." && pwd)"
  local readonly directory="dist/$version" checksum_name=checksums.txt package_name=package linux=linux windows=windows disabled=0
  local readonly cleanup='rm -rf -- "$temporary"' temporary_pattern=saveme-release.XXXXXXXX
  [[ "$version" =~ $pattern ]] || { printf "$format" 'Expected a stable version such as 0.1.0 or v0.1.0.' >&2; return "$failure"; }
  local temporary
  temporary="$(mktemp -d "${TMPDIR:-/tmp}/$temporary_pattern")"
  readonly temporary
  trap "$cleanup" EXIT
  mkdir -p "$root/$directory"
  local arch command
  for arch in amd64 arm64; do
    local bundle="$temporary/$arch/$package_name"
    mkdir -p "$bundle/bin" "$bundle/deploy"
    for command in saveme saved; do
      (cd "$root" && CGO_ENABLED="$disabled" GOOS="$linux" GOARCH="$arch" \
        go build -mod=readonly -trimpath -ldflags='-s -w' -o "$bundle/bin/$command" "./cmd/$command")
    done
    cp -- "$root/deploy/"{install.sh,update.sh,uninstall.sh,check.sh,config.json,saved.service,chrome.json,firefox.json} "$bundle/deploy/"
    cp -- "$root/LICENSE" "$bundle/LICENSE"
    printf "$format" "$version" > "$bundle/VERSION"
    tar -czf "$root/$directory/saveme_${version#v}_${linux}_$arch.tar.gz" -C "$bundle" bin deploy LICENSE VERSION
  done
  for arch in amd64 arm64; do
    local bundle="$temporary/$windows-$arch/$package_name"
    mkdir -p "$bundle/bin" "$bundle/deploy/$windows"
    for command in saveme saved; do
      (cd "$root" && CGO_ENABLED="$disabled" GOOS="$windows" GOARCH="$arch" \
        go build -mod=readonly -trimpath -ldflags='-s -w' -o "$bundle/bin/$command.exe" "./cmd/$command")
    done
    cp -- "$root/deploy/$windows/"{install.ps1,update.ps1,uninstall.ps1,common.ps1,config.json} "$bundle/deploy/$windows/"
    cp -- "$root/LICENSE" "$bundle/LICENSE"
    printf "$format" "$version" > "$bundle/VERSION"
    (cd "$bundle" && zip -qr "$root/$directory/saveme_${version#v}_${windows}_$arch.zip" bin deploy LICENSE VERSION)
  done
  (cd "$root/$directory" && sha256sum "saveme_${version#v}_${linux}_amd64.tar.gz" \
    "saveme_${version#v}_${linux}_arm64.tar.gz" "saveme_${version#v}_${windows}_amd64.zip" \
    "saveme_${version#v}_${windows}_arm64.zip" > "$checksum_name")
  printf "$format" "Bundles and SHA-256 manifest written to $directory/"
)

package "$@"
