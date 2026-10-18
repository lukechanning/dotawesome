function slakker-dev --description 'Open a project in an up-to-date local Slakker build'
    set -l launcher "$HOME/Code/slakker/bin/slakker-dev"
    if not test -x "$launcher"
        echo "Slakker launcher missing: $launcher; clone and prepare Slakker first." >&2
        return 127
    end
    "$launcher" $argv
end
