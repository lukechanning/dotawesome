#!/bin/bash

sudo apt install i3 nitrogen redshift kitty \
  picom rofi flameshot playerctl dunst polybar 

ln -sf "${PWD}"/.config/rofi ~/.config/rofi
ln -sf "${PWD}"/.config/picom ~/.config/picom
ln -sf "${PWD}"/.config/dunst ~/.config/dunst
ln -sf "${PWD}"/.config/polybar ~/.config/polybar
ln -sf "${PWD}"/.config/redshift ~/.config/redshift
ln -sf "${PWD}"/.config/kitty ~/.config/kitty
ln -sf "${PWD}"/.config/nitrogen ~/.config/nitrogen

# kick off redshift
systemctl --user enable --now redshift.service
