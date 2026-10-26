function steam-stop --description 'Stop the Steam Headless session on radium'
    if test (hostname -s) != radium
        echo 'steam-stop runs on radium.' >&2
        return 1
    end
    systemctl --user stop radium-gaming.service
end
