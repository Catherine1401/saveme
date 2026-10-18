#!/usr/bin/env bash
set -euo pipefail

uninstall() (
  local readonly root_uid=0 failure=1 format='%s\n'
  local readonly service=saved.service config=/etc/saveme/config.json config_dir=/etc/saveme
  local readonly cli=/usr/local/bin/saveme daemon=/usr/local/libexec/saved share=/usr/local/share/saveme
  local readonly state_dir=/var/lib/saveme service_path=/etc/systemd/system/saved.service
  local readonly dropin_dir=/etc/systemd/system/saved.service.d forward_user=saveme-dns
  local readonly chrome=/etc/opt/chrome/policies/managed/saveme.json chromium=/etc/chromium/policies/managed/saveme.json
  local readonly firefox=/etc/firefox/policies/policies.json
  local readonly denied='Run with sudo.' missing='No saveme installation found.'
  local readonly helper=saveme-updater.service helper_root=/usr/local/libexec/saveme helper_service=/etc/systemd/system/saveme-updater.service
  local readonly done_message='Uninstalled. Restart browsers to drop the DNS-over-HTTPS policy.'
  [[ "$EUID" -eq "$root_uid" ]] || { printf "$format" "$denied" >&2; return "$failure"; }
  [[ -x "$daemon" && -f "$config" ]] || { printf "$format" "$missing" >&2; return "$failure"; }
  # Từ chối gỡ khi còn domain có khóa hoặc lịch, giống điều kiện update.
  "$daemon" -check-update -config "$config"
  systemctl disable --now "$helper" 2>/dev/null || true
  systemctl disable --now "$service" || true
  "$daemon" -check-update -config "$config"
  chattr -i "$service_path" 2>/dev/null || true
  local path
  for path in "$state_dir"/*; do
    [[ -e "$path" ]] && chattr -i "$path" 2>/dev/null || true
  done
  rm -rf -- "$service_path" "$dropin_dir" "$state_dir" "$config_dir" "$cli" "$daemon" "$share" "$chrome" "$chromium" "$firefox" "$helper_root" "$helper_service"
  systemctl daemon-reload
  if id "$forward_user" >/dev/null 2>&1; then
    userdel "$forward_user"
  fi
  printf "$format" "$done_message"
)

uninstall "$@"
