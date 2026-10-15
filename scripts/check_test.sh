#!/usr/bin/env bash
set -euo pipefail

test_rollback_readiness() (
  local readonly script_index=0 failure=1 success=0 attempts=10 format='%s\n' newline=$'\n'
  local readonly base="$(cd -- "$(dirname -- "${BASH_SOURCE[$script_index]}")/.." && pwd)"
  local readonly expected='PASS rollback readiness' missing='Rollback readiness returned an inherited failure.'
  local output result="$success"
  output="$(bash -s -- "$base/deploy/check.sh" "$attempts" "$expected" <<'BASH'
set -euo pipefail
source "$1"
recover() (
  local readonly limit="$2" message="$3" zero=0 one=1 owner=owner client=client format='%s\n'
  local checks="$zero"
  systemctl() { return "$zero"; }
  runuser() { ((checks+=one)); [[ "$checks" -gt "$limit" ]]; }
  sleep() { :; }
  recovered() {
    if ready "$owner" "$client"; then printf "$format" "$message"; fi
  }
  trap recovered EXIT
  ready "$owner" "$client"
)
recover "$@"
BASH
)" || result="$?"
  [[ "$result" -eq "$failure" && "${output##*"$newline"}" == "$expected" ]] || { printf "$format" "$missing" >&2; return "$failure"; }
  printf "$format" "$expected"
)

test_rollback_readiness
