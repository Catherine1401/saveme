<p align="center">
  <img src="docs/assets/banner.svg" width="720" alt="saveme — a website blocker for your Linux terminal" />
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

[Install options](docs/install.md) · [Usage](docs/setup.md#use) · [Releases](https://github.com/Catherine1401/saveme/releases) · [Limits](docs/setup.md#limits)

## Features

- A separate daily schedule or 1h–30d timed lock for each domain.
- A locked domain and its schedule cannot be edited or deleted; other domains stay editable.
- Locks survive closing the terminal and daemon restarts.

## License

[MIT](LICENSE) © 2026 saveme contributors.
