#!/bin/bash

# install package dependencies
sudo pacman -S nitrogen redshift kitty \
  yay rofi inter-font noto-fonts-emoji \
  lxappearance-gtk3 flameshot playerctl \
  arc-gtk-theme

# AUR packages
yay -S awesome-git picom-tryone-git \
  la-capitaine-icon-theme light-git

# move config files into place
ln -sf "${PWD}"/.config/redshift ~/.config/redshift
ln -sf "${PWD}"/.config/kitty ~/.config/kitty
ln -sf "${PWD}"/.config/awesome ~/.config/awesome
ln -sf "${PWD}"/.config/nitrogen ~/.config/nitrogen

# kick off redshift
systemctl --user enable --now redshift.service
