# CLI

Run `bin/dotawesome` from the checkout, or use its absolute path elsewhere.
Supported components: `niri`, `nvim`, `tmux`. One component per operation.

| Command | Effect |
| --- | --- |
| `list` | Show supported components |
| `status [COMPONENT]` | Show inactive, current, source-changed, or locally-modified state |
| `check COMPONENT` | Check dependencies and validate configuration |
| `plan COMPONENT` | Show changes and diffs; write nothing |
| `plan COMPONENT --remove` | Preview original-file restoration and eligible package cleanup |
| `apply COMPONENT` | Validate, snapshot, back up, and deploy |
| `remove COMPONENT` | Restore originals; retain packages and application data |

Niri check/plan/apply require `--profile generic`, `--profile g14`, or a TOML path.
Application components have no profiles. Old `build`, `activate`, and `deactivate`
commands have been removed.

```sh
bin/dotawesome check tmux
bin/dotawesome plan tmux
bin/dotawesome apply tmux
bin/dotawesome plan tmux --remove
bin/dotawesome remove tmux
```

Plans show missing prerequisites but still return a preview. Check/apply fail on
missing prerequisites or invalid configuration. Plans do not run native validators;
checks use temporary directories, without downloading plugins or tools.

## Packages

Use `apply COMPONENT --install-packages` to opt into interactive
`sudo pacman -Syu --needed`. Otherwise install dependencies yourself.
`remove COMPONENT --remove-packages` offers cleanup of eligible recorded additions,
after restoring configuration. Pre-existing and other components' required
packages are protected; the system upgrade itself is never undone.
`plan --remove` previews cleanup eligibility, while ordinary `remove` keeps packages.

## Ownership and recovery

Files are copied, with original bytes or symlinks saved on first apply. Later
applies preserve that backup. Unchanged files are not rewritten. Local edits to
managed files block apply/remove: save those edits and restore the recorded bytes
before retrying. No force-overwrite option is provided.

Private ledgers and the recovery journal live under `$XDG_STATE_HOME/dotawesome`
(default `~/.local/state/dotawesome`); snapshots live under `.build/dotawesome/`.
Keep them for restoration. Interrupted file transactions roll back on the next
mutation; unexpected later edits stop recovery. Interrupted package operations
require inspection of the package ledger and pacman log before retrying.

Niri's existing ledgers and backups remain compatible. Application transactions
never reload systemd or restart running applications. Recovery of an interrupted
Niri transaction still requires leaving the Niri session.
