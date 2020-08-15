local filesystem = require('gears.filesystem')
local config_dir = filesystem.get_configuration_dir()
--local bin_dir = config_dir .. 'binaries/'

return {

	-- The default applications that we will use in keybindings and widgets
	default = {

		-- Terminal Emulator
		terminal				= 'kitty',

		-- GUI Text Editor
		text_editor 			= 'subl3',

		-- Web browser
		web_browser 			= 'brave --password-store=basic',

		-- Screenshot
		screenshot 			= 'flameshot gui',

		-- GUI File manager
		file_manager 			= 'thunar',

    -- Email client
    email             = 'mailspring',

		-- Network manager
		network_manager 		= 'nm-connection-editor',

		-- Bluetooth manager
		bluetooth_manager 		= 'blueman-manager',

		-- Power manager
		power_manager 			= 'xfce4-power-manager',

		-- GUI Package manager
		package_manager 		= 'pamac-manager',

		-- Lockscreen
		lock 					= 'awesome-client "awesome.emit_signal(\'module::lockscreen_show\')"',
    
    -- Quick sidebar application launcher
    rofi_global				= 'rofi -dpi ' .. screen.primary.dpi .. 
                  ' -show drun -theme ' .. config_dir ..
                  '/configuration/rofi/appmenu/rofi.rasi',

		-- You can add more default applications here
	},
	
	-- List of apps to start once on start-up
	-- auto-start.lua module will start these

	run_on_start_up = {

		-- Compositor
		'picom -b --experimental-backends --dbus --config ' ..
		config_dir .. '/configuration/picom.conf',

		-- Bluetooth tray icon
		'blueman-applet',

		-- Power manager
		'xfce4-power-manager',

		-- Credential manager
		'/usr/lib/polkit-kde-authentication-agent-1 & eval $(gnome-keyring-daemon -s --components=pkcs11,secrets,ssh,gpg)',
		
		-- Load X Colors
		'xrdb $HOME/.Xresources',

		-- NetworkManager Applet
		'nm-applet',

		-- Background image
		'nitrogen --restore',

    -- Night colors
    'redshift -l 40.0149:-105.27055',

		-- -- Audio Equalizer
		-- 'pulseeffects --gapplication-service',

		-- You can add more start-up applications here
	},

	-- List of binaries/shell scripts that will execute a certain task
	bins = {
    -- add some binary scripts here
	}
}
