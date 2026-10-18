function vpn --description 'Control the local Surfshark OpenVPN connection'
    if test (count $argv) -ne 1; or not contains -- $argv[1] on off status
        echo 'Usage: vpn on|off|status' >&2
        return 2
    end
    if not command -q nmcli
        echo 'vpn requires nmcli (NetworkManager).' >&2
        return 127
    end
    if not command nmcli connection show id us-lax-surfshark >/dev/null 2>&1
        echo 'Missing VPN connection us-lax-surfshark; import it locally with NetworkManager first.' >&2
        return 1
    end
    switch $argv[1]
        case on
            command nmcli --ask connection up id us-lax-surfshark
        case off
            command nmcli connection down id us-lax-surfshark
        case status
            command nmcli -f GENERAL.STATE connection show id us-lax-surfshark
    end
end
