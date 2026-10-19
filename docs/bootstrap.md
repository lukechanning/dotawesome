# Shared configuration and machine bootstrap

Keep one Git checkout at `~/Code/dotawesome`. Both machines work without an
external SSD. Code remains Git-managed; bulk migration and Syncthing are separate.

## Setup and updates

Install Git and Go 1.26+, clone this checkout, then run `bin/dotawesome`.
Choose **Set up this machine**. The guided flow checks the rig and explains the
next missing step:

1. Confirm installation of missing shared prerequisites, if needed.
2. Choose G14 or desktop; existing machine selections are reused.
3. Enroll an age identity or recover the existing encrypted identity.
4. Review and confirm shared chezmoi configuration and hook setup.
5. Select app configurations, review their changes, and confirm deployment.

Ready steps are skipped. Existing identities and configuration are inspected
before changes; incomplete or conflicting state gets an explanation. Optional
GitHub and AWS login checks are offered at the end. **Troubleshoot or restore → Inspect app configuration** opens
app setup independently; **Troubleshoot or restore → Shared configuration** runs shared-only
setup. Standalone helpers remain available below.

Keep the recovery passphrase separately in 1Password or an offline record.
Existing/partial enrollment is preserved on failure. Review and commit only the
public recipient and encrypted recovery copy under `encryption/` before setting
up another machine. Never commit the plaintext identity.

The app configuration stage selects Niri, Neovim and tmux independently. Preflight checks
ownership and dependencies; review shows file diffs. Press `p` on review for a
confirmed package upgrade in the terminal, then inspect again. Niri changes
requiring logout are deferred while other components can complete. Results stay
open. Escape during execution requests a safe stop; rollback completes before
returning. Only the explicit **Sync this machine** flow pulls Git; no flow commits, pushes or stashes.

Profile and component selections are remembered under
`$XDG_STATE_HOME/dotawesome/preferences.json`. Existing chezmoi machine data and
custom Niri profile paths are reused. Component local edits block replacement;
preserve them and reconcile the recorded version before retrying. Existing
`local-edits/*/files.json` backups from earlier versions remain ordinary private
JSON file records and can be restored manually.

On another machine, clone the same checkout and choose **Set up this machine**.
Recovery uses the encrypted identity and its separate passphrase. For updates,
choose **Sync this machine**: it requires a clean branch with an upstream,
pulls with `--ff-only`, then rebuilds and resumes review using the updated code.
Divergent history or local changes stop before configuration application. Shared
configuration and managed apps are inspected again before confirmation.
The home check remains read-only. Restore previews and restores original
component files; application data is retained. Shared adoption backups are
separate and restored manually.

See [CLI usage](usage.md) for keyboard controls and explicit commands. `bin/init`,
`bin/enroll-age` and `bin/check-secrets` are thin wrappers over the same Go services.

## Ownership and review boundaries

| Owner | Destinations |
| --- | --- |
| chezmoi (`home/`, selected by `.chezmoiroot`) | `~/.gitconfig`, `~/.config/alacritty/alacritty.toml`, `~/.aws/config`, personal Fish `conf.d` and `functions` files; explicitly added encrypted credentials |
| `bin/dotawesome` | Niri/Noctalia configuration and session units, Neovim, `~/.tmux.conf`; existing ledger, validation, backups and restoration |
| CachyOS | Fish `config.fish`, prompt and packaged defaults; system packages and session entries |
| Local machine | age identity, chezmoi machine selection, Fish history/variables, AWS/GitHub authentication caches, VPN profiles/credentials, app state and wallpapers |

Do not add component destinations or parent directories to chezmoi. Bootstrap
checks the actual managed file/symlink list against the component boundary before
preview or apply. Tests also exercise overlap rejection. Use standard CachyOS
XDG configuration (`$HOME/.config`); data/state/cache can be isolated separately.

Review the implementation in four groups: ownership and profiles; shared sources;
bootstrap/enrollment and recovery tests; secret hooks and documentation. No live
configuration is adopted by the implementation itself. Prove isolated deployment
before doing the G14 rehearsal and adoption below.

The source subdirectory follows [chezmoi's supported layout](https://www.chezmoi.io/user-guide/advanced/customize-your-source-directory/).

## Advanced: install prerequisites explicitly

On each CachyOS machine:

```sh
sudo pacman -S --needed git go chezmoi age fish alacritty github-cli aws-cli-v2 gitleaks
cd ~/Code/dotawesome
```

The standalone `bin/init` never installs packages. The guided CLI offers packages
interactively; explicit component commands retain `--install-packages`; see [usage](usage.md). CI requires chezmoi, age, Fish and Gitleaks;
available native component validators run too. Gitleaks 8.30+ supports the modes
used here. `bin/init` installs the pre-commit hook and refuses an existing hook or `core.hooksPath`;
chain `hooks/pre-commit` from your current hook manually in that case, then use
`bin/init --skip-hooks` to retain your hook setup. This opt-out does not replace or
modify existing hooks.

## Advanced: enroll separately, once

Enrollment happens in your local terminal. Guided setup offers it when the
repository has no identity. If using the standalone helpers instead, enroll
before running `bin/init`; it reports missing enrollment until both files exist. No static AWS credentials are included in this repository.

```sh
bin/enroll-age
```

This creates one dedicated X25519 key, asks age for a passphrase through the
terminal, then asks again to verify decryption. Store the passphrase separately
in 1Password or an offline recovery record. It never enters command arguments or
an environment variable. Only after verification does enrollment publish:

- `~/.config/chezmoi/key.txt`: local private identity, mode `0600`.
- `encryption/recipient.txt`: public age recipient; commit it.
- `encryption/identity.age`: passphrase-encrypted recovery copy; commit it.

Keep a separate backup of the encrypted recovery copy. Enrollment refuses to
overwrite any existing key or enrollment file. If interrupted during publication,
inspect the partial files before retrying; do not generate a new key over an
existing enrollment. Temporary plaintext stays in a private local directory and
is removed on ordinary interruption. After a hard kill, remove only the stale
`.enroll-*` directory after checking that the published identity is valid.

This implements the [documented encrypted-key recovery pattern](https://www.chezmoi.io/user-guide/frequently-asked-questions/encryption/),
with recovery completed before preview, mode `0600` at creation, recipient
validation and atomic publication that cannot replace an existing identity.

## Advanced: shared configuration only

```sh
bin/init --machine g14        # desktop on the new machine
```

Without `--machine`, first initialization prompts for `g14` or `desktop`. The
selection lives in `~/.config/chezmoi/chezmoi.toml`; repeat runs retain it. An
explicit conflicting selection is refused; intentionally edit `[data].machine`
in that local file to change it. The configured source points at this checkout,
not a second clone in chezmoi's default data directory.

Initialization installs the repository hook, initializes chezmoi, recovers the key when absent, validates its public
recipient and permissions, lists owned targets, displays `chezmoi diff`, and
requires the literal answer `yes` before application. Diff may show decrypted
secrets: preview only in a private terminal. Wrong passphrases and invalid,
permissive or mismatched existing keys stop before applying. Repeat runs reuse
the validated key without asking for its passphrase. An existing chezmoi config
for a different checkout is preserved and must be reconciled manually.

Immediately before application, existing shared files are copied to a private
`$XDG_STATE_HOME/dotawesome/adoption/shared-*` directory (default
`~/.local/state/dotawesome/adoption`). `manifest.json` lists backed-up paths;
symlinks are preserved. To restore, stop editing the affected apps, inspect the
manifest and copy each recorded original back to its matching home path. Files
that did not previously exist have no backup; remove those individually if
undoing adoption. Shared backups are separate from the component ledger.

To update shared files only with the standalone helper:

```sh
git pull --ff-only
bin/init
chezmoi status
chezmoi diff
```

A clean status/diff means deployed shared files match the source. Before accepting
local drift, inspect it; use `chezmoi re-add PATH` to capture intentional edits or
rerun bootstrap to restore repository content. For targeted component work outside the guided flow:

```sh
bin/dotawesome plan niri --profile desktop
# Log out of Niri; run from KDE or a TTY:
bin/dotawesome apply niri --profile desktop
bin/dotawesome plan nvim
bin/dotawesome plan tmux
```

Niri's outside-session safeguards, validation and restoration remain in effect.
The desktop profile uses Alacritty and the G14 browser/workspace setup with
automatic GPU/output selection. It has no laptop display, charge policy, pinned
G14 DRM node, KDE wallet override or laptop-local wallpaper.

## Encrypted editing and recovery

AWS currently uses fresh login plus `credential_process`, so no credentials file
is created. If a real file later needs encryption, create it privately (`umask
077`) and add it with `chezmoi add --encrypt ~/.aws/credentials`. Confirm that its
source is `home/private_dot_aws/encrypted_private_credentials.age` and its target
mode is `0600`. Never add plaintext credentials to Git. Use `chezmoi edit
~/.aws/credentials` to edit via chezmoi's encrypted source handling, then preview
and apply. Keep editor swap, backup and recovery files outside the checkout.

If the local key is lost, run bootstrap in a terminal; it decrypts the committed
recovery copy once. If a key exists but is wrong, bootstrap preserves it and
refuses to proceed. Move it aside privately only after inspecting it, then retry.
No passphrase reset can recover a lost identity: retain both the encrypted copy
and its separate passphrase. Rotation requires decrypting/re-encrypting all
managed encrypted files before retiring the old identity/recovery copy.

`bin/check-secrets --staged` uses Gitleaks `git --pre-commit --staged`; local CI
uses `dir` over tracked working files and nonignored new files. All output is
redacted; scanner absence fails the check. Default Gitleaks rules are extended
with plaintext age identity and AWS credential assignment guards. Encrypted age
sources need no blanket exclusions. See [Gitleaks scanning modes](https://github.com/gitleaks/gitleaks).

## Machine-local logins and acceptance

Run `gh auth login` and `gh auth status` on each machine. Git's credential helper
calls the packaged `/usr/bin/gh`; tokens stay local. Install the current AWS CLI
v2 package, run `aws login --profile cof-aws`, and verify `aws sts
get-caller-identity --profile cof-tf`. `cof-tf` preserves `aws configure
export-credentials --profile cof-aws --format process`; login tokens and caches
are never deployed. Profile account/role identifiers are plaintext metadata.

Optional VPN: import your Surfshark OpenVPN profile locally into NetworkManager
as `us-lax-surfshark`, keeping credentials local. `vpn on` asks for credentials
through nmcli. Missing projects/tools/connections fail only when invoked. Clone
projects into `$HOME/Code`; mise activation is conditional and interactive.

On the G14, use guided setup, inspect the previews and confirm the files you
want to adopt with backups. Keep CachyOS Fish's main config intact.
On desktop arrival, complete guided setup from outside Niri,
log in graphically and verify rendering, Alacritty, browser/workspaces, audio,
locking and portals. Capture `niri msg outputs` with the shared monitor through
both dock HDMI and direct HDMI before choosing any pinned identity/mode; see
[Niri monitor checks](niri.md). Verify fresh GitHub/AWS logins and optional VPN.
These graphical/hardware checks require the actual machine.

Run `bin/ci` before adoption. Integration tests use isolated HOME and XDG paths,
disposable identities and terminal passphrase prompts; no real credential
destination is created by tests.
