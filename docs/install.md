# Install

The installer downloads a release bundle, checks its SHA-256 and uses sudo
for system deployment. You do not need Git, Go or Make for a binary install.
Binary installation requires a published release with bundles and checksums.
If none exists yet, use [the source build](#build-from-source).

## Requirements

- Linux with a running systemd and enabled IPv6 loopback.
- An ordinary account with sudo access.
- Bash, curl, tar, sha256sum, awk and coreutils.
- systemctl, timedatectl, chattr, useradd, runuser and ss.
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

Updates refuse any stored timed lock or daily schedule, including daily schedules
outside their active window. Wait for a timed lock to expire; remove a daily
schedule through the TUI outside its active window. The updater checks stored
state before stopping the daemon and again after it stops. A lock created between
those checks causes rejection and a restart with the previous binaries.

The updater saves the old binaries and restores them if the replacement fails
its readiness check. State is never reset. A new daemon can make normal state
writes during startup, so preserving state does not mean byte-identical files.
Updates restart the daemon; use them only when no commitment is configured.

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

Run `make test check` before deploying local changes. Source installation uses
the same service checks and conflict refusal as binary installation. To update
from a built checkout, run `sudo bash deploy/update.sh "$USER"`.

## Windows

Windows support is experimental and has not been verified on a real Windows
installation yet. Requirements: Windows 10/11 or Server with PowerShell 5.1,
an Administrator account to install, an ordinary account to manage locks, the
Windows Time service synchronized with a network source, and free
`127.0.0.1:53` and `[::1]:53`.

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
installation. Run `"%ProgramFiles%\saveme\saveme.exe"` as the selected account.

To update, extract the new bundle and run `deploy\windows\update.ps1` from an
elevated PowerShell. Like Linux, it refuses while any domain has a stored lock
and restores the previous binaries and state if the new service does not start.
Service logs are written to `C:\ProgramData\saveme\saved.log`.

While a domain is locked, the service adds a Name Resolution Policy Table rule
sending all DNS queries to its local resolver and Windows Firewall rules in
the `saveme` group that block outbound DNS except to the configured upstreams,
DNS-over-TLS on port 853 and the configured DoH addresses on port 443. Windows
Firewall cannot exempt only the service from a block rule, so an application
that queries the configured upstream resolvers directly on port 53 can bypass
the lock. An Administrator can remove the rules or stop the service.

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
