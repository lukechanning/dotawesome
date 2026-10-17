# Dotawesome: managed Niri + Noctalia alongside KDE

Status: implemented and activated on G14; native validation passes on Niri 26.04 and Noctalia 5.2.1. Login, workspace navigation, screenshots, Vivaldi wallet access, and AMD rendering have been exercised. Remaining hardware/session acceptance checks are recorded below.

## Architectural glass appearance (2026-10-16)

The repository now targets **Option C from the forest-wallpaper comparison**:
smoked blue-grey shell surfaces, cool blue controls, fine outlines and soft
shadows. This appearance revision is prepared and natively validated but **not
activated or visually accepted**. Earlier opaque/no-blur appearance requirements
below describe the previous baseline and are superseded by this section.

- A managed `Architectural-Glass` dark palette supplies fixed colours without
  wallpaper extraction, network downloads or automatic application theming.
  Its JSON file is included in snapshots, validation staging, deployment and
  restoration alongside Noctalia's TOML. Terminal colour metadata is required
  by Noctalia's runtime palette parser even when template application is off;
  including it does not theme any application.
- The top bar is 32 logical pixels high, with 4px outside margins, 8px corners,
  90% opacity, a fine outline, blur and a soft shadow. Existing widgets,
  world-clock hover expansion, actions and sampling intervals remain intact.
- Native glass panels use their existing layouts; the control center floats
  at the top right. Launcher, clipboard, menus and notifications share the
  palette. Shell shadows have 40% alpha; notifications use 90% opacity.
  Noctalia requests correctly shaped compositor blur using its native protocol.
  No forced layer effect is applied: testing showed that forcing `xray` on the
  full-screen panel input layer hides the underlying apps. Application opacity
  is unchanged.
- Niri retains 6px tiling gaps and short animations, with 8px window corners,
  a coordinated focus ring and restrained window shadows. Shell typography
  uses the existing Noto Sans font. App themes and KDE settings are untouched.
- Both wallpaper images, default/favourite paths, GUI selections, rotation
  settings and wallpaper effects are unchanged. The mockup is a style reference,
  not a requirement to reproduce invented app chrome or quick-setting layouts.

### Activation and remaining visual acceptance

Appearance updates can now be applied inside Niri:

```sh
bin/dotawesome activate niri --profile g14 --live --dry-run
bin/dotawesome activate niri --profile g14 --live
```

`--live` requires an existing deployment and permits only explicitly listed
shell styling keys, palette files, Niri framing/gap/animation settings, and
Noctalia's blur policy. Outputs, scaling, shortcuts, launch commands, services,
portals, wallpaper selection, app theming and idle/lock policy are compared and
refused if changed. Compiled profile metadata may be refreshed only while work
apps and the actual generated behaviour remain unchanged. Palette companions
are published first and Niri's config last; unchanged files are not replaced.
Native validation remains mandatory, with admission checked again under the
deployment lock. Live updates refuse interrupted transactions and package
installation, and do not update package provenance or restart session services.
Deactivation, initial installation and behaviour changes still require logout.

The isolated GUI overrides were inspected: only lock-screen layout settings were
present, with no conflicting appearance overrides to remove or reset.

Preflight found one existing managed-file edit: the live output mode is
`2880x1800@60.001`, whereas the deployment record contains `2880x1800@120`.
The user explicitly confirmed 60Hz as correct. After verifying this was exactly
the one-line difference, the old ledger and current config were backed up under
`.build/glass-preview/before-live/`. A locked metadata transaction adopted the
existing 60Hz bytes and recorded a corresponding baseline snapshot, preserving
the original restoration backups. No live configuration or display setting was
rewritten. The G14 build and live dry-run pass; appearance is not activated yet.

### Headless comparison

Actual Niri/Noctalia captures at 2880×1800 and 160% scale are available in
[the comparison report](.build/glass-preview/comparison.html), with the selected
Option C reference, desktop, control center, audio panel, launcher and a sample
notification. The disposable preview uses Xvfb, Niri's nested backend and
software rendering, private D-Bus/config/state/runtime paths, and disposable app
profiles. Preview locking, idle actions, polkit and clipboard history are off;
system-bus and audio services are isolated, so hardware placeholders and elevated
CPU samples are expected. It does not exercise the live wallet or alter wallpaper
selections. Xvfb was extracted locally into the ignored preview directory, not
installed system-wide. Preview processes are cleaned up by their private home.

The captures exposed and verified fixes for a missing runtime palette payload
and forced-xray interference with underlying windows. The native control center
is larger and more card-based, and workspace indicators are pills; those native
layout differences from the generated concept are intentional. Glass surfaces
pick up some green from the existing forest wallpaper. Pixel equality to the
generated mockup is not claimed.

After activation, capture actual screenshots at the existing 160% scale of the
tiled desktop, control center, launcher, clipboard and notifications. Compare
against Option C with both existing wallpapers. Check rounded blur edges,
contrast over green foliage, blue active states, shadows, world-clock expansion
and existing controls. Tune shell opacity/colour/spacing against those captures;
visual acceptance remains pending until reviewed with the user. Check lock and
unlock rendering too, because shell palette changes also affect native lock UI.


## Native validation after package installation (2026-10-15)

The installed validators are now available: `niri 26.04 (8ed0da4)` and
`noctalia v5.2.1 (5.2.1-1-dirty)`. They exposed three compatibility issues that the
previous repository-only checks could not establish: a missing KDL node
terminator in the compact keyboard block, duplicate top-level `binds` blocks,
and the unsupported Noctalia `widget.media.hide_when_idle` key. The keyboard
block is now multiline, command bindings render into the single portable binds
block, and the unsupported setting has been removed.

Both generated profiles pass Niri and Noctalia native validation without
warnings. The G14 build also passes `systemd-analyze --user verify` outside the
agent sandbox (inside it, systemd socket access is restricted). `bin/ci` now
includes native profile tests when the corresponding executables are present;
they are explicitly skipped when unavailable. All 62 tests passed with both
validators installed. No live deployment or session startup was performed while
fixing these issues. Earlier notes about missing validators below are the
historical implementation record.

After a failed activation at validation, packages stay installed and no configs
are deployed. Retry without requesting another package update:

```sh
bin/dotawesome activate niri --profile g14
```

## Shared G14 battery charging policy

An optional system-level [80% charge-limit unit](laptops/G14/NOTES.md) is prepared
for manual installation. It applies once at boot for both KDE and Niri, without
a background daemon. It is separate from Niri's deployment and is not installed
or enabled by `dotawesome activate`. Use it as the sole boot-time charge-limit
owner; leave KDE's GUI limit at the same 80%. Do not duplicate it in Noctalia,
udev, ASUS management, or TLP. Switching to another charge-limit manager means
disabling this unit first. No system or battery settings were changed while
preparing it.

## Lean daily-use baseline (2026-10-15)

The current baseline retains Niri's three named workspaces and the on-demand
work layout, with Noctalia's native bar and controls. No plugins, extra daemons,
or package requirements were added for this revision.

- Workspaces on the left; Amsterdam time in the center. Hover the clock group to
  expand New York, Los Angeles, and Auckland inline. These are native timezone
  clock widgets in an accordion group, not a separate tooltip window or World
  Clock plugin. Each clock's tooltip includes the date and timezone abbreviation
  so next-day differences are visible. The named IANA zones handle DST.
- On the right: CPU/RAM text, network, Bluetooth, volume, battery, caffeine
  state, clipboard, unread notifications, and app tray icons. Media controls
  remain available in the control center opened from the clock. Cider selection
  still needs live validation.
- Middle-click battery to toggle caffeine. The coffee
  glyph provides state feedback. Caffeine pauses idle actions, including the
  10-minute automatic lock and 15-minute screen-off. Manual lock and native
  lock-before-suspend remain configured. It does not change logind lid/suspend
  policy. Verify coffee-state feedback and idle inhibition after login.
- Network, clipboard and notification widgets open their native panels. Tray
  indicators depend on Slack/Signal/etc exposing them; notification counts are
  pending notifications, not guaranteed per-app unread-message counts.
- The Bluetooth bar icon opens its control-center section, also available with
  Super+Shift+B. Weather/calendar/screen-time tabs are hidden. Dock,
  taskbar, desktop widgets, wallpaper rotation, automatic application theming,
  WAN lookup, telemetry and launcher currency-rate fetching stay off.
- CPU samples every 3 seconds and memory every 5 seconds. Extra system-monitor
  GPU/network-throughput/disk polling is disabled. This does not disable
  NetworkManager, Wi-Fi selection, or Bluetooth controls. No resource savings
  are claimed until measured against KDE with the same workload.
- Appearance now follows the architectural-glass revision above, retaining the
  wallpapers already chosen for G14. Compositor transitions remain 100–150ms,
  lock-screen and wallpaper transitions stay off, and Noctalia UI animations
  run at 1.5x speed.
- Native Niri Alt+Tab/Shift+Alt+Tab and Super+Tab/Shift+Super+Tab handle recent
  windows; Alt+grave switches within an app. Directional H/J/K/L navigation and
  movement, width presets and column tabs remain available. No duplicate shell
  window switcher is bound.

The world-clock accordion and configured actions are documented by upstream
[Noctalia bar groups](https://docs.noctalia.dev/noctalia/bar/#capsule-groups),
[clock widgets](https://docs.noctalia.dev/noctalia/bar/widgets/clock/), and
[widget actions](https://docs.noctalia.dev/noctalia/bar/actions/). Niri owns
[recent-window switching](https://niri-wm.github.io/niri/Configuration:-Recent-Windows.html).
Repository tests check the generated configuration and preserve the existing
work-launcher lifecycle tests. Native schema validation, hover sizing at mixed
scales, authentication, app notification/tray behavior and player controls remain
pending until the packages are installed and a real Niri login is available.
No installation, activation, service reload or session switch was performed for
this revision.

## Implemented command and activation guide (2026-10-14)

The public entry point is `bin/dotawesome` (Python 3.11+, standard library only).
Run it from any directory using its absolute path, or from this checkout as below.
No command invokes the legacy installers. Only `niri` is supported. There is no
all-components mutation, dependency resolver, migration engine, or plugin system.
Existing i3, Awesome, shell, tmux, editor, and system-install files are unmanaged.

```sh
bin/dotawesome list
bin/dotawesome check niri --profile generic
bin/dotawesome check niri --profile g14
bin/dotawesome build niri --profile g14
bin/dotawesome activate niri --profile g14 --dry-run
bin/dotawesome activate niri --profile g14 --dry-run --install-packages
bin/dotawesome status
```

`check` returns nonzero on missing prerequisites or detected conflicts. Dry-run
returns a reviewable plan even when dependencies are missing; inspect its errors
before proceeding. Both are read-only: neither creates a build, home file, lock,
or state directory. Dry-run prints package operations, destination actions, text
diffs, and the unit reload operation. `build` only writes ignored repository
snapshots; native validation is attempted when all validators are available.

For **explicit Stage B activation**, from Plasma after reviewing the dry-run:

```sh
# Use already-installed dependencies:
bin/dotawesome activate niri --profile g14
# OR explicitly permit package installation and a full Arch system upgrade:
bin/dotawesome activate niri --profile g14 --install-packages
```

The package path is `sudo pacman -Syu --needed` with the exact package list in
`components/niri/definition.json`; it preserves pacman's interactive prompts and
never uses `-Sy` or an AUR helper. Packages remain installed after config rollback
or ordinary deactivation; optional tracked-package cleanup is described below. Minimum supported installed versions are Niri 26.04 and native
Noctalia 5.2; `noctalia-shell` v4 does not satisfy the requirement. Package
availability comes from the system's configured Arch/CachyOS repositories.

Activation validates with `niri validate --config`, `noctalia config validate`
on the generated TOML alone, and `systemd-analyze --user verify`. Validator
warnings on stdout **or stderr** fail activation even on exit zero. It requires
the packaged Niri session, `niri.service`, and `/etc/pam.d/login` (Noctalia's
upstream default PAM service). This only establishes file availability, not
successful authentication: manual lock/unlock remains a required first-login
check. No PAM or wallet settings are edited.

After deployment, log out and select the packaged **Niri** session in the existing
login manager. Keep **Plasma** as the default. Activation never starts/enables a
service, changes the login manager, switches session, or forces logout.
Activation and deactivation refuse to operate while Niri is active, including
when `niri.service` is active from another terminal. Concurrent same-user desktop
sessions are outside this implementation's scope.

### Files, ownership, and recovery

- `lib/dotawesome/manager.py`: shared copies, ownership, original backups,
  transaction journal, atomic file replacements, and rollback.
- `lib/dotawesome/niri.py`: profile checks, rendering, native validators, and
  Niri-only lifecycle policy. The component definition lists packages,
  destinations relative to `XDG_CONFIG_HOME`, and validators.
- `components/niri/base.kdl` and `noctalia.toml`: portable declarative baseline.
- `components/niri/profiles/{generic,g14}.toml`: machine outputs and argv arrays.
  A profile argument can also be a path to your own TOML file. Commands are
  executed as argv, never interpolated through a shell.
- `.build/dotawesome/niri/<sha256>/`: content-addressed generated snapshot,
  ignored by git. `manifest.json` maps absolute deployment paths to bytes and
  modes; numbered files contain those exact bytes. Snapshot files are read-only,
  and reuse checks for tampering. Paths are bound to the target user's XDG roots.
- `$XDG_STATE_HOME/dotawesome/installed.json` (default
  `~/.local/state/dotawesome/installed.json`): private component ownership ledger
  with original backups (base64 data/modes or symlink targets), installed bytes,
  snapshot, and profile source. Preserve this file until deactivation.

Activation copies bytes rather than linking to the checkout. Editing the repo
has no effect on the active deployment. `status` reports modified deployed files,
pending differences against the checkout/profile, GUI overrides, and outstanding
session validation. Updates retain the **first** original backup. Unexpected
changes to managed files, directory collisions, symlink parents, and destinations
owned by another component are refused. An existing regular file or final
symlink is backed up; its symlink target is never written through.

All mutations share a process lock. A private `transaction.json` records the
before/after images before deployment starts. Deployment or reload failure
restores completed file changes and the old ledger. An interrupted transaction
is recovered before the next ordinary mutation; live updates refuse recovery,
and read-only commands never recover it.
Recovery refuses post-interruption edits, retaining the journal for inspection.
If recovery is blocked, preserve the journal, ledger, and edited file before
manually restoring either recorded before/after bytes, then retry. A reload
failure retains recovery information until a later successful reload.

For an ordinary edited-file refusal, copy your edits somewhere safe, compare
against the recorded `installed` data, restore the recorded bytes/mode, and then
update or deactivate. Do not delete the ownership ledger to bypass the check.
Backups preserve file contents, permissions, and symlink targets; timestamps,
ACLs, and extended attributes are outside this first implementation. Empty
created parent directories may remain after removal.

### Niri-only shell and KDE coexistence

The managed unit is `dotawesome-niri-noctalia.service`. A single managed
`niri.service.d/50-dotawesome.conf` adds it to Niri's Wants. The shell unit has
`After`, `Requisite`, and `PartOf` set to `niri.service` and requires the Niri
desktop environment. It restarts on failure and stops with Niri. No duplicate
`spawn-at-startup`, global enablement, Plasma integration, or independent idle,
notification, polkit, Bluetooth, audio, or power daemon is added.

The portal file is **niri-portals.conf**, never a global portals.conf; it selects
GNOME/GTK for Niri and leaves KDE's selection alone. It does not select or launch
a new Secret Service provider. Existing wallet access must be checked after
login. Existing user/system desktop autostarts are inspected with user override,
Hidden, OnlyShowIn, and NotShowIn precedence; enabled user-unit links are also
checked for competing helpers. This is a conservative preflight audit, not a
complete proof of D-Bus/runtime ownership. Resolve identified conflicts by
reviewing their session scope; the command never automatically disables them.

Noctalia receives service-local paths:

| Purpose | Default location |
| --- | --- |
| Declarative config | `~/.config/dotawesome/niri/noctalia/noctalia/config.toml` |
| GUI overrides | `~/.local/state/dotawesome/niri-shell/noctalia/settings.toml` |
| Internal shell state | same directory, `state.toml` |
| Shell data/plugins | `~/.local/state/dotawesome/niri-shell-data/noctalia/` |

These honor XDG config/state roots. Noctalia-specific variables have home-root
shape, hence the final `noctalia/` directory. Credentials stay in the existing
Secret Service provider; clipboard/cache data remains local and uncommitted.
Without a usable provider, durable encrypted history is not assured: treat it as
session-only until restart/wallet behavior has been verified.

GUI overrides win over declarative TOML. Inspect the effective settings from a
Niri terminal, matching the service environment:

```sh
NOCTALIA_CONFIG_HOME="${XDG_CONFIG_HOME:-$HOME/.config}/dotawesome/niri/noctalia" \
NOCTALIA_STATE_HOME="${XDG_STATE_HOME:-$HOME/.local/state}/dotawesome/niri-shell" \
noctalia config export full
```

To reset overrides: log out of Niri, back up **only** that isolated
`settings.toml` outside its directory, then move it out of the state directory
and log in again. Do not delete the whole state directory, clipboard history,
cache, or wallet credentials. Exported settings may contain private values;
review before committing any selected changes to the baseline.

### Profiles, monitor capture, and shortcuts

The G14 profile reuses KDE's local `~/Pictures/Path.jpg` as the default wallpaper
and lists it plus `~/Pictures/OneStandsOut.jpg` as Noctalia wallpaper favorites.
Image files remain outside the repo and deployment ownership. These are static
choices, with rotation and automatic app theming disabled. Per-monitor selection
must be checked in Niri; saved GUI overrides can supersede the default. Missing
images produce preflight warnings and retain the dark fallback. The generic
profile keeps its plain dark background.

Generic uses automatic outputs and Alacritty, with an optional browser command.
G14 opens only Vivaldi and Alacritty, each in a full-width column on the default
workspace One. Two starts empty; editors and media apps are opened manually.
Exact app IDs are initial candidates; verify with `niri msg --json windows` and
adjust the profile if the installed build differs. A missing app is reported by
the work shortcut and never blocks login. Initial widths only apply to newly
opened windows; existing selected windows retain their manual sizes. Multiple
matches prefer the target workspace, then the lowest window ID. One runtime
lock suppresses overlapping invocations; each enabled app waits at most 12
seconds for its window (each IPC call also has a 5-second timeout). Failures are
printed and sent through Noctalia, while remaining apps continue.

G14's laptop rule is 2880x1800 at 60.001 Hz and scale 1.6. The external rule is
**absent** until capture. In Niri, record `niri msg --json outputs` through dock
HDMI and direct HDMI and compare make/model/serial and advertised modes. Copy
the G14 profile to a reviewed machine-specific TOML file and add:

```toml
[external]
identity = "MANUFACTURER MODEL SERIAL" # replace with exact Niri output identity
mode = "1920x1080@ACTUAL_REFRESH"     # replace with an advertised mode
scale = 0.85
verified_connections = ["dock-hdmi", "direct-hdmi"]
advertised_modes = ["1920x1080@ACTUAL_REFRESH"]
```

The placeholder mode above intentionally fails validation. Use real captured
values. Connector-only identities and unverified modes/connections are refused.
The verified external gets startup focus and the named-workspace preference,
with the laptop on its right, top-aligned, using scaled logical width (2259 at
85%). If absent, Niri falls back to available outputs. Unidentified displays
stay automatic; no refresh or VRR rule is guessed. VRR is left at Niri's disabled
default. For comparison, change only external `scale` to 0.9 or 1.0 in the
profile and build/activate after logout, preserving each observation. Sub-100%
scaling may reduce crispness; simultaneous app and shell validation is pending.

| Shortcut | Action |
| --- | --- |
| Super+1 / 2 | One / Two |
| Super+Shift+1 / 2 | Move window to workspace |
| Super+Enter | Terminal |
| Super+Shift+Enter | Reuse/open work layout, finish on One |
| Super+Space / V / S | Launcher / clipboard / control center |
| Super+B / Super+Shift+B | Network / Bluetooth controls |
| Super+N | Notification history |
| Super+Shift+S | Screenshot UI (caffeine remains on battery middle-click) |
| Alt+Tab / Alt+Shift+Tab | Native recent-window switching |
| Super+Tab / Super+Shift+Tab | Alternative recent-window switching |
| Super+H/J/K/L | Column left / window down / window up / column right |
| Super+Shift+H/J/K/L | Move columns horizontally, windows vertically |
| Super+R | Width presets: third, half, two-thirds, full |
| Super+Comma / Period | Consume or expel windows left/right |
| Super+W | Toggle column tabs (grouping) |
| Super+F / Super+Shift+Space | Fullscreen / floating |
| Super+Shift+Q | Close window |
| Super+Alt+L | Manual native lock |
| Print / Super+Shift+E | Screenshot UI / confirmed logout |
| Hardware volume, mute, brightness, media | Native Noctalia IPC |

Clipboard selection is copy-only; use the clipboard panel's clear/delete controls
with confirmation. Idle lock is explicitly enabled at 600 seconds, screen-off at
900 seconds, and native lock-before-suspend is enabled. Suspend policy remains
owned by existing services. Confirm lock readiness before relying on idle or
lid behavior. Caffeine temporarily inhibits both idle behaviors. Volume and
network clicks open their control-center sections; Bluetooth is reached through
its bar button, Super+Shift+B or the control center.

To remove the environment, first log out of Niri and log into Plasma:

```sh
bin/dotawesome deactivate niri
bin/dotawesome status niri
```

Only owned config/integration files are restored or removed, then user units are
reloaded. Ordinary deactivation retains packages, GUI overrides, clipboard data,
and credentials.

To also remove packages introduced by Dotawesome, from Plasma:

```sh
bin/dotawesome deactivate niri --remove-packages --dry-run
bin/dotawesome deactivate niri --remove-packages
```

Package provenance is captured when using `activate --install-packages` with this
version of the CLI. The private `packages.json` beside `installed.json` records
before/after additions and newly installed requested roots, independently of
configuration backups. It survives failed validation/deployment and ordinary
deactivation, so package cleanup can be requested later. Repeated activation
retains original ownership. Packages installed before tracking existed, or
installed manually, are never guessed to belong to Dotawesome.

Cleanup asks pacman for the currently unneeded dependency closure of the newly
installed requested roots, then limits candidates to recorded additions. It
preserves pre-existing packages/dependencies and recorded requirements of other
components. Unrelated additions from the full system update are not cleanup
roots. It restores configuration first, then runs interactive `sudo pacman -Ru`
with only the candidate names. No recursive, cascade, forced-dependency, or
no-save flag is passed to the actual removal. Pacman performs the final dependency
checks and can retain candidates now required elsewhere. Removal may leave
`.pacsave` files. The Arch update itself is never downgraded or undone.

Cancellation/failure leaves configuration deactivated and package records intact
for retry. Successfully removed packages lose ownership, so a later manual
reinstallation is not claimed. A missing requested root prevents automatic
cleanup of its now-unreachable dependency chain; keep ambiguous packages rather
than guessing. Do not run unrelated package transactions concurrently with
tracked activation. A hard interruption while collecting package provenance
leaves a pending marker and blocks automatic attribution/removal: review
`packages.json` and `/var/log/pacman.log` before manually resolving the marker;
keep uncertain additions unowned. No package history is inferred retroactively.

Read-only deactivation dry-run never creates state or reloads units. Removal still
requires being outside Niri. GUI state, credentials, cache and build snapshots
remain after optional package cleanup. Package behavior is tested with mocks;
no real package installation or removal was performed during implementation.
See [pacman's removal options](https://man.archlinux.org/man/pacman.8.en#REMOVE_OPTIONS_(APPLY_TO_-R))
for dependency protection and saved-configuration behavior.

### Verification record and outstanding acceptance

Repository checks: `bin/ci` runs Python syntax checks, TOML/JSON parsing, and the
complete unittest suite. Tests use temporary homes and mocked systemd/package
operations, including a test-only second ownership component. They cover
read-only boundaries, unsupported commands, dependency failures, collisions,
repeat updates, first-backup preservation, changed managed files, partial copy
and reload failures, interrupted transaction recovery, ownership isolation,
symlinks, snapshot tampering, and empty/partial/multiple/failed/bounded/concurrent
work-layout invocations.

Observed during implementation on 2026-10-14: Niri and Noctalia executables are
absent; configured package metadata offers Niri 26.04 and Noctalia 5.2.1. Preflight
also reports missing xwayland-satellite and xdg-desktop-portal-gnome. No packages
were installed, live configs written, services activated, or sessions switched.
Generated snapshots and review artifacts are confined to this repo or `/tmp`.
Native KDL/TOML semantic validation and executable-dependent unit verification
remain **pending**. TOML parsing alone does not prove Noctalia schema support.

Stage B must establish login, shell readiness, a single notification/polkit/tray
owner, lock/unlock and PAM, Secret Service/wallet access, screen sharing/file
chooser portals, idle/display-off/suspend, and clean logout with the managed
service inactive in Plasma. Inspect `systemctl --user status
 dotawesome-niri-noctalia.service` and its journal after login and logout.

Stage C must complete every acceptance item below: repeated work layout and
shortcuts; Cider controls; Pixel Buds routing/reconnect; Logitech input; laptop,
dock/direct HDMI, hotplug and suspend/resume; app/menu/pointer/clipboard/shell/
authentication/lock behavior at 160%/85%; KDE → Niri → KDE; GUI override reset;
and comparable idle/coding memory, CPU, GPU and battery measurements. Record
actual observations with versions and display modes. No resource improvement is
claimed before measurement.

Prepared: 2026-10-13. Recheck package versions and upstream guidance before implementation.

Revised: use Noctalia's current native v5 shell for the desktop controls, replacing the originally proposed collection of separate helpers.

## Goal and decisions

Build a slim, keyboard-driven graphical coding session on Arch/CachyOS, with a repeatable working layout and a small amount of desktop information. Keep KDE available for the traditional desktop experience, calls, gaming, and broader desktop tasks.

Niri is the chosen compositor. The goal is a minimal graphical workspace, rather than an actual TTY or a terminal-only workflow. Vivaldi and Alacritty form the default loadout; editors and media apps are opened manually. Familiar i3 shortcuts should carry over where sensible, while embracing Niri's scrolling columns rather than attempting to recreate i3's arbitrary split tree.

Noctalia is the chosen shell for its integrated bar, launcher, audio/Bluetooth control center, clipboard, notifications, and locking. Niri owns window layout and output scaling; Noctalia owns the desktop controls. Keep Plasma Login Manager; do not adopt Noctalia Greeter or the Umbriel compositor. Prefer upstream shell integration over custom glue while keeping the visible UI small.

Primary benefits are the working model, reduced desktop complexity, and per-monitor scaling below 100%. Lower desktop resource usage is an expectation, not a measured result. Do not promise faster builds, lower input latency, or improved battery life without comparison on this machine.

## Current environment

Observed during planning:

- ASUS ROG Zephyrus G14 GA403WR, approximately 30 GiB usable RAM.
- AMD Radeon 880M/890M integrated graphics and NVIDIA RTX 5070 Ti Mobile, using amdgpu and NVIDIA drivers respectively.
- Plasma on Wayland, with Plasma Login Manager as the existing display manager.
- Saved laptop display configuration: 2880×1800, 120 Hz, 160% scaling.
- PipeWire, WirePlumber, BlueZ, NetworkManager, power-profiles-daemon, and KDE wallet components already available.
- Existing app launchers for Slakker, Alacritty, Vivaldi, and Cider.
- NVIDIA reported runtime-suspended and AMD active during the planning inspection. Preserve efficient laptop-only behavior where possible; do not assume this state persists under other workloads.
- Stable Niri and the supporting tools are available in the configured repository metadata. Niri was listed as 26.04; target the actual installed version when implementing.
- Repository metadata lists native `noctalia` 5.2.1 and legacy `noctalia-shell` 4.7.7. Install `noctalia`, not the archived Quickshell-based v4 package; recheck versions before implementation.

User's work equipment:

- A large 1080p gaming monitor, normally connected over HDMI through a USB-C dock; direct HDMI is also available.
- Mostly external-screen work, with the built-in display sometimes used alongside it.
- Logitech ergonomic mouse and keyboard; exact models and special-button requirements are not yet known.
- Bluetooth Pixel Buds 2A.

The external monitor and peripherals were disconnected during planning. Their model identifiers, refresh modes, GPU routing, and live behavior still need to be captured and tested.

## Working layout and shortcuts

Create permanent named workspaces:

| Shortcut | Workspace | Apps and initial layout |
| --- | --- | --- |
| Super+1 | One | Vivaldi and Alacritty in separate full-width columns |
| Super+2 | Two | Empty |

Use Super+Shift+Return to open the working environment. This is an on-demand action, not login autostart. Open no project by default.

The launcher must:

- Identify windows through Niri IPC using verified app identifiers, including any Xwayland differences.
- Launch missing apps and reuse existing matching windows.
- Place selected windows on One, then focus One and the first selected window.
- Preserve manually adjusted sizes of existing windows. Initial sizes apply to newly opened windows.
- Serialize repeated invocations so rapid key presses do not launch duplicates.
- Use bounded waits for app windows and report launch failures through notifications without preventing other apps from opening.
- When an app has multiple existing windows, choose a matching window already on its target workspace first, otherwise one matching window; leave other windows alone.

Other bindings:

- Super+Enter: Alacritty.
- Super+Space: Noctalia app launcher.
- Super+V: Noctalia clipboard panel.
- Super+S: Noctalia control center.
- Super+H/J/K/L: directional navigation; Shift variants move windows or columns using the appropriate Niri actions.
- Super+Shift+Q: close focused window.
- Super+F: fullscreen toggle.
- Super+Shift+Space: floating toggle.
- Super+R: cycle column-width presets of one-third, one-half, two-thirds, and full width.
- Super+Comma/Period: consume or expel windows between columns.
- Super+Alt+L: Noctalia manual lock.
- Print or Super+Shift+S: Niri screenshot UI.
- Super+Shift+E: Niri's confirmed session exit.
- Hardware media, volume, mute, and brightness keys: use Noctalia's version-matched IPC and the working panel backlight device. Verify Cider player selection; use a Cider-targeted playerctl binding only if the native media selection cannot reliably target it.

Use the chosen G14 wallpapers (plain dark fallback), small gaps, a clear focus ring, modest corners and short transitions. Keep transparency, blur, shadows and decorative shader effects disabled. Do not port the extensive old Awesome widgets or X11 startup stack.

## Minimal Noctalia shell

- **Bar:** one compact top bar on each enabled display. Show workspace names, hover-expandable world clocks, battery/caffeine, CPU/RAM, network, Bluetooth, volume, clipboard, pending notifications and a small tray. Access media controls through the control center. Prefer text/readable status over decorative graphs.
- **Control center:** audio device/stream selection, Bluetooth pairing and reconnection, Wi-Fi/network status, power profiles, and notifications. Clicking volume or network opens the relevant section directly; Super+Shift+B opens Bluetooth.
- **Launcher and clipboard:** use Noctalia's native panels. Enable text/image history with a 100-entry unpinned limit, clear/delete confirmation, and copy-only selection rather than automatic paste. Verify encrypted persistence with the existing Secret Service provider; session-only history while the wallet is unavailable must be clearly documented.
- **Notifications and authentication:** explicitly enable Noctalia's notification daemon and native polkit agent for Niri. Ensure competing agents do not start there. Keep KDE's own agents in Plasma.
- **Locking and idle:** explicitly enable native idle lock at 600 seconds and screen-off at 900 seconds; retain lock-before-suspend. Verify the packaged PAM configuration, lock readiness, idle inhibition, and resume behavior. Do not also run swaylock/swayidle.
- **Appearance:** fixed dark palette, opaque surfaces, modest rounded corners, chosen wallpapers with a dark fallback, and short transitions without blur or shadows. Explicitly disable client-requested Noctalia blur in Niri's layer rules; merely omitting blur rules may not disable it.
- **Optional features:** disable dock, desktop widgets, weather/location, calendar integration, screen-time tracking, audio visualization, wallpaper rotation, automatic app theming, and overview type-to-launch. Start without plugins.
- **Configuration:** version the declarative Noctalia TOML baseline. Keep GUI-managed overrides and runtime state outside the repo. Document that GUI overrides win, how to inspect/export effective settings, and how to back up and clear conflicting overrides without deleting history or credentials. Treat validator warnings, especially unknown keys, as issues to resolve even if validation exits zero.
- **Portals:** Niri's documented GNOME/GTK integration, with KDE retaining its own backend. Noctalia does not replace the portal backend.
- **X11 compatibility:** xwayland-satellite through Niri's automatic integration, not a separately globally enabled service.

Reuse system-level networking, Bluetooth, audio, and power services. Noctalia is their UI, not a second service stack. Preserve the existing suspend/lid policy and make shell lock/display handling consistent with it. Do not add Waybar, Fuzzel, Mako, cliphist, Blueman, pavucontrol, or standalone lockers to the baseline; any required fallback needs a concrete observed gap and exclusive ownership.

## Monitor scaling and placement

Configure the laptop at its native 2880×1800 resolution and 160% scale (matching the preferred KDE setting). Configure the work monitor at native 1920×1080 and **85% scale**.

Niri explicitly supports sub-100% scaling. At 85%, 1080p provides approximately 2259×1271 logical pixels, about 38% more logical area than 100%. The physical display remains 1080p. Upstream acknowledges some loss of crispness below 100%; do not claim pixel-perfect text or universal app compatibility at this scale.

- Capture monitor manufacturer/model/serial with `niri msg outputs` and match the work display by identity rather than connector name. Verify identity through both dock HDMI and direct HDMI.
- Prefer the work display for startup focus and the three named workspaces when connected; use the laptop when absent. Implement this using functionality supported by the installed Niri release, rather than unreleased documentation examples.
- Default arrangement: external screen to the left of the laptop, aligned at the top. Use scaled logical dimensions when positioning outputs.
- Keep both displays enabled initially. External-only and closed-lid operation must be tested against the existing suspend policy before claiming support.
- Select only advertised refresh modes. Use the laptop's advertised 120 Hz mode and the external's highest stable advertised 1080p mode supported through the connection. Keep VRR disabled initially.
- Prefer native Wayland app operation. Any required launch flags or environment settings belong to the Niri session or its app wrappers, not global shell/profile files or KDE's launchers.
- Verify Noctalia's bars, panels, settings window, authentication prompts, and lock screen at both 85% and 160%. Shell UI scale is distinct from Niri output scale; avoid using one global shell override to compensate for an app-specific problem.
- Provide an explicit way to compare the external at 85%, 90%, and 100% during validation; start at 85%. Do not silently replace the requested scale if an app is problematic.

Native Wayland scaling and legacy X11 scaling need separate checks. xwayland-satellite documents that mixed-DPI setups use the lowest monitor DPI for its scaling settings, potentially leaving text small on higher-DPI screens. Test the installed app builds rather than assuming all Chromium/Electron applications use the same rendering backend.

## KDE coexistence and installation

Use Niri's packaged session entry and native systemd session integration. Switch through the existing login manager by logging out, selecting a session, and logging back in. Concurrent KDE and Niri sessions for the same user are outside the initial scope.

- Bind Noctalia and helper startup/shutdown to the Niri session, using a Niri-only systemd service and upstream lifecycle guidance. Restart Noctalia on failure while its session is active; verify startup readiness and missing-shell behavior for lock and launcher shortcuts. It must not autostart in KDE.
- Give each helper one startup owner. Avoid duplicate config autostart plus service activation.
- Keep portal selection desktop-specific; preserve KDE's backend and app associations.
- Keep existing system services and display-manager configuration. Do not replace Plasma Login Manager or change the default session as part of setup.
- Disable automatic app-theme generation and greeter sync. Noctalia must not rewrite shared GTK/Qt themes, terminal/editor configs, wallpaper settings for Plasma, or login-manager state.
- Verify the existing KDE wallet/Secret Service provider under Niri for Vivaldi and Noctalia clipboard persistence. Reuse one provider rather than blindly launching another keyring. Session switching is by logout/login, not concurrent same-user sessions.
- Audit existing autostart entries and user units before installation. Make changes only to actual conflicting entries; do not rerun the old i3/Awesome installers, which globally enable Redshift and contain obsolete X11 helpers.
- Add dedicated Niri and Noctalia configuration, the work-layout script, and Niri-only session integration to this repo, plus setup/removal scripts and a short usage guide.
- Setup must be repeatable, back up file collisions, avoid overwriting unrelated user configuration, and clearly report its package and configuration changes.
- Removal must undo only this setup's links, units, and integration changes and restore backups where applicable. Do not automatically remove shared packages or services.
- Install stable repository packages rather than git/AUR variants when available. Do not perform an Arch partial upgrade; use the normal supported system-update path when package installation requires it.

## Hardware and efficiency checks

The normal Logitech input path should not require a WM-specific driver. Verify typing, scrolling, media keys, acceleration, and any required extra buttons. Add Solaar only if an actual device feature requires it; verify exact model support first.

Pixel Buds reuse BlueZ/PipeWire/WirePlumber. Test pairing if needed, music output, reconnect after suspend, volume controls, and KDE/Niri session switching. Do not assume codec availability or microphone behavior from the product name alone.

On the hybrid-GPU laptop:

- Identify which GPU drives the panel, dock output, and direct HDMI; different connections may have different power implications.
- Check startup, hotplug, suspend/resume, brightness, and runtime GPU suspension.
- Inspect Niri VRAM usage when NVIDIA is involved. Apply the documented per-process NVIDIA buffer-pool mitigation only if indicated by the observed driver/profile behavior.
- Avoid speculative global GPU environment overrides or kernel changes.
- Preserve existing power profiles and thermal behavior; a smaller desktop does not inherently keep the discrete GPU asleep.

Compare KDE and Niri + Noctalia with the same app workload, display modes, brightness, Bluetooth state, and power profile. Include the shell's resource use; fewer integration files do not establish lower runtime overhead. Record both settled idle and ordinary coding use. Use proportional memory accounting where available, CPU activity, GPU activity/VRAM, and battery discharge over repeated comparable intervals. A restricted agent process namespace may not expose the actual desktop processes; measurements must come from the real sessions.

## Implementation sequence

1. Recheck native Noctalia/Niri versions, upstream guidance, packaged PAM support, existing session units, app commands, and configuration collisions. Prepare changes and reversible setup in this repo.
2. Install Niri + Noctalia and establish laptop-only login, scaling, shell readiness, locking, notifications, authentication, wallet access, portals, audio, and clean logout.
3. Configure the restrained shell profile, capture live app identifiers, and implement the work-layout shortcut and native bar/control-center/clipboard interactions.
4. At work, capture the monitor identity and advertised modes. Configure and validate 85% scaling through dock HDMI and direct HDMI alongside the 160% laptop display.
5. Validate peripherals, suspend/reconnect, session switching, and resource/power behavior. Record actual results and remaining limitations in the usage guide.

## Acceptance checklist

- [ ] Niri and Noctalia configurations validate against installed releases with no unresolved unknown-key warnings; scripts and systemd units pass their applicable static checks.
- [ ] The work shortcut behaves correctly from an empty session, a partially populated session, multiple app windows, and rapid repeated invocation.
- [ ] Slakker, Alacritty, Vivaldi, and Cider start with the expected backend and initial placement; existing manual sizes are preserved.
- [ ] Time, battery, CPU, RAM, audio, Bluetooth, Cider controls, clipboard selection, and clipboard clearing work.
- [ ] Native control-center audio routing and Pixel Buds pairing/reconnection work; clipboard history survives restart when the existing wallet is available and behaves predictably when locked.
- [ ] Only Noctalia owns shell panels, notifications, idle locking, and polkit prompts in Niri; optional features and automatic app theming are disabled.
- [ ] Laptop-only, dock HDMI, and direct HDMI operate at their intended native modes and scales.
- [ ] At simultaneous 160%/85%, text remains acceptable to the user; pointer targeting, menus, dialogs, text selection, drag operations, screenshots, and clipboard behave correctly.
- [ ] Apps moved between displays resize/repaint correctly; hotplug restores usable workspaces without losing windows.
- [ ] Logitech input and required buttons work; Pixel Buds reconnect and route Cider audio correctly.
- [ ] Manual/idle locking, lock-before-suspend, brightness controls, lid behavior, and suspend/resume work.
- [ ] Noctalia restart restores the shell correctly; lock requests are verified against the real session and shell failure is not mistaken for a successful lock.
- [ ] KDE → Niri → KDE leaves no duplicate or lingering WM helpers and preserves wallet access, app associations, portals, and file dialogs.
- [ ] Noctalia UI, menus, lock screen, and authentication prompts remain usable across both display scales and hotplug; shared KDE/app appearance is unchanged.
- [ ] The documented baseline can be reproduced after backing up and resetting GUI overrides, without deleting private runtime data.
- [ ] Setup and removal are repeatable and restore conflicting-file backups safely.
- [ ] Comparable resource/power observations are recorded; improvements are reported only where measured.

External hardware checks remain pending until those devices are connected. An untested dock, monitor, peripheral, or app-scaling path must remain explicitly unvalidated rather than being marked complete.

## Validated sources

Sources were consulted during planning; revisit version-sensitive instructions before execution.

- [Niri project and window model](https://github.com/niri-wm/niri)
- [Niri setup, desktop coexistence, and session launch](https://niri-wm.github.io/niri/Getting-Started.html)
- [Niri session-bound systemd helpers](https://niri-wm.github.io/niri/Example-systemd-Setup.html)
- [Niri named workspaces](https://niri-wm.github.io/niri/Configuration:-Named-Workspaces.html)
- [Niri window rules](https://niri-wm.github.io/niri/Configuration:-Window-Rules.html)
- [Niri output identity, modes, and per-output scaling](https://niri-wm.github.io/niri/Configuration:-Outputs.html)
- [Upstream sub-100% scaling support and crispness tradeoff](https://github.com/niri-wm/niri/discussions/480)
- [Niri Xwayland integration](https://niri-wm.github.io/niri/Xwayland.html)
- [xwayland-satellite mixed-DPI caveat](https://github.com/Supreeeme/xwayland-satellite#scalinghidpi)
- [Niri NVIDIA considerations](https://niri-wm.github.io/niri/Nvidia.html)
- [Niri lid/switch events](https://niri-wm.github.io/niri/Configuration:-Switch-Events.html)
- [Noctalia current native shell and legacy-version distinction](https://docs.noctalia.dev/noctalia/)
- [Noctalia installation](https://docs.noctalia.dev/noctalia/getting-started/installation/)
- [Noctalia Niri integration, blur disabling, and suspend locking](https://docs.noctalia.dev/noctalia/compositor-settings/niri/)
- [Noctalia control center: audio, Bluetooth, and power](https://docs.noctalia.dev/noctalia/control-center/)
- [Noctalia declarative configuration and GUI override precedence](https://docs.noctalia.dev/noctalia/configuration/)
- [Noctalia shell: clipboard history and native polkit agent](https://docs.noctalia.dev/noctalia/configuration/shell/)
- [Noctalia native idle actions](https://docs.noctalia.dev/noctalia/services/idle/)
- [Desktop-specific portal selection](https://flatpak.github.io/xdg-desktop-portal/docs/portals.conf.html)
- [Desktop autostart scoping](https://specifications.freedesktop.org/autostart/latest/)
- [WirePlumber Bluetooth audio and active-session handling](https://pipewire.pages.freedesktop.org/wireplumber/daemon/configuration/bluetooth.html)
- [libinput](https://wayland.freedesktop.org/libinput/doc/latest/what-is-libinput.html)
- [Solaar device support](https://pwr-solaar.github.io/Solaar/)

### Vivaldi wallet portal correction (2026-10-15)

In the first Niri session, Vivaldi reported it could not unlock its secure key
store even after `kdewallet` was unlocked. Its Local State contained an
`os_crypt.portal` entry (no secret values inspected). The deployed Niri portal
configuration selected only GNOME/GTK providers, neither of which supplied the
Secret interface. A read of `org.freedesktop.portal.Secret.version` failed with
“No such interface”.

The G14 profile now selects `secret_portal = "kwallet"`, rendered as
`org.freedesktop.impl.portal.Secret=kwallet` in the Niri-specific portal file.
This reuses KDE's existing encrypted wallet without changing KDE portal routing
or Vivaldi's encryption backend. Other profiles can select `kwallet` or
`gnome-keyring` when that provider is installed.

Applied this one-file correction to the active session with the deployment
transaction engine, preserving original backups and ownership, then restarted
`xdg-desktop-portal.service`. The Secret interface now reports version `1`.
Repository CI: 63 tests passed, including native Niri/Noctalia validation.
Vivaldi must be fully exited and relaunched; browser recovery remains pending
user confirmation. No credentials were read, reset, or deleted.


### Minimal loadout adjustment (2026-10-15)

The named workspaces are One → Two. Super+1/2 selects those workspaces;
Super+Shift+1/2 moves the focused window to them. Super+Shift+Enter reuses or
opens Vivaldi and Alacritty together on the default workspace One, then focuses
One and the first selected window. Two remains available as a spare workspace. Newly opened
columns fill the available width. No editor or music app is launched. Existing
windows are reused and unrelated applications are never closed automatically;
this does not reset browser tabs or terminal sessions.

Live inspection found the active laptop connector is eDP-2, not the saved
eDP-1; the old rule did not match and automatic scale was actually 175%.
Corrected the connector and selected 160% at the user’s request, matching KDE.
Applied the validated config and profile to the running session using the
managed transaction engine, preserving backups. Renamed Music to Media,
ordered the named workspaces, and moved/sized the existing browser and terminal.
No apps were launched or closed. Native validation passed; repository CI passed
64 tests. CLI status reports no managed-file differences. Earlier 160% display
observations in this document describe the original baseline.

### Confirmed Vivaldi desktop-switching fix (2026-10-15)

Signal follow-up: G14 also opts into `signal_password_store = "kwallet6"`,
rendering `~/.config/signal-desktop-flags.conf` for the installed Arch launcher.
Signal's saved backend was already `kwallet6`. Session D-Bus inspection confirmed
`ksecretd` owns `org.freedesktop.secrets` and `kwalletd6` owns the KDE wallet API;
no extra Secret Service provider was needed. The flag was installed through a
targeted rollback transaction, preserving existing manual Niri configuration
edits and all other deployment entries. Native validation and 67 repository
tests passed. Signal startup confirmation and login-time automatic wallet unlock
remain pending; no credentials or Signal database were reset.

The portal routing correction alone did not resolve Vivaldi's warning. Launching
with `--password-store=kwallet6` opened normally; the user confirmed success.
Vivaldi's official troubleshooting guidance explicitly requires the same key
store across desktop environments:
https://help.vivaldi.com/desktop/troubleshoot/decryption-failed-risk-of-data-loss-error-dialog-on-startup/

G14 now opts into `vivaldi_password_store = "kwallet6"`. This manages
`~/.config/vivaldi-stable.conf`, the flags file supported by the installed Arch
launcher, so terminal starts, Noctalia, work layout, and KDE all select the same
encrypted store. This deliberate cross-desktop browser preference changes no
KDE service or appearance settings. Generic profiles do not deploy the file.
The flags file is owned by the component and removed/restored by deactivate.
No wallet, browser profile, or credentials were reset. Native validation and
all 65 repository tests passed; deployed status reports no file differences.
Login-time automatic wallet unlocking and Slack recovery remain unverified.

### G14 renderer selection (2026-10-15)

Niri's session log selected renderD128 (NVIDIA, PCI 0000:64:00.0) as primary;
NVIDIA runtime status was active. The integrated AMD GPU is PCI 0000:65:00.0.
The G14 profile now sets `render_device` to its verified stable path,
`/dev/dri/by-path/pci-0000:65:00.0-render`, rendered as Niri's documented
`debug { render-drm-device ... }` option. Generic profiles remain automatic.

Saved the change in the repo and managed live config, preserving rollback.
Renderer selection takes effect at the next Niri login. KDE, GPU drivers,
firmware, global environment, and power policies were not changed; no GPU is
ignored or disabled. Native validation passed and all 66 tests passed. User
will assess battery draw after relogin; no further power measurements requested.
External-display performance and NVIDIA runtime suspension remain unverified.


### Daily-use observations before initial commits (2026-10-15)

After relogin, Niri selected AMD renderD129 and NVIDIA runtime status was
`suspended`. A single battery sample was about 10.3 W, compared with earlier
samples around 22.6–24 W with NVIDIA rendering. These are observations under
uncontrolled workloads, not a controlled KDE/Niri comparison or battery-life
benchmark. The user reported smoother operation and improved power draw.

Vivaldi opened normally with KWallet 6 explicitly selected, confirmed by the
user; its managed flags file now pins that backend across launch methods.
Slack still requested sign-in with the same flag, so its configuration was left
unchanged; no keyring cause was established. Automatic wallet unlock after login
remains unverified. Alacritty was adjusted locally to 11 pt after resetting
transient zoom; its shared config remains outside this component's ownership.
Super+Shift+S now opens Niri screenshots; battery middle-click toggles caffeine.

Older dated sections retain the implementation history. The opening status and
latest dated observations describe current validation; external displays,
suspend/resume, locking/authentication, and clean KDE coexistence still require
the remaining acceptance checks. The optional charge-limit unit remains a
separate manual installation.
