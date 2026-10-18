# Dotawesome

My workstation configuration: **Niri + Noctalia, Neovim, and tmux**.
Arch/CachyOS on an ASUS G14 and desktop, with KDE available alongside Niri.
Chezmoi manages shared Git, Alacritty, AWS profiles and personal Fish additions.

```sh
cd ~/Code/dotawesome
bin/dotawesome
```

The guided CLI inspects this machine, remembers your profile and selected
components, offers missing packages, handles age enrollment or recovery, and
previews configuration changes before applying them with backups. Run the same
command for first setup, updates, inspecting differences or component restoration.
Niri changes that need logout are deferred while other work can continue.

Python 3.11+, no pip dependencies. Package installation and full system upgrades
are explained and confirmed in the flow. Explicit `status`, `plan`, `check`,
`apply` and `remove` commands remain available for scripts and targeted work.

- [Bootstrap & architecture](docs/bootstrap.md) — ownership, age enrollment, new machines, updates and recovery
- [CLI guide](docs/usage.md) — commands, packages, restoration
- [Niri](docs/niri.md) — profiles, live updates, shortcuts
- [Neovim & tmux](docs/applications.md) — setup and daily use
- [Development](docs/development.md) — layout and local checks
- [G14 hardware](laptops/G14/NOTES.md) — optional battery charge limit

Retired setups live in [`deprecated/`](deprecated/README.md).
Earlier Niri decisions live in [the historical plan](docs/history/NIRI-PLAN.md).
