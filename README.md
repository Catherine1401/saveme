<p align="center">
  <img src="docs/assets/banner.svg" width="720" alt="saveme — a website blocker for Linux and Windows terminals" />
</p>

<p align="center">
  <a href="docs/assets/demo.mp4"><img src="docs/assets/demo.gif" width="720" alt="Full demo: domains, daily schedule, 12-hour lock and reopening the client." /></a>
</p>

## Install

Pick the block for your system.

### Linux (bash)

Requires systemd and sudo. Run as your ordinary account:

```sh
curl -fsSL https://raw.githubusercontent.com/Catherine1401/saveme/main/install.sh | bash
saveme
```

### Windows (PowerShell, experimental)

Paste into a normal PowerShell and accept the administrator prompt. The current
account becomes the one that manages locks:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Catherine1401/saveme/main/install.ps1)))
& "$env:ProgramFiles\saveme\saveme.exe"
```

Windows support is experimental: the one-line install needs a release that
includes the Windows bundle, and blocking has not yet been verified on a real
Windows installation.

To remove saveme, run the same command with `--uninstall` (Linux) or
`-Uninstall` (Windows). Like updates, removal is refused while any domain has a
lock or daily schedule.

[Install options](docs/install.md) · [Usage](docs/setup.md#use) · [Releases](https://github.com/Catherine1401/saveme/releases) · [Limits](docs/setup.md#limits)

## Features

- A separate daily schedule or 1h–30d timed lock for each domain, or one lock
  for several marked domains at once.
- A locked domain and its schedule cannot be edited or deleted; other domains stay editable.
- A countdown with the lock type, progress, unlock time and a large clock.
- Settings for 16 color themes, 10 clock styles and 14 combinable clock effects.
- Locks survive closing the terminal and daemon restarts.
- Linux (systemd) and Windows (experimental).

## Keys

| Key | Action |
| --- | --- |
| `a` | Add a domain |
| `↵` | Open a domain: lock it or see its countdown |
| `/` | Filter the list |
| `space` | Mark domains to lock together |
| `s` | Settings: theme, clock style and effects |
| `?` | All keys |
| `q` | Back, or quit from the list |

## License

[MIT](LICENSE) © 2026 saveme contributors.
