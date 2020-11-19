#!/bin/bash

# copy .zshrc to home
mv ~/.zshrc ~/.zshrc.oh-my-zsh
cp .zshrc ~/.zshrc
# install zsh syntax highlighting
git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting

# git aliases
git config --global alias.co checkout

# nvm and node
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.35.3/install.sh | zsh
# load nvm so we can use it
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion
nvm install node
nvm use node
# setup yarn
brew install yarn

# git stuff here 
brew install diff-so-fancy
git config --global core.pager "diff-so-fancy | less --tabs=4 -RFX"
git config --global alias.co checkout

# pure prompt stuff
# create directory for git clone
mkdir -p "$HOME/.zsh"
git clone https://github.com/sindresorhus/pure.git "$HOME/.zsh/pure"

# neovim and vim stuff
brew install neovim
mkdir ~/.config/nvim
cp -r "${PWD}"/.config/nvim/coc-settings.json ~/.config/nvim/coc-settings.json
ln -sf "${PWD}"/.config/nvim/init.vim ~/.config/nvim/init.vim
mkdir ~/.config/nvim/undodir

# add rigrep for fzf
brew install ripgrep
# use :CocInstall coc-json coc-tsserver coc-rls coc-eslint
# to complete Coc installation

# add Vim Plug
sh -c 'curl -fLo "${XDG_DATA_HOME:-$HOME/.local/share}"/nvim/site/autoload/plug.vim --create-dirs \
    https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim'
nvim +'PlugInstall --sync' +qa

# add tmux
brew install tmux
ln -sf "${PWD}"/.tmux.conf ~/.tmux.conf
# setup tpm
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm

# add meslo font to local fonts
cp "${PWD}"/meslo.ttf ~/Library/Fonts/
