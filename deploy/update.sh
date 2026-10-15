#!/usr/bin/env bash
set -euo pipefail

update() (
  local readonly owner="${1:?Specify the Linux account that manages saveme.}"
  local readonly root_uid=0 failure=1 script_index=0
  local readonly base="$(cd -- "$(dirname -- "${BASH_SOURCE[$script_index]}")/.." && pwd)"
  local readonly config=/etc/saveme/config.json service=saved.service
  local readonly cli=/usr/local/bin/saveme daemon=/usr/local/libexec/saved
  local readonly share=/usr/local/share/saveme mode=0755 data_mode=0644
  local readonly state_dir=/var/lib/saveme state_mode=0600 state_files='state.json state.bak.json'
  local readonly backup_pattern=saveme-update.XXXXXXXX empty='' zero=0 one=1 format='%s\n'
  local readonly denied='Run with sudo and select a non-root account.'
  local readonly rollback_message='Update failed; restored previous binaries. Inspect saved.service.'
  local readonly updated_message='Updated. Configuration, state, policies and service drop-ins preserved.'
  local readonly incomplete='Existing installation is incomplete; refusing update.'
  [[ "$EUID" -eq "$root_uid" && "$(id -u "$owner")" -ne "$root_uid" ]] || { printf "$format" "$denied" >&2; return "$failure"; }
  local tool
  for tool in systemctl install mktemp cp runuser chattr; do command -v "$tool" >/dev/null; done
  local path
  for path in "$cli" "$daemon" "$config"; do
    [[ -f "$path" && ! -L "$path" ]] || { printf "$format" "$incomplete" >&2; return "$failure"; }
  done
  [[ ! -L "$share" && ! -L "$share/VERSION" ]] || { printf "$format" "$incomplete" >&2; return "$failure"; }
  local backup
  backup="$(mktemp -d "/var/tmp/$backup_pattern")"
  readonly backup
  local stopped="$zero" committed="$zero" had_version="$zero" state_saved="$zero"
  source "$base/deploy/check.sh"
  # Khôi phục state cũ vì daemon mới có thể đã ghi state v2 mà bản cũ không đọc được.
  restore_state() {
    local name
    for name in $state_files; do
      chattr -i "$state_dir/$name" 2>/dev/null || true
      if [[ -f "$backup/state/$name" ]]; then
        install -m "$state_mode" "$backup/state/$name" "$state_dir/$name"
        chattr +i "$state_dir/$name"
      else
        rm -f -- "$state_dir/$name"
      fi
    done
  }
  rollback() {
    if [[ "$stopped" -eq "$one" && "$committed" -eq "$zero" ]]; then
      systemctl stop "$service" || true
      install -m "$mode" "$backup/saveme" "$cli"
      install -m "$mode" "$backup/saved" "$daemon"
      [[ "$state_saved" -ne "$one" ]] || restore_state || true
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
  install -d "$backup/state"
  local name
  for name in $state_files; do
    [[ ! -f "$state_dir/$name" ]] || cp -- "$state_dir/$name" "$backup/state/$name"
  done
  state_saved="$one"
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
