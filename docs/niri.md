# Niri + Noctalia

Architectural Jade, refined: graphite surfaces, muted jade accents, broad workspace
tabs, and coordinated window framing. Existing wallpaper
files and selections stay local. App themes and KDE preferences are independent.

## Apply

Requires Arch/CachyOS, Niri 26.04+, native Noctalia 5.2+, and the packaged Niri
session. The component's package list is in `components/niri/definition.json`.

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
- A future desktop gets its own profile once its outputs and hardware are known.

G14 references `~/Pictures/Path.jpg` and `~/Pictures/OneStandsOut.jpg`; images are
not managed or copied. Missing images produce warnings and a dark fallback.
GUI overrides take precedence and are reported by `status`.

External display settings remain absent until identity/modes are captured with
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
| Super+1 / 2 | Workspace One / Two |
| Super+Shift+1 / 2 | Move window to workspace |
| Super+H/J/K/L | Navigate columns/windows |
| Super+Shift+H/J/K/L | Move columns/windows |
| Super+Space / V / S | Launcher / clipboard / control center |
| Super+B / Super+Shift+B / Super+N | Network / Bluetooth / notifications |
| Alt+Tab / Alt+Shift+Tab | Recent windows |
| Super+R / Super+W | Cycle width / toggle column tabs |
| Super+F / Super+Shift+Space | Fullscreen / floating |
| Super+Comma / Period | Consume or expel windows |
| Super+Shift+S / Print | Screenshot UI |
| Super+Alt+L / Super+Shift+E | Lock / confirmed logout |
| Super+Shift+Q | Close window |

Idle lock: 10 minutes; screen-off: 15 minutes. Caffeine inhibits both.
Clipboard selection copies; deletion uses panel confirmation.
Verify lock/unlock, suspend/resume, external displays, and wallet access in the
actual session after relevant changes. Config validation cannot establish these.

Earlier acceptance notes and headless comparison details are in
[the historical plan](history/NIRI-PLAN.md). Its `.build/` previews are local-only.

## Selected design

The managed design is Architectural Jade, refined: a transparent 42px bar with
separate navigation, CPU/RAM, controls, and expanding world-clock groups; solid
panels; Noto Sans at weight 600; 8px tiling gaps and 6px window corners.

Workspace tabs use Noctalia's native grouped taskbar with shallow corners. Each
tab shows its workspace label first, followed by app icons at scale 1.35.
Labels use bar font scale 1.2; the bar remains 42px tall. Workspace boxes sit
directly in the bar, without an outer navigation capsule consuming space. Multiple
windows of the same app share an icon; clicking cycles through those windows.
The focused window has a jade dot. Empty inactive workspaces are hidden,
including the trailing numbered workspace; active empty workspaces remain visible.
Noctalia 5.2.1 displays numeric taskbar labels (`1`, `2`), while Niri's One/Two
names and keyboard shortcuts remain unchanged. This is native taskbar behavior;
the taskbar has no label-source setting in that release.

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
DRM path, laptop output, wallet override or local wallpaper is inherited.
Apply from KDE or a TTY outside Niri. See [bootstrap](bootstrap.md).
