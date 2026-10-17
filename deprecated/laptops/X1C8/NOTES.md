# General Fixes / Config for X1 Carbon Gen 8

## Patches 

### HDA Fix for Bass Sound

Requires `alsa-tools` package, operates on system wake and boot

- /etc/systemd/system/hda-fix.service
- /usr/share/alsa/ucm2/sof-hda-dsp/HiFi.conf

### Battery performance throttling

Requires the `throttled` package, though could work with `undervolt` or similar

- /etc/lenovo_fix.conf
