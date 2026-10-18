# Personal additions; CachyOS owns config.fish, its prompt and packaged defaults.
alias c 'clear'
alias gpull 'git pull origin'
alias gpush 'git push origin'
alias gc 'git commit -m'
alias gstatus 'git status'
alias gs 'git status'
alias gstash 'git stash'
alias dokker 'docker-compose'

for project in benepicks tirion-pro kyper slakker
    set -l directory $project
    if test "$project" = tirion-pro
        set directory tirion-pro-logging
    end
    function @$project --inherit-variable directory --description "Open $project project"
        if not test -d "$HOME/Code/$directory"
            echo "Missing project: $HOME/Code/$directory; clone it first." >&2
            return 1
        end
        cd "$HOME/Code/$directory"
    end
end
