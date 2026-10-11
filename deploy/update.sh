#!/usr/bin/env bash
set -euo pipefail

update() (
  local readonly owner="${1:?Specify the Linux account that manages saveme.}"
  local readonly root_uid=0 failure=1 script_index=0
  local readonly base="$(cd -- "$(dirname -- "${BASH_SOURCE[$script_index]}")/.." && pwd)"
  local readonly config=/etc/saveme/config.json service=saved.service
  local readonly cli=/usr/local/bin/saveme daemon=/usr/local/libexec/saved
  local readonly share=/usr/local/share/saveme mode=0755 data_mode=0644
  local readonly backup_pattern=saveme-update.XXXXXXXX empty='' zero=0 one=1 format='%s\n'
  local readonly denied='Run with sudo and select a non-root account.'
  local readonly rollback_message='Update failed; restored previous binaries. Inspect saved.service.'
  local readonly updated_message='Updated. Configuration, state, policies and service drop-ins preserved.'
  local readonly incomplete='Existing installation is incomplete; refusing update.'
  [[ "$EUID" -eq "$root_uid" && "$(id -u "$owner")" -ne "$root_uid" ]] || { printf "$format" "$denied" >&2; return "$failure"; }
  local tool
  for tool in systemctl install mktemp cp runuser; do command -v "$tool" >/dev/null; done
  local path
  for path in "$cli" "$daemon" "$config"; do
    [[ -f "$path" && ! -L "$path" ]] || { printf "$format" "$incomplete" >&2; return "$failure"; }
  done
  [[ ! -L "$share" && ! -L "$share/VERSION" ]] || { printf "$format" "$incomplete" >&2; return "$failure"; }
  local backup
  backup="$(mktemp -d "/var/tmp/$backup_pattern")"
  readonly backup
  local stopped="$zero" committed="$zero" had_version="$zero"
  source "$base/deploy/check.sh"
  rollback() {
    if [[ "$stopped" -eq "$one" && "$committed" -eq "$zero" ]]; then
      systemctl stop "$service" || true
      install -m "$mode" "$backup/saveme" "$cli"
      install -m "$mode" "$backup/saved" "$daemon"
      if [[ "$had_version" -eq "$one" ]]; then
        install -m "$data_mode" "$backup/VERSION" "$share/VERSION"
      else
        rm -f -- "$share/VERSION"
      fi
      systemctl start "$service" || true
      printf "$format" "$rollback_message" >&2
    fi
    rm -rf -- "$backup"
  }
  trap rollback EXIT
  cp -- "$cli" "$backup/saveme"
  cp -- "$daemon" "$backup/saved"
  if [[ -f "$share/VERSION" ]]; then
    cp -- "$share/VERSION" "$backup/VERSION"
    had_version="$one"
  fi
  runuser -u "$owner" -- "$base/bin/saveme" -status >/dev/null
  "$base/bin/saved" -check-update -config "$config"
  stopped="$one"
  systemctl stop "$service"
  "$base/bin/saved" -check-update -config "$config"
  install -m "$mode" "$base/bin/saveme" "$cli"
  install -m "$mode" "$base/bin/saved" "$daemon"
  systemctl start "$service"
  ready "$owner" "$cli"
  if [[ -f "$base/VERSION" ]]; then
    install -d "$share"
    install -m "$data_mode" "$base/VERSION" "$share/VERSION"
  else
    rm -f -- "$share/VERSION"
  fi
  committed="$one"
  printf "$format" "$updated_message"
)

update "$@"
