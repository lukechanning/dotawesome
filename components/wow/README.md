# Radium WoW realm

This component contains the reproducible control plane for a private AzerothCore
development and testing realm on `radium`. It is not deployed to other machines.

This repository does not contain or distribute Blizzard software or game assets.
It is not affiliated with or endorsed by Blizzard Entertainment. The operator
must supply a lawfully obtained compatible client and comply with the licenses
and terms that apply to all upstream software and assets.

## Repository boundary

Git owns the Compose definition, administration scripts, configuration generator,
service units, tests, and revision pins in `versions.txt`. The pins are the
authoritative source manifest; live checkouts and builds can be recreated from
them.

Runtime and generated state stays under `/mnt/hive/WoW` and must not be committed:

- source checkouts, build output, container data, caches, logs, and clients;
- Blizzard software, game data, addon contents, torrent metadata, and download
  locations;
- the populated `server/config/realm.env` and generated configuration containing
  credentials; and
- live databases and database backups.

`WOW_CLIENT_ARCHIVE` must point to an operator-supplied client archive. Its
adjacent `${WOW_CLIENT_ARCHIVE}.torrent` metadata is required by the local fetch
workflow and is never stored in this repository.

## Recovery

Keep the `realm.env` values in a password manager and copy versioned database
backups off `radium`. Syncthing is suitable for copying completed backups, but it
must not synchronize the live MySQL data directory or an actively written backup.

To rebuild the host, restore `/mnt/hive/WoW/server/backups` and the secrets,
recreate the source checkouts at the revisions in `versions.txt`, run `bin/wow
layout`, regenerate configuration, build the images and client data, then restore
the latest verified database backup with `bin/wow restore FILE`.

## Controller and client enhancements

The couch profile uses the complete pinned upstream ConsolePortLK suite and its
stock Xbox layout. The former `AzerothFFXIVController` preset is intentionally
removed by `install-addons` so client rebuilds cannot silently restore its
custom crossbar, calibration, or UI placement. The builder-provided `WotlkHD`
UI addon is also removed: Radium retains `Wow-hd.exe` and `dinput8.dll` for the
renderer enhancements, while ConsolePort alone owns action bars, bags, menus,
loot, maps, and controller navigation.

Smart interaction comes from the pinned
[`noname08662/awesome_wotlk`](https://github.com/noname08662/awesome_wotlk)
fork. Its client extension supplies a forward-facing interaction target, object
highlighting, repeated-loot cycling, the `INTERACTIONKEYBIND` binding, and the
`/interact` command. ConsolePort remains responsible for the controller UI and
all other bindings. The only intended Xbox-profile deviation is logical X (the
left face button) bound to `INTERACTIONKEYBIND`; logical A remains stock Jump.

The third-party release archive, DLL, patched executable, cache, and addon copy
stay under `/mnt/hive/WoW`. Before applying the patch, preserve the playable
executable as `Wow-hd.exe.pre-awesome`. The active client currently uses these
conservative defaults:

- `interactionMode=1`, `interactionAngle=60`, `interactionHighlight=1`;
- UI half-pixel, pixel-snap, texture-sampling, and cooldown-swipe fixes enabled;
- projected-texture and ground-target clamping fixes enabled;
- LuaJIT, indirect camera visibility, expanded nameplate/tab distance, TTS, and
  automatic login disabled unless separately evaluated.

The quest/gossip owner is the pinned 3.3.5a
[`s0h2x/Immersion-WotLK`](https://github.com/s0h2x/Immersion-WotLK)
backport. `DialogUI` is removed during addon installation because the two addons
replace the same Blizzard frames and must not be enabled together. Immersion's
scale and placement remain user-profile settings; start with its stock layout at
the RP6-oriented 0.90 UI scale, then tune from `/immersion` after a controller
test.
The pinned 1.4.14 backport ships `Logic/Controller.lua` but omits it from its TOC;
`install-addons` adds that existing upstream file to the load order after all UI
frames, enabling its intended ConsolePort focus, hints, D-pad selection, and
confirm/cancel handling without carrying a fork.

`RadiumCouchUI` is the small first-party handheld layout layer installed after
ConsolePort and Immersion. It does not add another action bar or input system.
It reserves the lower screen for ConsolePort, moves chat to the upper-left,
enlarges the minimap and objective tracker, anchors ordinary tooltips away from
the controller bar, and stacks open bags above the reserved controller region.
The layout targets the RP6's 1920x1080 six-inch display and deliberately keeps
all gameplay behavior in the stock addons.

MultiBot's playerbot command row otherwise sits permanently across the right
controller cluster, so `RadiumCouchUI` dismisses it at login through MultiBot's
own visibility toggle. LT plus right-stick click (or `/rcui bots`, or MultiBot's
minimap button) summons the real row centered above the controller clusters,
with ConsolePort's interface cursor driving its buttons and submenus; the same
chord dismisses it. MultiBot's saved bar position is left untouched.
ConsolePort's own pet-command ring, which appears only while a combat pet is
out, is held with RT plus right-stick click. Plain right-stick click retains its
mouse-mode and contextual right-click behavior, and existing custom chord
bindings are never overwritten.

The MultiBot, AtlasLoot, and DBM minimap buttons default to the same lower-left
spot (also used by the LFG eye). They are fanned out once to the nearest rim
slots that clear the stock minimap buttons and the zone-name banner; later drags
are kept. ConsolePort's game menu is drawn at 0.95 scale so its System column
fits the 1080p screen; a non-default scale chosen in ConsolePort is kept.

The companion addon also bridges an input-registration gap between the pinned
Immersion backport and ConsolePortLK. While Immersion is visible, face buttons
and the D-pad are routed to Immersion's existing accept, cancel, text, reward,
and selection handlers; hiding Immersion immediately restores gameplay input.
The conversation box is raised 55 UI points above Immersion's default to clear
the controller action region without changing its scale.

Run `/rcui unlock` to display draggable labels for chat, the minimap, objectives,
the player status frame, buffs/statuses, the overall ConsolePort bar, the pet
ring, and Immersion. `/rcui lock` hides the labels and keeps the saved positions
across launches; `/rcui reset` restores the RP6 defaults. ConsolePort's internal
button/cluster geometry remains configurable through `/cp`. In unlock mode,
LB/RB select a handle, the D-pad moves it, A
toggles 2-point/10-point movement, and B saves and locks. Mouse or touch dragging
is optional; neither stick click is used by the editor.

The RP6 profile uses the client's maximum `uiScale` of 1.00 plus a conservative
1.04 UIParent scale from `RadiumCouchUI`. This makes the stock interface about
15.6% larger than the former 0.90 baseline while retaining the 1920x1080 render.

Rollback consists of restoring `Wow-hd.exe.pre-awesome`, removing
`AwesomeWotlkLib.dll` and `AwesomeCVar`, and restoring the timestamped WTF backup
stored beside the client backups on Hive. No Blizzard software or Awesome WotLK
release binaries belong in this repository.
