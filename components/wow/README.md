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
