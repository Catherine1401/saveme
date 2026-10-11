<p align="center">
  <img src="docs/assets/banner.svg" width="720" alt="saveme" />
</p>

<p align="center"><strong>Block distractions. Keep your promise.</strong><br />
A website blocker for your Linux terminal.</p>

<p align="center"><code>Linux</code> <code>Go</code> <a href="LICENSE"><code>MIT</code></a></p>

<p align="center">
  <a href="docs/install.md">Install</a> ·
  <a href="docs/setup.md#use">Usage</a> ·
  <a href="docs/setup.md#try-the-interface-safely">Demo</a> ·
  <a href="docs/protocol.md">Protocol</a> ·
  <a href="https://github.com/Catherine1401/saveme/issues">Issues</a>
</p>

<p align="center">
  <img src="docs/assets/demo.gif" width="720" alt="Demo mode: add a domain, select 12 hours and enter the confirmation code. No network blocking." />
  <br /><sub>Demo mode · no network blocking</sub>
</p>

---

**Daily schedules · 12h–30d locks · Restart recovery**

### Install

Requires systemd and sudo. [Options and requirements](docs/install.md).

```sh
curl -fsSL https://raw.githubusercontent.com/Catherine1401/saveme/main/install.sh | bash
saveme
```

**Add domains → choose a schedule → type the code.**
Closing the terminal keeps the lock running.

Root and DNS bypasses remain possible. See [limits](docs/setup.md#limits).

---

[MIT](LICENSE) © 2026 saveme contributors.
