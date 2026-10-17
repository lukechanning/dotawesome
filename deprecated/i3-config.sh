#!/bin/bash

# install package dependencies
sudo pacman -S nitrogen redshift kitty \
  yay rofi inter-font noto-fonts-emoji \
  lxappearance-gtk3 flameshot playerctl \
  arc-gtk-theme chromium pulseeffects \
  tlpui tlp-rdw s-tui \
  i3-gaps dunst polybar

# AUR packages
yay -S picom \
  la-capitaine-icon-theme light-git \
  nerd-fonts-iosevka

# move config files into place
ln -sf "${PWD}"/.config/i3 ~/.config/i3
ln -sf "${PWD}"/.config/rofi ~/.config/rofi
ln -sf "${PWD}"/.config/picom ~/.config/picom
ln -sf "${PWD}"/.config/dunst ~/.config/dunst
ln -sf "${PWD}"/.config/polybar ~/.config/polybar
ln -sf "${PWD}"/.config/redshift ~/.config/redshift
ln -sf "${PWD}"/.config/kitty ~/.config/kitty
ln -sf "${PWD}"/.config/nitrogen ~/.config/nitrogen
# ln -sf "${PWD}"/.config/chromium-flags.conf ~/.config/chromium-flags.conf

# kick off redshift
systemctl --user enable --now redshift.service
