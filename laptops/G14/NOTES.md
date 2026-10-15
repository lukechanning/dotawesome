# ASUS G14 battery charge limit

`dotawesome-charge-limit.service` applies an 80% charging ceiling once at boot,
using the existing kernel battery interface. It has no background process and
works before either KDE or Niri starts. It does not discharge an already fuller
battery, change power profiles, or alter suspend/lid behavior.

This is a separate machine-level policy, not part of Niri activation or removal.
The repository only contains the unit; it is not installed or enabled yet.

## One owner

Use this unit as the boot-time owner. Do not add a matching Niri/Noctalia startup
command, udev rule, or another charge-limit service. No existing custom charge
rule/service or ASUS daemon was found during preparation. KDE's battery setting
can still change the same kernel value during a session; leave it at 80% to
agree with this policy. No KDE preferences are rewritten by this unit.

If you later adopt ASUS's `asusd`, TLP charge control, or another persistent
charge-limit manager, disable this unit first and configure 80% there instead.
The service deliberately has no Conflicts directive that might stop another
power-management service. It runs once and does not fight subsequent GUI changes.

## Install manually

From the repository root:

```sh
sudo install -m 644 laptops/G14/dotawesome-charge-limit.service /etc/systemd/system/dotawesome-charge-limit.service
sudo systemctl daemon-reload
sudo systemctl enable --now dotawesome-charge-limit.service
```

This writes only `/sys/class/power_supply/BAT1/charge_control_end_threshold`.
The unit skips machines without that interface. An inability to write the
threshold is reported as a service failure.

## Remove manually

```sh
sudo systemctl disable --now dotawesome-charge-limit.service
sudo rm /etc/systemd/system/dotawesome-charge-limit.service
sudo systemctl daemon-reload
```

Removal stops future boot-time application; it leaves the current hardware
threshold unchanged. If desired, select a different limit in KDE after removal.

## Repository validation

`systemd-analyze verify laptops/G14/dotawesome-charge-limit.service` checks the
unit without running it. The service has not been installed or started, and no
reboot/suspend persistence testing is claimed.
