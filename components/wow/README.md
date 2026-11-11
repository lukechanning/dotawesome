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

The couch profile uses the pinned upstream ConsolePortLK Xbox layout. The former
`AzerothFFXIVController` preset is intentionally removed by `install-addons` so
client rebuilds cannot silently restore its custom crossbar, calibration, or UI
placement.

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
the existing 0.84 UI scale, then tune from `/immersion` after a controller test.
The pinned 1.4.14 backport ships `Logic/Controller.lua` but omits it from its TOC;
`install-addons` adds that existing upstream file to the load order after all UI
frames, enabling its intended ConsolePort focus, hints, D-pad selection, and
confirm/cancel handling without carrying a fork.

Rollback consists of restoring `Wow-hd.exe.pre-awesome`, removing
`AwesomeWotlkLib.dll` and `AwesomeCVar`, and restoring the timestamped WTF backup
stored beside the client backups on Hive. No Blizzard software or Awesome WotLK
release binaries belong in this repository.
