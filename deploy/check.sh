#!/usr/bin/env bash

ready() {
  local readonly owner="$1" client="$2" service=saved.service attempts=10 pause=1 zero=0
  local attempt
  for ((attempt=zero; attempt<attempts; attempt++)); do
    if systemctl is-active --quiet "$service" && runuser -u "$owner" -- "$client" -status >/dev/null; then
      sleep "$pause"
      systemctl is-active --quiet "$service" && runuser -u "$owner" -- "$client" -status >/dev/null && return "$zero"
    fi
    sleep "$pause"
  done
  local readonly message='Daemon did not become ready. Inspect journalctl -u saved.service.' format='%s\n' failure=1
  printf "$format" "$message" >&2
  return "$failure"
}

preflight() {
  local readonly daemon="$1" configuration="$2" state_parent=/var/lib
  local readonly systemd_path=/run/systemd/system ipv6_path=/proc/sys/net/ipv6/conf/lo/disable_ipv6
  local readonly listener_rule='$5 ~ /^(127[.]0[.]0[.]1|0[.]0[.]0[.]0|\*|\[::\]):(53|5353)$/ || $5 == "[::1]:53" { found=1 } END { exit !found }'
  local readonly no_systemd='systemd must be running.' no_ipv6='IPv6 loopback must be enabled.'
  local readonly occupied='Ports 53 or 5353 are already in use. Inspect ss -lntup.'
  local readonly no_immutable='State filesystem must support immutable files.'
  local readonly zero=0 failure=1 format='%s\n' temporary_pattern=saveme-check.XXXXXXXX
  local tool
  for tool in systemctl timedatectl chattr install useradd sed runuser ss awk mktemp; do
    command -v "$tool" >/dev/null || { printf "$format" "Missing command: $tool" >&2; return "$failure"; }
  done
  [[ -d "$systemd_path" ]] || { printf "$format" "$no_systemd" >&2; return "$failure"; }
  [[ -f "$ipv6_path" && "$(cat "$ipv6_path")" == "$zero" ]] || { printf "$format" "$no_ipv6" >&2; return "$failure"; }
  "$daemon" -check-config -config "$configuration"
  if ss -H -lntu | awk "$listener_rule"; then
    printf "$format" "$occupied" >&2
    return "$failure"
  fi
  local probe
  probe="$(mktemp "$state_parent/$temporary_pattern")"
  readonly probe
  if ! chattr +i "$probe"; then
    rm -f -- "$probe"
    printf "$format" "$no_immutable" >&2
    return "$failure"
  fi
  chattr -i "$probe"
  rm -f -- "$probe"
}
