#!/bin/bash

# basic setup stuff
sudo apt update
sudo apt install git curl

# install zsh
sudo apt install zsh
# setup oh-my-zsh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# bump out here and  re-run as we swap
# to zsh as default terminal from prompt

# copy .zshrc to home
mv ~/.zshrc ~/.zshrc.oh-my-zsh
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
# setup yarn
curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add -
echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list
sudo apt update && sudo apt install --no-install-recommends yarn

# pure prompt stuff
# create directory for git clone
mkdir -p "$HOME/.zsh"
git clone https://github.com/sindresorhus/pure.git "$HOME/.zsh/pure"

# diff-so-fancy install
npm install -g diff-so-fancy

# neovim and vim stuff
sudo apt install neovim
mkdir ~/.config/nvim
ln -sf "${PWD}"/.config/nvim/coc-settings.json ~/.config/nvim/coc-settings.json
ln -sf "${PWD}"/.config/nvim/init.vim ~/.config/nvim/init.vim
mkdir ~/.config/nvim/undodir

# add rigrep for fzf
curl -LO https://github.com/BurntSushi/ripgrep/releases/download/11.0.2/ripgrep_11.0.2_amd64.deb
sudo dpkg -i ripgrep_11.0.2_amd64.deb
# use :CocInstall coc-json coc-tsserver coc-rls coc-eslint
# to complete Coc installation

# add Vim Plug
sh -c 'curl -fLo "${XDG_DATA_HOME:-$HOME/.local/share}"/nvim/site/autoload/plug.vim --create-dirs \
    https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim'
vim +'PlugInstall --sync' +qa

# install powerline font
mkdir -p "$HOME/.fonts"
cp meslo.ttf ~/.fonts/meslo.ttf

# add tmux
sudo apt install tmux
ln -sf "${PWD}"/.tmux.conf ~/.tmux.conf
# setup tpm
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm

# setup rust
curl --proto '=https' --tlsv1.2 https://sh.rustup.rs -sSf | sh

# setup docker
sudo apt-get install \
    apt-transport-https \
    ca-certificates \
    gnupg-agent \
    software-properties-common

curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
sudo apt-key fingerprint 0EBFCD88
sudo add-apt-repository \
   "deb [arch=amd64] https://download.docker.com/linux/ubuntu \
   $(lsb_release -cs) \
   stable"

sudo apt-get update
sudo apt-get install docker-ce docker-ce-cli containerd.io
# add user to docker group
sudo groupadd docker
sudo usermod -aG docker $USER
