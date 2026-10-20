# Documents between the G14 and desktop

Run `bin/dotawesome syncthing setup` on **both machines**. This guide is independent
of shared configuration, chezmoi and age enrollment. Repeating it reuses the local
identity and service; missing peer information can be deferred. The other machine
need not be online during preparation. Keep Syncthing's config, identity, database
and credentials local; never add them to this repository.

1. Check requirements. The guide inspects Syncthing, Tailscale, local configuration
   and connectivity. Missing tools can be installed through the existing confirmed
   Arch package flow: `sudo pacman -Syu --needed`, including a full system upgrade.
   Requirements are recorded under the `syncthing` package owner.
2. Connect Tailscale. If disconnected, the guide offers tailscaled startup and
   interactive authentication. Existing membership and settings are preserved.
   To handle this manually, decline the startup prompt. Check
   `systemctl status tailscaled.service`; if needed, run
   `sudo systemctl enable --now tailscaled.service`. Then run `sudo tailscale up`,
   complete authentication and check `tailscale status` before rerunning
   `bin/dotawesome syncthing setup`. A running daemon can still be logged out.
   If Tailscale requests additional flags, resolve that explicitly and rerun.
   Menu handoffs retain the failed step and recovery instructions in Results.
3. Prepare this machine. Confirm adoption if a configuration already exists.
   The packaged `syncthing.service` user service is enabled at login. Existing
   identities are reused. The guide prints the loopback administration URL,
   device ID and Tailscale IPv4; exchange the ID and address with the other machine.
4. Pair. Enter the other machine's ID and Tailscale IPv4. Blank ID defers pairing;
   local preparation remains. Run the same guide on the other machine with this
   machine's values. Never copy identity files between the machines.
5. Review. Only `~/Documents` is added, using folder ID `dotawesome-documents`,
   two-way synchronization, filesystem watching, and staggered versioning with a
   maximum age of 90 days. Existing Documents mappings with another ID or path
   require manual reconciliation first. Unrelated folders/devices and ignore rules
   are preserved when adopting an installation. A fresh unshared default folder is
   removed from configuration only; its files are retained.
6. Enable. Initial contents **merge**; neither machine is authoritative. Later
   deletions propagate. Simultaneous edits may create `.sync-conflict-*` copies;
   compare and merge them yourself, then remove unwanted copies. Dotawesome never
   empties Documents or resolves conflicts.
7. Check with `bin/dotawesome syncthing status`. Complete pending steps on both
   machines. An offline peer is expected until it reconnects. API or service
   failures retain completed work: inspect diagnostics and rerun setup.

Transport uses only explicit Tailscale IPv4 addresses on TCP 22000. Syncthing's
listener binds to the local Tailscale IPv4; the peer has a single explicit TCP
address. Global/local discovery, relays and NAT traversal are disabled globally
with confirmation. The paired device is neither an introducer nor allowed to
accept folders automatically. Tailscale's own transport behavior is unchanged.
Existing installations with other transport requirements should be reconciled
before adoption. The administrative interface must remain authenticated and on
loopback. Non-loopback or HTTPS GUI configurations require local reconciliation;
the guide does not overwrite them or bypass certificate validation.

Status distinguishes missing tools, disconnected Tailscale, stopped service,
pairing pending, offline peer, paused, scanning, syncing, up to date and folder
errors. “Up to date” requires an idle local folder, no local or peer outstanding
items/deletions, a connected peer, and 100% peer completion. A running service
alone does not establish synchronization.

If TCP 22000 is unreachable, check both machines' Tailscale connectivity, tailnet
permissions and host firewall. Dotawesome does not change firewall or tailnet
policy. Inspect `systemctl --user status syncthing.service` and
`journalctl --user -u syncthing`. A listener bound to Tailscale must recover when
Tailscale appears after login; verify this on the actual machines before relying
on unattended operation. The guide offers `loginctl enable-linger` for desktop
availability before login and after logout. It does not change suspend or power
settings; a sleeping or powered-off machine cannot transfer files.

Incoming replacement/deletion can archive the previous version in `.stversions`.
Versioning does not archive local edits and is **not a complete backup**. In the
loopback Syncthing UI, pause the Documents folder on both machines before recovery,
use the folder's Versions view to restore an archived version, inspect the result,
then resume. Retention can remove older versions.
See [Syncthing versioning](https://docs.syncthing.net/users/versioning.html) and
[granular configuration API](https://docs.syncthing.net/rest/config.html).

To disable transfers without deleting files, pause Documents on both machines in
the loopback UI, or stop and disable each user's `syncthing.service`. Removing a
folder from Syncthing configuration also leaves its files in place. Do not delete
Documents to disable syncing. Additional folders, remote administration, automatic
backups and tailnet policy management are outside this guide.

## Acceptance on the actual machines

Automated fixtures do not establish hardware acceptance. On the G14 and desktop:

- Complete pairing, then transfer a new file in both directions from different
  networks; check file contents and status on both machines.
- Modify and delete a disposable file remotely; verify the peer receives changes
  and retains versions. Test simultaneous edits with disposable data.
- Disconnect/reconnect Tailscale and confirm transfers recover without re-pairing.
- Reboot the desktop, including a delayed Tailscale connection; confirm its listener
  recovers, and inspect journal diagnostics if it does not.
- With lingering enabled, test before login and after logout; confirm power settings
  allow the required unattended availability.

These hardware checks are pending until performed on both machines.

## Developer verification

Normal `bin/ci` runs fake-service and disposable-home tests without changing the
machine's Syncthing or Tailscale configuration. To opt into the two-instance test,
install Syncthing on the test PATH and run:

```sh
DOTAWESOME_SYNCTHING_INTEGRATION=1 go test ./internal/app -run TestSyncthingTwoInstanceTransfer -v -count=1
```

This launches two disposable instances on loopback and checks bidirectional
transfer, remote deletion and archived version retention. It does not replace the
Tailscale/hardware acceptance above. Implementation verification in this change
included this test with Syncthing v2.1.6 and the full repository CI suite.
