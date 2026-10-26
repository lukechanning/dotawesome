function steam-start --description 'Start the Steam Headless session on radium'
    if test (hostname -s) != radium
        echo 'steam-start runs on radium.' >&2
        return 1
    end
    systemctl --user start radium-gaming.service
end
