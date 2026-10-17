# Niri + Noctalia

Architectural glass shell, cool blue accents, compact framing. Existing wallpaper
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
