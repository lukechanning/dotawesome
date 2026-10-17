# Dotawesome

My workstation configuration: **Niri + Noctalia, Neovim, and tmux**.
Arch/CachyOS on an ASUS G14, with KDE available alongside Niri.

```sh
bin/dotawesome list
bin/dotawesome status
bin/dotawesome plan nvim
bin/dotawesome apply nvim
bin/dotawesome plan niri --profile g14 --live
```

`plan` previews. `apply` validates and backs up. `remove` restores originals.
Python 3.11+, no pip dependencies. Packages are installed only when requested.

- [CLI guide](docs/usage.md) — commands, packages, restoration
- [Niri](docs/niri.md) — profiles, live updates, shortcuts
- [Neovim & tmux](docs/applications.md) — setup and daily use
- [Development](docs/development.md) — layout and local checks
- [G14 hardware](laptops/G14/NOTES.md) — optional battery charge limit

Retired setups live in [`deprecated/`](deprecated/README.md).
Earlier Niri decisions live in [the historical plan](docs/history/NIRI-PLAN.md).
