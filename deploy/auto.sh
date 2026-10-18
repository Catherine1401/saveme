#!/usr/bin/env bash

setup_auto() (
  local readonly base="${1:?Verified bundle directory required}"
  local readonly helper=saveme-updater.service unit=/etc/systemd/system/saveme-updater.service
  local readonly current=/usr/local/libexec/saveme/current mode=0644 success=0
  local readonly version="$base/VERSION" binary="$base/bin/saveme-update" service="$base/deploy/saveme-updater.service"
  if [[ ! -f "$version" ]]; then
    if systemctl cat "$helper" >/dev/null 2>&1; then systemctl disable --now "$helper"; fi
    rm -f -- "$current"
    return "$success"
  fi
  if systemctl cat "$helper" >/dev/null 2>&1; then systemctl stop "$helper"; fi
  "$binary" -install-runtime -source "$base"
  install -m "$mode" "$service" "$unit"
  systemctl daemon-reload
  systemctl enable --now "$helper"
)
