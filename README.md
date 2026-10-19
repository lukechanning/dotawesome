# Dotawesome

My workstation configuration: **Niri + Noctalia, Neovim, and tmux**.
Arch/CachyOS on an ASUS G14 and desktop, with KDE available alongside Niri.
Chezmoi manages shared Git, Alacritty, AWS profiles and personal Fish additions.

```sh
cd ~/Code/dotawesome
bin/dotawesome
```

The guided home keeps the ASCII logo and offers three clear tasks: **Set up this
machine**, **Sync this machine**, and **Troubleshoot or restore**.
Setup checks readiness and guides prerequisites, age enrollment/recovery, shared
chezmoi configuration and app setup in order. Sync pulls Git with `--ff-only`,
rebuilds, then reviews managed configuration. Review changes and confirm before
applying. Results stay visible; package upgrades, age and login use the terminal.

Go 1.26+ builds locally into an ignored `.build/` cache; normal runtime has no
Python dependency. Failed compilation stops the invocation. There is no release
service or hosted CI. Linux on Arch/CachyOS is the supported deployment target.

Existing JSON ledgers, backups, age identities and profiles remain usable.
Deployment preserves original files and rejects local drift. Niri behavior
changes require logout; appearance-only changes can use explicit `--live`.

- [Bootstrap & architecture](docs/bootstrap.md) — ownership, enrollment, recovery
- [CLI guide](docs/usage.md) — dashboard, commands, packages and restoration
- [Niri](docs/niri.md) — profiles, live updates, shortcuts
- [Neovim & tmux](docs/applications.md) — setup and daily use
- [Development](docs/development.md) — architecture, checks and migration evidence
- [G14 hardware](laptops/G14/NOTES.md) — optional battery charge limit
