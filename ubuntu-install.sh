#!/bin/bash

# basic setup stuff
## sudo apt update

# get some basic packages
## sudo apt install git \
    ## curl

# generate an ssh key
## ssh-keygen

# install zsh
## sudo apt install zsh
# setup oh-my-zsh
## sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"
# set default shell to zsh
## chsh -s $(which zsh)
# copy .zshrc to home
## mv ~/.zshrc ~/.zshrc.oh-my-zsh
## cp .zshrc ~/.zshrc
## install zsh syntax highlighting
## git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting

# nvm and node
## curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.35.3/install.sh | zsh
## nvm install node
## nvm use node
# setup yarn
## curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
## echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
## sudo apt update && sudo apt install --no-install-recommends yarn

# pure prompt stuff
## create directory for git clone
## mkdir -p "$HOME/.zsh"
## git clone https://github.com/sindresorhus/pure.git "$HOME/.zsh/pure"

# neovim and vim stuff
## sudo apt install neovim
## ln -sf "${PWD}"/.config/nvim/coc-settings.json ~/.config/nvim/coc-settings.json
## ln -sf "${PWD}"/.config/nvim/init.vim ~/.config/nvim/init.vim
## use :CocInstall coc-json coc-tsserver coc-rls
## to complete Coc installation

# add Vim Plug
## sh -c 'curl -fLo "${XDG_DATA_HOME:-$HOME/.local/share}"/nvim/site/autoload/plug.vim --create-dirs \
    ## https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim'
## vim +'PlugInstall --sync' +qa

# add tmux
## sudo apt install tmux
## ln -sf "${PWD}"/.tmux.conf ~/.tmux.conf
# setup tpm
## git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm

# setup rust
## curl --proto '=https' --tlsv1.2 https://sh.rustup.rs -sSf | sh

# setup docker
## sudo apt-get install \
##     apt-transport-https \
##     ca-certificates \
##     gnupg-agent \
##     software-properties-common
## 
## curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
## sudo apt-key fingerprint 0EBFCD88
## sudo add-apt-repository \
##    "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
##    $(lsb_release -cs) \
##    stable"
## 
## sudo apt-get update
## sudo apt-get install docker-ce docker-ce-cli containerd.io
# add user to docker group
## sudo groupadd docker
## sudo usermod -aG docker $USER