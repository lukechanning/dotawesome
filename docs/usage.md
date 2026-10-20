# CLI

Run `bin/dotawesome` from the checkout, or use its absolute path elsewhere.
Go 1.26+ builds the application locally when sources or module files change.
No arguments, or `bin/dotawesome setup`, opens the action menu:

| Action | Terminal command | Outcome |
| --- | --- | --- |
| Syncthing Setup | `bin/dotawesome syncthing setup` | Prepare and pair Documents over Tailscale |
| Syncthing Status | `bin/dotawesome syncthing status` | Check connectivity and sync progress |
| Config Setup | `bin/dotawesome config setup` | Set up shared Git, Fish, Alacritty and AWS configuration |
| Config Sync | `bin/dotawesome config sync` | Pull updates, rebuild, review and apply configuration |
| App Setup | `bin/dotawesome app setup [niri\|nvim\|tmux]` | Select applications or review one directly |
| Troubleshoot or Restore | Existing diagnostic commands below | Inspect, recover or restore |

**Full Machine Setup** remains under Troubleshoot or Restore. It checks readiness,
installs missing prerequisites with confirmation, selects the machine, enrolls or
recovers age, reviews shared chezmoi configuration, then selects apps. App Setup
checks only the selected app's prerequisites and does not require shared enrollment.
[Documents setup](syncthing.md) explains pairing, conflicts, versions and disabling
transfers. Config Sync never starts or waits for Documents transfers.

Use arrows or Tab/Shift-Tab to choose an item and Enter to open it. Escape goes
back one screen; `q` quits navigation. `?` opens contextual help. Narrow terminals
wrap explanations and keep the selected action visible. `NO_COLOR=1` removes
color. No special fonts are required.

The home screen summarizes saved configuration, without raw status codes or file
paths. An interrupted update recommends Troubleshoot or restore. Component
labels describe configuration files, not whether the applications are installed
or running:

| Label | Meaning |
| --- | --- |
| Not managed yet | Dotawesome has not changed these files; the app may already exist |
| Configuration matches | Managed files match the repository |
| Repository changes available | Repository configuration differs and can be reviewed |
| Local edits found | Files differ from the deployed version; updates pause to protect edits |
| Unable to check | The check is incomplete or source is unavailable; inspect technical details |

For local edits, **Inspect differences** compares the saved deployed version with
your current files. This read-only action remains available when deployment is
blocked by drift. Other components compare your files with repository settings.
File paths, package records, and diagnostics stay in technical details.

During selection, Space toggles components and `p` cycles the Niri profile.
Existing custom profile paths are retained. Review opens directly on the selected
diff: left/right changes the file; up/down or Page Up/Down scrolls. `f` opens the
file list, `c` explains prerequisite checks, and `p` opens confirmed package setup
in the terminal. Missing prerequisites lead to their explanations before you can
confirm deployment. Reinspect after package operations.

During execution, Escape or Ctrl-C requests a safe stop and waits for rollback.
Completed, skipped, deferred and failed results remain visible. Enter returns
home; `r` retries through fresh inspection. Backing out of confirmation returns
to the reviewed changes.

Setup explains the next missing step and checks again after terminal commands.
Existing machine selections and identities are reused. Declined, deferred and
failed steps remain visible; they do not count as completed setup. Shared setup
shows its diff in the terminal and requires confirmation. Age owns passphrase
prompts. Optional GitHub and AWS login checks are available after setup; AWS uses
the local `cof-aws` account.

Sync requires a clean checkout on a branch with an upstream. It uses
`git pull --ff-only`; local changes and divergent history stop the flow with an
explanation. It never stashes, commits or pushes. The launcher rebuilds after the
pull before configuration inspection; a failed build stops the flow. Without
shared enrollment, sync updates managed apps without requiring new enrollment.

| Command | Effect |
| --- | --- |
| `bin/dotawesome sync` | Pull latest changes, relaunch, then review managed configuration |
| `bin/dotawesome list` | List Niri, Neovim and tmux |
| `bin/dotawesome status [COMPONENT]` | Inspect inactivity, source changes, local drift and backups |
| `bin/dotawesome check COMPONENT` | Inspect dependencies and run native validation |
| `bin/dotawesome plan COMPONENT` | Preview file changes without deploying |
| `bin/dotawesome plan COMPONENT --remove` | Preview original-file restoration and package eligibility |
| `bin/dotawesome apply COMPONENT` | Review, confirm, validate, snapshot and deploy |
| `bin/dotawesome remove COMPONENT` | Confirm and restore originals; keep application data |
| `bin/dotawesome init [--machine g14\|desktop] [--skip-hooks]` | Enroll shared chezmoi configuration and recover identity |
| `bin/dotawesome enroll-age` | Confirm, generate identity, encrypt and verify recovery before publication |
| `bin/dotawesome check-secrets [--staged]` | Redacted Gitleaks scan of working content or index |
| `bin/dotawesome recover` | Confirm and recover an interrupted transaction |

Niri check/plan/apply require `--profile generic`, `--profile g14`, `--profile desktop`,
or a TOML path. Applications do not use profiles. Flags can follow the component.

```sh
bin/dotawesome plan tmux
bin/dotawesome apply tmux
bin/dotawesome plan niri --profile g14
bin/dotawesome apply niri --profile g14 --live
bin/dotawesome plan tmux --remove
bin/dotawesome remove tmux
```

Plans report missing prerequisites while still showing the preview. Check/apply
fail on missing prerequisites or native validation errors. Native validators stage
files in temporary directories, without plugin downloads. Exit codes are 0 for
success, 1 for operational failure or declined confirmation, 2 for invalid command
syntax, and 130 for an interrupted explicit operation.

Use `apply COMPONENT --install-packages` to opt into confirmed, interactive
`sudo pacman -Syu --needed` (a full system upgrade). Configuration is freshly
inspected and confirmed afterwards. `remove COMPONENT --remove-packages` offers
confirmed cleanup after restoring files. Pre-existing packages and other
components' recorded requirements are protected. Upgrades are never undone.
Interrupted package operations require inspecting `packages.json` and pacman logs.

Original bytes, permissions and symlinks are saved on first deployment and retained
across updates. Local edits block apply/remove: preserve them and restore the
recorded installed version before retrying. There is no force-overwrite option.
Plans are checked again under the mutation lock; changed inputs require renewed
review. Atomic file writes, journals and rollback protect partial failures.

Ledgers and original backups remain under `$XDG_STATE_HOME/dotawesome` (default
`~/.local/state/dotawesome`); snapshots remain under `.build/dotawesome/`. Keep the
ledgers for restoration. Existing Python-era JSON is supported without reset.
Unexpected edits after interruption stop recovery. Interrupted Niri recovery
requires leaving the session. Regular application changes do not reload systemd.
Appearance-only Niri live updates preserve behavior, ownership and local edits;
helper migration and other behavior changes must be applied outside Niri.
