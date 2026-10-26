function steam-connect --description 'Open the radium Steam dashboard on helium'
    if test (hostname -s) != helium
        echo 'steam-connect runs on helium.' >&2
        return 1
    end
    "$HOME/Code/dotawesome/bin/steam-connect" $argv
end
