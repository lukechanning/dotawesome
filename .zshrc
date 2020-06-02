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

# Code aliases
alias @hooks="cd ~/Code/webhooks-server"
alias @coda="cd ~/Code/elements"
alias @web="cd ~/Code/elements/packages/ts-boilerplate"
alias @templates="cd ~/Code/elements/packages/ts-templates"
alias @demo="cd ~/Code/elements/packages/ts-demoday"
alias @utils="cd ~/Code/elements/packages/ts-utils"
alias @core="cd ~/Code/elements/packages/ts-elements-core"
alias @style="cd ~/Code/elements/packages/ts-style"
alias @contentful="cd ~/Code/elements/packages/ts-contentful"
alias @client="cd ~/Code/elements/packages/ts-client"

# Git aliases
alias pull="git pull origin"
alias push="git push origin"
alias commit="git commit -m"
alias status="git status"

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion
