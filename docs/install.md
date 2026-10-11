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
build. macOS, Windows and systems without systemd are not supported.

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

```sh
make test check
bash scripts/release.sh v0.1.0
bash scripts/install_test.sh dist/v0.1.0
cd dist/v0.1.0
sha256sum --check checksums.txt
```

Each archive includes both static Go binaries, deployment scripts, service,
configuration and policy templates, MIT license and `VERSION`. The release
script does not publish anything. Bootstrap tests use real bundles with curl
and sudo fixtures; they do not deploy to the machine running those tests.

After approving and committing a version, create and push its stable `vX.Y.Z`
tag. The Release workflow runs tests and vet, checks Bash syntax, builds both
architectures and publishes the archives with `checksums.txt`. Use a new tag for
each release. Package-manager channels are not advertised until packages actually
exist. Distribution packages and automatic self-updates are separate future work.
