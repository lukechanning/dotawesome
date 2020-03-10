plugins=(
  git
  zsh-syntax-highlighting
)

## Docker stuff
alias dokker="docker-compose"
alias dokker-clean="sudo rm tmp/pids/server.pid"
alias dokker-console="dokker exec web /bin/bash"

# Git stuff
alias gdiff="git status -s \
 | fzf --no-sort --reverse \
 --preview 'git diff --color=always {+2} | diff-so-fancy' \
 --bind=ctrl-j:preview-down --bind=ctrl-k:preview-up \
 --preview-window=right:60%:wrap"
