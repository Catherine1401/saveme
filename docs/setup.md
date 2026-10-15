# Setup and verification

## Before installing

Choose a [binary or source installation](install.md). Use Linux with systemd,
`timedatectl`, `chattr`, `useradd`, Bash and a filesystem supporting immutable
flags. `make test` also needs a C compiler for Go's race detector.

The daemon needs these local addresses to be free:

| Listener | Purpose |
| --- | --- |
| `127.0.0.1:53` and `[::1]:53`, UDP/TCP | Filtering resolver |
| `127.0.0.1:5353`, UDP/TCP | DNS forwarding child |

IPv6 loopback must be available. Check listeners before installing:

```sh
sudo ss -lntup
```

Review [deploy/config.json](../deploy/config.json) before installation. The
installer sets `allowed_uid` from the account you pass. Defaults are:

- Timezone: `Asia/Ho_Chi_Minh`.
- Upstreams: `1.1.1.1:53` and `8.8.8.8:53`.
- Socket: `/run/saveme.sock`.
- State directory: `/var/lib/saveme`.
- Forwarding account: `saveme-dns`.

Change the template timezone before installing if necessary. Each lock stores
its timezone; changing the configuration later does not move existing schedules.
Upstreams must be non-loopback IP addresses on port 53, not hostnames.

## Install and check

Pass the ordinary Linux account that will manage locks, not root:

```sh
sudo bash deploy/install.sh "$USER"
systemctl status saved.service --no-pager
timedatectl show --property=NTPSynchronized --value
saveme
```

The service should remain `active (running)`. NTP must report `yes` before
production changes are accepted. The socket belongs to the selected account;
other accounts cannot manage the daemon.

The fresh installer refuses existing target files or the `saveme-dns` account. It
does not merge policies or roll back a partial install. Use the separate
[update flow](install.md#update) for an existing installation.
Review conflicts before retrying; do not delete saved state to resolve them.

It installs:

| Path | Contents |
| --- | --- |
| `/usr/local/bin/saveme` | Ordinary-user TUI |
| `/usr/local/libexec/saved` | Root daemon and forwarding child |
| `/etc/saveme/config.json` | Configuration |
| `/etc/systemd/system/saved.service` | Immutable service unit |
| `/var/lib/saveme/` | Two checksummed, immutable state copies |
| `/etc/opt/chrome/policies/managed/saveme.json` | Chrome DoH policy |
| `/etc/chromium/policies/managed/saveme.json` | Chromium DoH policy |
| `/etc/firefox/policies/policies.json` | Firefox DoH policy |

Restart the browser and inspect `about:policies` in Firefox or `chrome://policy`
in Chrome/Chromium. Firefox should show DNSOverHTTPS disabled and locked;
Chrome/Chromium should show DnsOverHttpsMode `off`. These policies also apply
outside lock periods. Snap/Flatpak installations need separate verification.

## If the service keeps restarting

Read the actual error:

```sh
journalctl -u saved.service --no-pager -n 50
```

On the tested Arch host with systemd 262, the daemon started but failed to launch
its forwarding child with `fork/exec /usr/local/libexec/saved: operation not
permitted`. Keeping `CAP_SETUID` ambient resolved that specific failure. If logs
show this same problem, add the tested drop-in:

```sh
sudo install -d /etc/systemd/system/saved.service.d
sudo tee /etc/systemd/system/saved.service.d/cap.conf >/dev/null <<'UNIT'
[Service]
AmbientCapabilities=CAP_SETUID
UNIT
sudo systemctl daemon-reload
sudo systemctl restart saved.service
systemctl status saved.service --no-pager
```

Fresh installations include this setting in the service template. Preserve
an existing drop-in when updating binaries. Diagnose other startup errors from
the logs; this setting does not fix port conflicts or invalid configuration.

## Try the interface safely

Run `./bin/saved -demo` and `./bin/saveme -demo` in separate terminals as an
ordinary account. Demo mode skips network enforcement, NTP requirements and
immutable flags. Its socket and state live under `/tmp/saveme-demo-<uid>`.
To reset the demo, stop that demo daemon and remove its own directory. The
production service never enables demo mode.

For a first walkthrough:

1. On the domain list, press `a`, add `https://www.Example.com/a`, and confirm
   the saved domain is `example.com`. Add `example.org` the same way.
2. Open `example.com` with Enter, choose Timed lock, and select 12 hours.
3. Type the four-character code slowly and press Enter. Pasting the code should
   be rejected. The list should show `example.com` locked with its remaining
   time and `example.org` unlocked; `e` and `d` are refused only on
   `example.com`.
4. Open `example.com` with Enter to see its countdown, then press `q` to return.
5. Quit and reopen the demo TUI. The lock should still be active.

Demo mode proves the interface flow, not network blocking.

The README demo is recorded with [VHS](https://github.com/charmbracelet/vhs):
run `make build && vhs docs/assets/demo.tape` with `vhs`, `ttyd` and ffmpeg on
the `PATH`. The tape starts the demo daemon with `SAVEME_DEMO_CODE=K7MQ`, which
makes every confirmation code `K7MQ` so the recording can type it. Only
`saved -demo` reads this variable; the protected service ignores it.

### Windows demo

For a protected installation, see [Windows](install.md#windows). To try the
interface without an installation, build with Go on Windows, then run the
daemon and client in separate terminals as the same non-elevated account:

```powershell
go build -o bin\saved.exe .\cmd\saved
go build -o bin\saveme.exe .\cmd\saveme
.\bin\saved.exe -demo
.\bin\saveme.exe -demo
```

The demo daemon listens on a named pipe, `\\.\pipe\saveme-demo-<SID>`, that
only SYSTEM and your account can open. Its state lives under
`%TEMP%\saveme-demo-<SID>`; stop the daemon and remove that directory to reset.

## Use

The main screen lists every domain with its state: locked with the remaining
time, the next daily lock, or unlocked.

1. Press `a`, enter a domain or URL, then press Enter.
2. Select a domain and press Enter. Choose Daily schedule (`1`) or Timed lock (`2`).
3. Set the hours or select a duration. Type the displayed code one character at
   a time, then press Enter. Paste is rejected.

Each domain has its own lock. Locked domains are listed first, then domains
with a schedule. Use arrow keys to select, Enter to open or save, and Esc to go
back. On the domain list, `a` adds, `e` edits, `d` deletes, `/` filters the list
and `?` shows every key. Adding is always allowed; editing or deleting a locked
domain is refused. Deleting always asks for confirmation; a stored schedule is
removed with its domain. Press space to mark several unlocked domains without a
schedule, then open one of them: the daily schedule or timed lock you confirm
applies to every marked domain. In a daily window, left/right selects a field,
up/down adjusts it and `a` adds another window before confirmation. A new window
starts from a choice of preset hours (night, work hours, late night, lunch or
custom); the editor then shows the window on a 24-hour bar with its length and
whether it crosses midnight. In the saved daily schedule, each window shows when it next
starts or ends; select one and press `e` to edit its hours. Press Enter, review
the schedule, and type the confirmation code to save. In the duration list, `0`
selects the tenth choice. Action errors disappear after a few seconds. When the
clock is not synchronized, a warning explains that locks stay on and only adding
domains works.

Opening a locked domain shows its countdown; `q` returns to the list and the
daemon keeps enforcing the lock. Other domains remain editable. Timed choices
are 1, 2, 4, 6 or 12 hours, or 1, 3, 7, 14 or 30 days.

## Limits

Root can bypass the lock. Cached addresses, existing connections, VPNs and direct
IP access can bypass DNS filtering. Browser DoH policies and a fixed resolver IP
list reduce bypasses; they cannot guarantee that every app uses filtered DNS.
The terminal cannot prevent screenshots or synthetic keystrokes.

Facebook is an exception: `facebook.com` blocks only itself and
`www.facebook.com`. Other subdomains remain accessible unless separately listed;
this does not distinguish feeds from messages by URL. Clock and storage limits
are described below.

## Lock behavior

Daily windows include their start and exclude their end. They can cross midnight,
can touch but cannot overlap. A schedule covering the entire day has no automatic
unlock time. A timed lock starts when confirmation succeeds. Each domain holds
at most one lock; delete a domain's daily schedule before giving it a timed
lock. Only domains whose lock is active are blocked. While the clock is not
synchronized with NTP, every stored lock stays active and only adding domains
is accepted. Updating the binaries is refused while any domain has a stored
lock. State from earlier versions, which shared one lock across all domains, is
converted on load by giving each domain a copy of that lock.

Domains are lowercased, converted through IDNA and stripped of a leading `www.`.
The `facebook.com` entry blocks only the apex and `www.facebook.com`; separately
listed subdomains still block themselves and their descendants. A user reported
Facebook blocked and Messenger working on the installed host. That observation
is not a guarantee for every Facebook/Messenger entry point.

Confirmation codes last 60 seconds. A wrong attempt replaces the code; three
wrong attempts cancel the request. Requests belong to their socket connection.
Authorized clients can read and submit the code automatically, so it is a pause
in the TUI flow rather than a secret protecting against the account owner.

State is saved before a new lock's rules are applied. If enforcement fails, the
saved lock remains and the daemon retries. Two checksummed copies support
recovery after one copy is damaged. A storage failure stops mutations until
restart. If both copies are invalid, startup attempts to clear saveme's firewall
table and exits with an error. Inspect logs instead of editing state files.

Effective time cannot go backward while running and uses a persisted `MaxSeen`
floor after restart. It is saved every minute. Physical power loss and a crash
at every point during a write have not been verified.

## Development checks

```sh
make test check build
```

Ordinary tests cover core rules, storage recovery, socket ownership, UI states,
DNS UDP/TCP filtering and forwarding, CNAME blocking and release. nftables and
immutable tests require an isolated network namespace or a filesystem with
`CAP_LINUX_IMMUTABLE`; they skip in ordinary runs.

Separate checks passed in a Debian 12 VM: installation, restart, rule recovery,
NTP gating, immutable storage, reboot during a lock, abrupt VM termination after
durable state, and clock changes. Firefox ESR and Chromium policy activation
were checked in separate containers. This does not verify GUI browsing in the
same enforcement VM, Google Chrome, Snap/Flatpak, ARM or Go 1.24 specifically.

See [the protocol](protocol.md) for direct socket requests. The TUI runs as an
ordinary account; `saved` owns state and enforces mutation guards. Its child
forwards DNS as a separate non-root account. nftables redirects DNS during locks,
blocks encrypted DNS on port 853, protects the forwarding port and blocks the
configured DoH IPs on TCP/UDP 443. The DoH list is finite.

Implementation references: [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[miekg/dns](https://github.com/miekg/dns),
[google/nftables](https://github.com/google/nftables),
[Firefox DNSOverHTTPS policy](https://mozilla.github.io/policy-templates/#dnsoverhttps),
[Chrome DnsOverHttpsMode policy](https://chromeenterprise.google/policies/#DnsOverHttpsMode),
[Cloudflare DNS addresses](https://developers.cloudflare.com/1.1.1.1/ip-addresses/),
[Google Public DNS addresses](https://developers.google.com/speed/public-dns/docs/using).
Dependencies are pinned in `go.mod` and `go.sum`; these links do not imply that
the project uses the latest major versions.

## Terminal appearance

The TUI fills the terminal: a heading with the domain and lock counts at the
top, the current screen below it, and messages and one line of key hints
separated by `·` pinned to the bottom. The domain list aligns each domain's name, status and remaining time in columns; long domains are shortened with `…`, and long lists
scroll with `↑ N more` / `↓ N more` markers. Content wider than
120 columns is centered. Delete and confirmation prompts appear in a centered
rounded box. `❯` marks the selected row, and errors appear after `⎿`. Colors follow
Catppuccin: Mocha on dark terminals and Latte on light ones, with mauve for
selection, green for confirmation and overlay grey for hints. Press `s` on the
domain list to open Settings, where you choose the theme, the countdown clock
style, the clock effects and whether the clock warns during the final minute.
Moving through a list previews the choice, Enter saves it to
`saveme/preferences.json` in your user configuration directory (`~/.config` on
Linux, `%AppData%` on Windows) and Esc keeps the saved one. Run `saveme --theme
NAME`, `--clock NAME` or `--clock-effects LIST` (comma-separated, or `none`) to change
one for a session.
Clock styles: half (default), block, rounded, classic, double, shade, dots, lcd,
ascii and text. Effects can be combined; space toggles each one: blink (on by
default), slide, flip, tick, fill (progress shown inside the digits), gradient,
rainbow, shimmer, breathe, scanline, glitch, neon, sparkle and rain;
`--no-motion` stops every moving effect and the warning pulse, keeping fill,
gradient, a still rainbow and the neon outline. Themes:
catppuccin, mint, dracula, gruvbox, solarized, tokyonight, rosepine, everforest,
kanagawa, nightfox, flexoki, selenized, modus, ayu, nord and ansi (your
terminal's own 16 colors).
Loading indicators animate;
confirmation feedback is brief. Animation never advances the lock clock or
increases daemon polling. Your terminal font and color scheme stay unchanged.

```sh
saveme --no-motion
saveme --no-color --no-motion
```

`NO_COLOR` also disables colors. To preview colors when that variable is set:

```sh
env -u NO_COLOR saveme
```

Narrow terminals wrap text and show the confirmation code as ordinary text when
the four character cells cannot fit. Confirmation and paste restrictions still
apply. For a local preview without network enforcement, run `./bin/saved -demo`
and `./bin/saveme -demo` in separate terminals.

Vim navigation is always enabled: `j`/`k` move through lists, `gg`/`G` jump
to the first/last item, and `h`/`l` select time fields. In time fields, `k`
increases and `j` decreases the value. Arrow keys remain available. Domain
text and confirmation codes use normal text input. Outside input screens, `h`
returns one level and `l` opens the selected item (or edits a saved window).

The active-lock countdown shows the lock type (timed lock with its length, or
the running daily window), the remaining days above large seven-segment digits,
a progress bar with the elapsed share and the local unlock time. Without an
unlock time it explains why: an unsynchronized clock or a schedule covering the
whole day. Small terminals use ordinary text instead of large digits. Confirmation displays the four
real characters in separate cells, using your existing terminal font.

Outside input screens, `q` returns one level. From the domain list, it exits
the TUI; from a locked domain's countdown, it returns to the list. When entering a domain, time or confirmation code,
`q` does not navigate or quit. Use Esc to cancel input.
