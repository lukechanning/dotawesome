#!/bin/bash

# basic setup stuff
sudo pacman -Syu

# setup oh-my-zsh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# bump out here and  re-run as we swap
# to ohmyzsh as default terminal from prompt

# diff-so-fancy
sudo pacman -Ss diff-so-fancy
# git aliases
git config --global alias.co checkout
git config --global core.pager "diff-so-fancy | less --tabs=4 -RFX"

# copy .zshrc to home
mv ~/.zshrc ~/.zshrc.arch-backup
cp .zshrc ~/.zshrc

# install zsh syntax highlighting
git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting

# nvm and node
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.38.0/install.sh | zsh
# load nvm so we can use it
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

nvm install node
nvm use node

# setup diff-so-fancy
npm install -g diff-so-fancy

# setup yarn
sudo pacman -S yarn

# pure prompt stuff
# create directory for git clone
mkdir -p "$HOME/.zsh"
git clone https://github.com/sindresorhus/pure.git "$HOME/.zsh/pure"

# neovim and vim stuff
sudo pacman -S neovim ripgrep
mkdir ~/.config/nvim
ln -sf "${PWD}"/.config/nvim/coc-settings.json ~/.config/nvim/coc-settings.json
ln -sf "${PWD}"/.config/nvim/init.vim ~/.config/nvim/init.vim
mkdir ~/.config/nvim/undodir

# use :CocInstall coc-json coc-tsserver coc-rls coc-eslint coc-omnisharp coc-kotlin
# to complete Coc installation

# install powerline font
mkdir -p "$HOME/.fonts"
cp "${PWD}"/fonts/meslo.ttf ~/.fonts/meslo.ttf

# add Vim Plug
sh -c 'curl -fLo "${XDG_DATA_HOME:-$HOME/.local/share}"/nvim/site/autoload/plug.vim --create-dirs \
    https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim'

nvim +'PlugInstall --sync' +qa

# add tmux
sudo pacman -S tmux
ln -sf "${PWD}"/.tmux.conf ~/.tmux.conf
# setup tpm
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm

# setup rust
curl --proto '=https' --tlsv1.2 https://sh.rustup.rs -sSf | sh

# setup docker
sudo pacman -S docker docker-compose
sudo systemctl start docker.service
sudo systemctl enable docker.service

# add user to docker group
sudo groupadd docker
sudo usermod -aG docker $USER



