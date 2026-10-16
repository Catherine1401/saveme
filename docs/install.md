# Install

The installer downloads a release bundle, checks its SHA-256 and uses sudo
for system deployment. You do not need Git, Go or Make for a binary install.
Binary installation requires a published release with bundles and checksums.
If none exists yet, use [the source build](#build-from-source).

## Requirements

- Linux with a running systemd and enabled IPv6 loopback.
- An ordinary account with sudo access.
- Bash, curl, tar, sha256sum, awk and coreutils.
- systemctl, timedatectl, chattr, useradd, runuser, flock and ss.
- A state filesystem supporting immutable files.
- Free `127.0.0.1:53`, `[::1]:53` and `127.0.0.1:5353`, UDP/TCP.

Bundles are built for `amd64` (`x86_64`) and `arm64` (`aarch64`). ARM64 is
cross-compiled; runtime verification is tracked separately from a successful
build. Windows uses separate bundles; see [Windows](#windows). macOS and Linux
systems without systemd are not supported.

Run the installer as the ordinary account that will manage locks:

```sh
curl -fsSL https://raw.githubusercontent.com/Catherine1401/saveme/main/install.sh | bash
```

The downloader runs without root privileges. Only deployment uses sudo.
The installer leaves shell profiles unchanged and installs the TUI in
`/usr/local/bin`. It checks configuration, systemd, IPv6, listeners and
immutable support before creating the forwarding account or installing files.
Multicast-only mDNS listeners on port 5353 do not conflict with the loopback
forwarder.

To download and inspect the script before running it:

```sh
curl -fsSLo install.sh https://raw.githubusercontent.com/Catherine1401/saveme/main/install.sh
less install.sh
bash install.sh --help
bash install.sh --user "$USER"
```

## Choose a version

Use a published stable tag, such as `v0.1.0`, with `--version`:

```sh
bash install.sh --version v0.1.0 --user "$USER"
```

An omitted version resolves to the latest GitHub release. The archive and
checksum manifest are then downloaded from that exact tag. Unsupported systems,
unavailable releases, missing checksums and checksum mismatches stop installation.
SHA-256 checks the archive against the publisher's manifest; it is not a separate
signature or an independent guarantee of authenticity.

## Check the installation

```sh
systemctl status saved.service --no-pager
saveme -status
timedatectl show --property=NTPSynchronized --value
saveme
```

The installer waits for both the service and its socket to respond. NTP must
report `yes` before you can create or edit production locks. JSON status can
report `ERR_CLOCK_UNSYNC` while synchronization is pending.

Review [configuration, browser policies and usage](setup.md). Browser policies
also apply outside lock periods; restart the browser and check its policy page.

## Update

Download the current installer again, then request an update:

```sh
curl -fsSLo install.sh https://raw.githubusercontent.com/Catherine1401/saveme/main/install.sh
bash install.sh --update --version v0.1.0 --user "$USER"
```

Omit `--version` to select the latest release. Updates replace only the two
binaries and the version receipt. Unversioned source updates remove the receipt.
Configuration, state, browser policies,
the service unit and its drop-ins remain in place.

The Linux updater in `v0.1.0` supports active timed locks and daily schedules, including
updates while NTP is untrusted. It takes an exclusive update lock, pauses
mutations, stops the daemon without clearing its nftables rules, then snapshots
the validated commitment before replacing either binary. After restart it checks
stored domains, each lock, the clock floor, owner status, nftables rules and blocked
UDP/TCP DNS responses. A timed lock may expire normally when NTP is trusted.

If startup or verification fails, the updater restores the previous binaries,
version receipt and saved commitment, preserving any newer valid clock floor,
then restarts and verifies the previous daemon. If recovery fails, it reports the
retained recovery directory; do not delete it or reset state. Configuration,
browser policies, the service unit and its drop-ins are preserved.

DNS can be briefly unavailable while the daemon restarts. This does not reset
existing browser connections or provide protection against root changing the
system. State generations and checksums can change during normal startup or
recovery; preservation refers to the commitment, not identical file bytes.

Releases `v0.0.1` and `v0.0.2` contain the previous updater, which refuses stored
locks. Downloading a newer installer does not change an older bundle's update
script. The `v0.1.0` Linux snapshot updater requires version 2 state and refuses
version 1 before stopping the service; it does not migrate older installations.

This also supports the existing source installation when its configuration and
state remain valid. No valid stored state, an unavailable socket, wrong owner
or incomplete installation causes refusal. Existing service fixes stay in place.

## Offline bundle

Download `install.sh`, `checksums.txt` and the archive for your architecture
from [GitHub](https://github.com/Catherine1401/saveme/releases) on another machine.
Copy all three to the Linux machine. No network access is needed for deployment:

```sh
bash install.sh --version v0.1.0 --user "$USER" \
  --archive saveme_0.1.0_linux_amd64.tar.gz --checksums checksums.txt
```

Use the ARM64 archive on an ARM64 system. Add `--update` to update an existing
installation under the same commitment checks.

## Build from source

Install Go 1.24+, Git and Make, then build from the repository:

```sh
git clone https://github.com/Catherine1401/saveme.git
cd saveme
make build
sudo bash deploy/install.sh "$USER"
saveme
```

Source installation uses the same service checks and conflict refusal as binary
installation. After changing local code, run as the ordinary managing account:

```sh
make update
saveme
```

`make update` runs tests, vet and build in order, then uses sudo to call the
existing updater. A failed step stops the update. The updater checks the service
and socket, then verifies the preserved commitment and enforcement after restart.
Active locks do not prevent updating. Configuration, state, policies and service
drop-ins stay in place.
To update binaries you already built and checked, run
`sudo bash deploy/update.sh "$USER"`.

The source Linux updater verifies each domain's commitment after restart and
rolls back on verification failure. It requires version 2 state and refuses
legacy version 1 state before stopping the service; rollback to a version 1
daemon is not supported by this snapshot path. Windows updates and uninstall
retain the stored-lock refusal guard.

## Windows

Windows support is experimental and has not been verified on a real Windows
installation yet. Requirements: Windows 10/11 or Server with PowerShell 5.1,
an Administrator account to install, an ordinary account to manage locks, the
Windows Time service synchronized with a network source, and free
`127.0.0.1:53` and `[::1]:53`. Install and update set Windows Time to start
automatically and request a sync, because computers outside a domain may never
start it on their own; uninstall leaves it running.

From a normal PowerShell, the bootstrap script asks for administrator rights
through UAC, downloads the bundle for your architecture, verifies it against
`checksums.txt` and runs its installer for the current account. Add
`-User ACCOUNT` to select another account, `-Version vX.Y.Z` to pin a release,
`-Update` to update, or `-Version vX.Y.Z -Archive FILE -Checksums FILE` for an
offline bundle:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Catherine1401/saveme/main/install.ps1)))
```

To install manually instead, download `saveme_<version>_windows_<arch>.zip`
and `checksums.txt` from the release, verify the hash, extract the archive,
then run from an elevated PowerShell in the extracted directory:

```powershell
Get-FileHash .\saveme_<version>_windows_amd64.zip -Algorithm SHA256
powershell -ExecutionPolicy Bypass -File .\deploy\windows\install.ps1 -User <account>
```

The installer copies binaries to `%ProgramFiles%\saveme`, writes
`C:\ProgramData\saveme\config.json` with the account's SID, limits that
directory to SYSTEM, Administrators and read access for the account, registers
the automatic `saved` service and disables DNS-over-HTTPS in Chrome, Edge,
Chromium and Firefox through registry policies. It refuses an existing
installation. It adds `%ProgramFiles%\saveme` to the system `Path`, so new
terminals can run `saveme`; uninstall removes it. If a later step fails, it removes the service, files, rules and
policies it created, so you can run it again. Run `"%ProgramFiles%\saveme\saveme.exe"` as the selected account.

To update, extract the new bundle and run `deploy\windows\update.ps1` from an
elevated PowerShell. It refuses while any domain has a stored lock
and restores the previous binaries and state if the new service does not start.
Service logs are written to `C:\ProgramData\saveme\saved.log`.

While a domain is locked, the service adds a Name Resolution Policy Table rule
sending all DNS queries to its local resolver and Windows Firewall rules in
the `saveme` group that block outbound DNS except to the configured upstreams,
DNS-over-TLS on port 853 and the configured DoH addresses on port 443. Windows
Firewall cannot exempt only the service from a block rule, so an application
that queries the configured upstream resolvers directly on port 53 can bypass
the lock. An Administrator can remove the rules or stop the service.

## Uninstall

Removal is refused while any domain has a stored lock or daily schedule,
including when Linux updates are allowed. Wait for timed locks to expire and
delete daily schedules first. On Windows, a partial installation without a
daemon that can read its configuration has no enforced lock, so uninstall
removes it without that check.

Linux, as your ordinary account:

```sh
curl -fsSL https://raw.githubusercontent.com/Catherine1401/saveme/main/install.sh | bash -s -- --uninstall
```

This runs `deploy/uninstall.sh` from a verified bundle. It stops and disables
`saved.service`, removes the binaries, configuration, state, service unit and
drop-ins, the Chrome, Chromium and Firefox policies and the `saveme-dns`
account. From an extracted bundle you can run `sudo bash deploy/uninstall.sh`.

Windows, from a normal PowerShell:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Catherine1401/saveme/main/install.ps1))) -Uninstall
```

This runs `deploy\windows\uninstall.ps1`, which removes the `saved` service,
the `saveme` firewall rules and name resolution policy, the browser
DNS-over-HTTPS policies, `%ProgramFiles%\saveme` and `C:\ProgramData\saveme`.
Your settings in `saveme/preferences.json` are kept.

## Troubleshooting

Read `journalctl -u saved.service --no-pager -n 50` if the service fails.
The fresh-install unit includes the tested `AmbientCapabilities=CAP_SETUID`
setting for its forwarding child. Updates preserve older units and drop-ins;
see [the startup workaround](setup.md#if-the-service-keeps-restarting).

A fresh install refuses existing target files and the `saveme-dns` account.
It preserves existing browser policy files instead of merging or replacing them.
Use `--update` for an existing complete saveme installation. Preflight checks
reduce partial installs; deployment failures can still leave installed files.
Inspect the error and existing files before retrying. Do not delete saved state.

## Maintainers: release a version

Follow [the release flow](release.md) for source isolation, exact commit approval,
canonical `vX.Y.Z` tags, publication, downloaded-asset verification, and handoff.
Package only the verified committed SHA from a clean checkout. A local build or
successful bootstrap fixture does not prove that GitHub published the release.

Use a new canonical tag for each release. Distribution packages and automatic
self-updates remain separate work; do not advertise channels before they exist.
