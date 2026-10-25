# Niri + Noctalia

Architectural Jade, refined: graphite surfaces, muted jade accents, broad workspace
tabs, and coordinated window framing. Existing wallpaper
selections stay local. The two reference images are preserved in
`components/niri/wallpapers`; see its README to install them in `~/Pictures`.
App themes and KDE preferences are independent.

## Apply

Requires Arch/CachyOS, Niri 26.04+, native Noctalia 5.2+, and the packaged Niri
session. The component's package list is in `components/niri/definition.json`.

The Super+G jump menu also requires `wlr-which-key` 1.3+. It is a separate
user-installed dependency, outside the managed pacman package list. With Rust/Cargo
and the Cairo, Pango and xkbcommon development libraries available:

```sh
cargo install wlr-which-key --version 1.3.0 --locked --root ~/.local
```

Ensure `~/.local/bin` is on the Niri session's PATH. `check` reports a missing
command before deployment. Removal of the Niri component leaves this shared tool
installed.

```sh
bin/dotawesome check niri --profile g14
bin/dotawesome plan niri --profile g14
# From Plasma, with Niri stopped:
bin/dotawesome apply niri --profile g14
```

Select the packaged Niri session at login. Apply does not start/enable services,
change the login manager, or force logout. Keep KDE available for recovery.
Native Niri, Noctalia, and systemd validation must pass; warnings fail validation.

For an existing deployment, appearance-only updates can run inside Niri:

```sh
bin/dotawesome plan niri --profile g14 --live
bin/dotawesome apply niri --profile g14 --live
```

`--live` permits palette, shell styling, framing, gaps, and animation changes.
Outputs, shortcuts, commands, services, wallpaper selection, and lock/idle changes
require normal apply outside Niri. Live updates cannot install packages or recover
interrupted transactions. No session restart is performed.

To restore originals, leave Niri and run `bin/dotawesome remove niri`.
See [ownership and package cleanup](usage.md).

The control center opens on Home with Super+S; its Calendar tab provides month
navigation without any configured accounts or synchronization. Enabling this tab
is a behavior change: apply normally from Plasma with Niri stopped, using the
commands above. It is deliberately excluded from appearance-only live updates.

## Profiles

- `generic`: automatic outputs, Alacritty, optional browser command.
- `g14`: AMD rendering, KDE wallet integration, Vivaldi and Alacritty work layout;
  built-in display **2880×1800 at 60.001 Hz, scale 1.6**.
- `desktop`: automatic GPU selection, KDE wallet integration, Vivaldi and Alacritty work layout.

The desktop toolbar shows CPU usage, CPU temperature in Celsius, and RAM, with
no battery widget. CPU readings refresh every three seconds. The G14 keeps its
battery widget and CPU/RAM toolbar. For a terminal temperature check, run `sensors`.

G14 references `~/Pictures/Path.jpg` and `~/Pictures/OneStandsOut.jpg`; images are
not managed or copied. Missing images produce warnings and a dark fallback.
GUI overrides take precedence and are reported by `status`.

All profiles share the ISO Shift remap and directional monitor shortcuts.
External monitors use Niri's automatic scale, resolution, refresh rate and
placement. The desktop workspaces 1 and 2 (Super+1/2) use an available output;
the G14 prefers its built-in display. Neither profile references the retired
AOC monitor. Ultrawide outputs need no overrides.

Pinning a mode and workspace/output placement still requires identity/modes captured with
`niri msg --json outputs` through both dock HDMI and direct HDMI. Copy the G14
profile to a custom TOML path and add verified values:

```toml
[external]
identity = "MANUFACTURER MODEL SERIAL"
mode = "1920x1080@ACTUAL_REFRESH"
scale = 0.85
verified_connections = ["dock-hdmi", "direct-hdmi"]
advertised_modes = ["1920x1080@ACTUAL_REFRESH"]
```

Placeholders fail validation. Review actual identity, advertised modes, and scale;
connector-only identities are refused.

## Daily shortcuts

| Keys | Action |
| --- | --- |
| Super+Enter / Super+Shift+Enter | Terminal / reuse or open work layout |
| Super+1 / 2 | Workspace 1 / 2 |
| Super+Shift+1 / 2 | Move window to workspace |
| Super+H/J/K/L | Navigate columns/windows |
| Super+Shift+H/J/K/L | Move columns/windows |
| Super+Ctrl+H/J/K/L | Focus monitor left/down/up/right |
| Super+Ctrl+Shift+H/J/K/L | Move focused window to monitor left/down/up/right |
| Super+Space / V / S | Launcher / clipboard / control center |
| Super+G, then S / B / T / slash | Jump to Slack / browser / terminal / search open windows |
| Super+B / Super+Shift+B / Super+N | Network / Bluetooth / notifications |
| Alt+Tab / Alt+Shift+Tab | Recent windows |
| Super+R / Super+W | Cycle width / toggle column tabs |
| Super+F / Super+Shift+Space | Fullscreen / floating |
| Super+Comma / Period | Consume or expel windows |
| Super+Shift+S / Print | Screenshot UI |
| Super+Alt+L / Super+Shift+E | Lock / confirmed logout |
| Super+Shift+Q | Close window |
| Keyboard backlight up / down keys | Adjust keyboard backlight one level |

Press **Super+G**, release both keys, then press a **lowercase** `s`, `b`, `t`,
or `/`. The small jade menu closes after selection, and typing resumes in the
selected app. Escape cancels. No Enter is needed for an app choice; window search
uses Noctalia's `/win ` provider, where you type a title/app name and press Enter.

App jumps focus the most recently used matching window across workspaces and
monitors, without launching, moving or resizing windows. If an app has no open
window, a notification explains that; Super+Space remains the app launcher.
Slack matches the exact IDs `slack`, `Slack`, and `com.slack.Slack`.
Browser and terminal use each profile's exact `app_ids`, selected with
`jump_key = "b"` and `jump_key = "t"` in its `[[apps]]` entries. Custom profiles
can assign those keys once each; entries without a jump key remain layout-only.
The menu theme lives in `components/niri/jump.yaml` (JSON syntax, valid YAML).

This adds a shortcut and updates the installed helper, so activate with a normal
apply **outside Niri**, not appearance-only `--live`. After activation, manually
check Super+G → `s` from a distant window, `b`/`t`, Escape cancellation, and `/`
search. Selection must return keyboard input immediately to the chosen app.

The ISO `< >` key immediately right of left Shift acts as an additional left
Shift (`lv2:lsgt_switch`), matching the existing KDE keyboard option. This applies
to keyboards with that physical key. Keyboard/remap and shortcut changes require
normal apply outside Niri, not appearance-only `--live`.

For a one-off move before these shortcuts are applied, run
`sleep 3; niri msg action move-window-to-monitor DP-3` and focus the desired
window within three seconds. `DP-3` is the current connector; find the actual
connector on another connection using `niri msg outputs`.

`niri msg outputs` reports the active scale, current mode and logical size. The
laptop's 2880×1800 at scale 1.6 gives 1800×1125 logical pixels. A 1920×1080
external output at scale 0.85 gives roughly 2258×1270 logical pixels. This scale
is shared by the G14 and desktop configurations. Capture identity and modes on
both connection paths before pinning a mode or workspace/output layout.

Idle lock: 10 minutes; screen-off: 15 minutes. Caffeine inhibits both.
Clipboard selection copies; deletion uses panel confirmation.
Verify lock/unlock, suspend/resume, external displays, and wallet access in the
actual session after relevant changes. Config validation cannot establish these.

Earlier design and acceptance history remains available in Git history.

## Selected design

The managed design is Architectural Jade, refined: a transparent 42px bar with
separate navigation, CPU/RAM, controls, and expanding world-clock groups; solid
panels; a background-free center clock; Noto Sans at weight 600; 8px tiling gaps and 6px window corners.

Workspace tabs use Noctalia's native grouped taskbar with shallow corners. Each
tab shows its workspace label first, followed by app icons at scale 1.35.
Labels use the original bar font scale 1.0; the bar remains 42px tall. Workspace boxes sit
directly in the bar, without an outer navigation capsule consuming space. Multiple
windows of the same app share an icon; clicking cycles through those windows.
The focused window has a jade dot. Empty inactive workspaces are hidden,
including the trailing numbered workspace; active empty workspaces remain visible.
Niri workspaces are named `1` and `2`, matching Noctalia's numeric taskbar labels
and the Super+1/2 keyboard shortcuts.

The grouped taskbar was checked in a disposable desktop at 2880×1800, scale 1.6,
including shared app icons, separate populated workspaces, and an active empty
workspace. The local [preview](../.build/refinements/Jade-sizing/split.png)
shows the managed configuration. Activate from Plasma with Niri stopped using
normal apply above; replacing the workspace widget changes behavior and is
excluded from `--live` updates.

The earlier [managed build sheet](../.build/refinements/Jade-managed/option-sheet.png)
predates the grouped taskbar. The local [comparison](../.build/refinements/comparison.html) and approved
[refined sheet](../.build/refinements/Jade-refined/option-sheet.png) contain actual
2880×1800 captures at scale 1.6. Previews and the disposable harness remain ignored
local artifacts, absent from fresh clones. Hardware readings may be placeholders
or reflect software-rendering load; real tray clients need live verification.

This selection and the independent calendar change require normal apply outside
Niri. Workspace visibility and group changes remain excluded from appearance-only
live updates. Preparing this configuration does not activate it. After activation,
check real tray clients, Bluetooth/audio controls, lock/unlock, suspend/resume and
wallet access. Existing application themes, accounts, synchronization and KDE
settings are independent of this design.

## Desktop

Use `--profile desktop` for Alacritty and the shared browser/workspace setup.
GPU and displays are automatic pending monitor identity/mode capture; no G14
DRM path or laptop output is inherited. Like the G14, the wallpaper defaults to
`~/Pictures/Path.jpg`, with `~/Pictures/OneStandsOut.jpg` as another favorite.
Install the bundled copies from `components/niri/wallpapers` into `~/Pictures`
before applying the profile; missing files use the dark fallback.
Like the G14, the desktop profile reuses KDE's Secret portal and pins Vivaldi and
Signal to `kwallet6` across both sessions. KDE/KWallet must already be installed;
wallet contents and credentials remain local. After a normal apply with Niri
stopped, restart the affected apps and verify wallet access in both sessions.
Apply from KDE or a TTY outside Niri. See [bootstrap](bootstrap.md).
