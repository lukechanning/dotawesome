# Development

Python 3.11+, standard library only. No package build or hosted CI.

| Location | Purpose |
| --- | --- |
| `bin/dotawesome` | CLI entry point |
| `lib/dotawesome/` | Commands, explicit component registry, deployment, packages |
| `components/` | Maintained Niri, Neovim, and tmux sources |
| `laptops/G14/` | Separate machine-level hardware policy |
| `tests/` | Local deployment/behavior checks |
| `deprecated/` | Retired configurations; excluded from checks |
| `.build/` | Ignored snapshots and disposable previews |

Run local checks with `bin/ci`. The name is historical; nothing runs remotely.
Checks cover file ownership, backup/restoration, interrupted transactions, package
provenance, CLI behavior, and available native validators. System/package changes
use mocks and private temporary homes. Native checks skip unavailable tools;
tmux needs permission to create a private Unix socket.

```sh
bin/ci
bin/dotawesome check nvim
bin/dotawesome check tmux
bin/dotawesome check niri --profile g14
```

Neovim validation loads core settings and parses all Lua files without plugins or
network access. Plugin integration requires a separate disposable editor with
private HOME and XDG config/data/state/cache directories. Smoke-test startup,
file-tree opening, language buffers, and formatter availability before changing
pins. Never use your live editor profile for experimental validation.

Register new components explicitly; keep component-specific policies out of the
shared file engine. Preserve existing ledger/backups and add focused checks when
changing file writes, restoration, or recovery. Documentation history is reference,
not the current operating guide.
