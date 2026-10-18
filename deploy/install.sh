#!/usr/bin/env bash
set -euo pipefail

readonly root_uid=0
readonly failure_status=1
readonly script_index=0
readonly owner_name="${1:?Specify the Linux account that manages saveme.}"
readonly owner_uid="$(id -u "$owner_name")"
readonly base_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[$script_index]}")/.." && pwd)"
readonly forward_user=saveme-dns
readonly config_dir=/etc/saveme
readonly state_dir=/var/lib/saveme
readonly service_path=/etc/systemd/system/saved.service
readonly chrome_dir=/etc/opt/chrome/policies/managed
readonly chromium_dir=/etc/chromium/policies/managed
readonly firefox_dir=/etc/firefox/policies
readonly firefox_path="$firefox_dir/policies.json"
readonly policy_name=saveme.json
readonly file_mode=0644
readonly private_mode=0600
readonly dir_mode=0755
readonly state_mode=0700
readonly binary_mode=0755
readonly config_name=config.json
readonly cli_dir=/usr/local/bin
readonly daemon_dir=/usr/local/libexec
readonly no_login=/usr/sbin/nologin
readonly print_format='%s\n'
readonly config_pattern='s/"allowed_uid": 1000/"allowed_uid": '
readonly config_end='/'
readonly ready_message='Installed. Check systemctl status saved and browser policies before locking.'
readonly denied_message='Run with sudo and select a non-root account.'
readonly existing_message='Target files or saveme-dns account exist; refusing to overwrite.'
readonly share_dir=/usr/local/share/saveme
readonly helper_path="$base_dir/bin/saveme-update" helper_root=/usr/local/libexec/saveme
readonly helper_service=/etc/systemd/system/saveme-updater.service auto_script="$base_dir/deploy/auto.sh"

source "$base_dir/deploy/check.sh"

if [[ "$EUID" -ne "$root_uid" || "$owner_uid" -eq "$root_uid" ]]; then
  printf "$print_format" "$denied_message" >&2
  exit "$failure_status"
fi

for binary in "$base_dir/bin/saveme" "$base_dir/bin/saved" "$helper_path"; do
  test -x "$binary"
done

for tool in systemctl timedatectl chattr install useradd sed; do
  command -v "$tool" >/dev/null
done

for target in "$service_path" "$config_dir/$config_name" "$firefox_path" "$chrome_dir/$policy_name" "$chromium_dir/$policy_name" "$cli_dir/saveme" "$daemon_dir/saved" "$state_dir" "$share_dir" "$helper_root" "$helper_service"; do
  if [[ -e "$target" || -L "$target" ]]; then
    printf "$print_format" "$existing_message" >&2
    exit "$failure_status"
  fi
done

if id "$forward_user" >/dev/null 2>&1; then
  printf "$print_format" "$existing_message" >&2
  exit "$failure_status"
fi

preflight "$base_dir/bin/saved" "$base_dir/deploy/config.json"

useradd --system --no-create-home --shell "$no_login" "$forward_user"
install -d -m "$dir_mode" "$cli_dir" "$daemon_dir" "$config_dir" "$chrome_dir" "$chromium_dir" "$firefox_dir"
install -d -m "$state_mode" "$state_dir"
install -m "$binary_mode" "$base_dir/bin/saveme" "$cli_dir/saveme"
install -m "$binary_mode" "$base_dir/bin/saved" "$daemon_dir/saved"
sed "$config_pattern$owner_uid$config_end" "$base_dir/deploy/config.json" > "$config_dir/$config_name"
chmod "$private_mode" "$config_dir/$config_name"
install -m "$file_mode" "$base_dir/deploy/chrome.json" "$chrome_dir/$policy_name"
install -m "$file_mode" "$base_dir/deploy/chrome.json" "$chromium_dir/$policy_name"
install -m "$file_mode" "$base_dir/deploy/firefox.json" "$firefox_path"
install -m "$file_mode" "$base_dir/deploy/saved.service" "$service_path"
chattr +i "$service_path"
systemctl daemon-reload
systemctl enable --now saved.service
ready "$owner_name" "$cli_dir/saveme"
if [[ -f "$base_dir/VERSION" ]]; then
  install -d -m "$dir_mode" "$share_dir"
  install -m "$file_mode" "$base_dir/VERSION" "$share_dir/VERSION"
fi
source "$auto_script"
setup_auto "$base_dir"
printf "$print_format" "$ready_message"
