#!/usr/bin/env bash
set -euo pipefail

update() (
  local readonly owner="${1:?Specify the Linux account that manages saveme.}"
  local readonly root_uid=0 failure=1 script_index=0
  local readonly base="$(cd -- "$(dirname -- "${BASH_SOURCE[$script_index]}")/.." && pwd)"
  local readonly auto_script="$base/deploy/auto.sh"
  local readonly config=/etc/saveme/config.json service=saved.service
  local readonly cli=/usr/local/bin/saveme daemon=/usr/local/libexec/saved
  local readonly share=/usr/local/share/saveme mode=0755 data_mode=0644
  local readonly lock=/run/saveme-update.lock snapshot_name=commitment.json status_name=status.json
  local readonly backup_pattern=saveme-update.XXXXXXXX zero=0 one=1 format='%s\n'
  local readonly denied='Run with sudo and select a non-root account.'
  local readonly rollback_message='Update failed; restored previous binaries. Inspect saved.service.'
  local readonly updated_message='Updated. Configuration, state, policies and service drop-ins preserved.'
  local readonly incomplete='Existing installation is incomplete; refusing update.'
  [[ "$EUID" -eq "$root_uid" && "$(id -u "$owner")" -ne "$root_uid" ]] || { printf "$format" "$denied" >&2; return "$failure"; }
  local tool
  for tool in systemctl install mktemp cp runuser flock; do command -v "$tool" >/dev/null; done
  local path
  for path in "$cli" "$daemon" "$config"; do
    [[ -f "$path" && ! -L "$path" ]] || { printf "$format" "$incomplete" >&2; return "$failure"; }
  done
  [[ ! -L "$share" && ! -L "$share/VERSION" ]] || { printf "$format" "$incomplete" >&2; return "$failure"; }
  [[ ! -L "$lock" ]] || return "$failure"
  umask 077
  local update_lock
  exec {update_lock}>"$lock"
  flock -n "$update_lock"
  local backup
  backup="$(mktemp -d "/var/tmp/$backup_pattern")"
  readonly backup
  local stopped="$zero" committed="$zero" had_version="$zero" replaced="$zero"
  source "$base/deploy/check.sh"
  verify_commitment() {
    runuser -u "$owner" -- "$cli" -status >"$backup/$status_name"
    "$backup/checker" -verify-update "$backup/$snapshot_name" -update-status "$backup/$status_name" -config "$config"
  }
  rollback() {
    if [[ "$stopped" -eq "$one" && "$committed" -eq "$zero" ]]; then
      if ! systemctl stop "$service"; then
        printf "$format" "Rollback could not stop service; recovery files retained at $backup" >&2
        return "$failure"
      fi
      install -m "$mode" "$backup/saveme" "$cli" || return "$failure"
      install -m "$mode" "$backup/saved" "$daemon" || return "$failure"
      if [[ "$had_version" -eq "$one" ]]; then
        install -m "$data_mode" "$backup/VERSION" "$share/VERSION" || return "$failure"
      else
        rm -f -- "$share/VERSION"
      fi
      if [[ "$replaced" -eq "$one" ]]; then
        if ! "$backup/checker" -restore-update "$backup/$snapshot_name" -config "$config"; then
          printf "$format" "Rollback could not restore commitment; service stays stopped, recovery files retained at $backup" >&2
          return "$failure"
        fi
      fi
      if ! systemctl start "$service" || ! ready "$owner" "$cli"; then
        printf "$format" "Rollback could not recover service; recovery files retained at $backup" >&2
        return "$failure"
      fi
      if [[ "$replaced" -eq "$one" ]] && ! verify_commitment; then
        printf "$format" "Rollback verification failed; recovery files retained at $backup" >&2
        return "$failure"
      fi
      printf "$format" "$rollback_message" >&2
    fi
    rm -rf -- "$backup"
  }
  trap rollback EXIT
  cp -- "$cli" "$backup/saveme"
  cp -- "$daemon" "$backup/saved"
  cp -- "$base/bin/saved" "$backup/checker"
  if [[ -f "$share/VERSION" ]]; then
    cp -- "$share/VERSION" "$backup/VERSION"
    had_version="$one"
  fi
  runuser -u "$owner" -- "$base/bin/saveme" -status >/dev/null
  "$base/bin/saved" -check-preserved-update -config "$config"
  stopped="$one"
  systemctl stop "$service"
  "$backup/checker" -update-snapshot "$backup/$snapshot_name" -config "$config"
  replaced="$one"
  install -m "$mode" "$base/bin/saveme" "$cli"
  install -m "$mode" "$base/bin/saved" "$daemon"
  systemctl start "$service"
  ready "$owner" "$cli"
  verify_commitment
  if [[ -f "$base/VERSION" ]]; then
    install -d "$share"
    install -m "$data_mode" "$base/VERSION" "$share/VERSION"
  else
    rm -f -- "$share/VERSION"
  fi
  source "$auto_script"
  setup_auto "$base"
  committed="$one"
  printf "$format" "$updated_message"
)

update "$@"
