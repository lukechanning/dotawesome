export ZSH="/home/luke/.oh-my-zsh"
ZSH_THEME="robbyrussell"

# Pure prompt stuff
fpath+=$HOME/.zsh/pure

autoload -U promptinit; promptinit
prompt pure

# zsh plugins
plugins=(
git
zsh-syntax-highlighting
)
source $ZSH/oh-my-zsh.sh

# Lines configured by zsh-newuser-install
HISTFILE=~/.histfile
HISTSIZE=1000
SAVEHIST=1000
bindkey -e
# End of lines configured by zsh-newuser-install
# The following lines were added by compinstall
zstyle :compinstall filename '/home/luke/.zshrc'

autoload -Uz compinit
compinit
# End of lines added by compinstall


# FZF config
export FZF_DEFAULT_COMMAND='rg --files --no-ignore --hidden --follow --glob "!.git/*"'

# ALIASES
# Local Nav stuff
alias c="clear"

# Local Aliases
alias ohmyedit="nvim ~/.zshrc"
alias ohmyvim="nvim ~/.config/nvim/init.vim"
alias vim="nvim"
alias vi="nvim"

# Docker stuff
alias dokker="docker-compose"

# Rust stuff
alias cr="cargo run"

# Code aliases
# insert local projects here

# Git aliases
alias pull="git pull origin"
alias push="git push origin"
alias commit="git commit -m"
alias status="git status"
