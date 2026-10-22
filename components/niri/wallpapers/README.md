# Wallpapers

`Path.jpg` (forest path) and `OneStandsOut.jpg` (moss with one orange plant) are
unmodified 2560×1600 photographs by Risto Saukonpää <paristo@gmail.com>, licensed
LGPLv3. Upstream metadata and license texts are included alongside the images.

Source: KDE's [plasma-workspace-wallpapers](https://github.com/KDE/plasma-workspace-wallpapers/tree/3e4537eb3852a1c769f6fa003e74d06260c266fb),
commit `3e4537eb3852a1c769f6fa003e74d06260c266fb`:

- `Path/contents/images/2560x1600.jpg` → `Path.jpg`
- `OneStandsOut/contents/images/2560x1600.jpg` → `OneStandsOut.jpg`

The G14 and desktop Niri profiles refer to local copies in `~/Pictures`.
From the repository root, install them before applying either profile:

```sh
mkdir -p ~/Pictures
cp -n components/niri/wallpapers/Path.jpg components/niri/wallpapers/OneStandsOut.jpg ~/Pictures/
```

To select the forest image in KDE:

```sh
plasma-apply-wallpaperimage ~/Pictures/Path.jpg
```
