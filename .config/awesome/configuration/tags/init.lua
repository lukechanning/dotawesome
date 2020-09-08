local awful = require('awful')
local gears = require('gears')
local beautiful = require('beautiful')
local icons = require('theme.icons')

local tags = {
  -- one
	{
		icon = icons.web_browser,
		type = 'chrome',
		default_app = 'chromium',
		screen = 1
	},
  -- two
	{
		icon = icons.terminal,
		type = 'terminal',
		default_app = 'kitty',
		screen = 1
	},
  -- three
  {
		icon = icons.multimedia,
		type = 'music',
		default_app = 'vlc',
		screen = 1
	},
  -- four
	{
		icon = icons.file_manager,
		type = 'files',
		default_app = 'thunar',
		screen = 1
	},
  -- five
  {
		icon = icons.text_editor,
		type = 'code',
		default_app = 'kate',
		screen = 1
	},
  -- six
	{
		icon = icons.games,
		type = 'game',
		default_app = 'steam',
		screen = 1
	},
  -- seven
	{
		icon = icons.graphics,
		type = 'art',
		default_app = 'gimp-2.10',
		screen = 1
	},
  -- eight
	{
		icon = icons.sandbox,
		type = 'virtualbox',
		default_app = 'virtualbox',
		screen = 1
	},
  -- nine
	{
		icon = icons.sandbox,
		type = 'any',
		default_app = '',
		screen = 1
	}
}

tag.connect_signal(
	'request::default_layouts',
	function()
	    awful.layout.append_default_layouts({
			awful.layout.suit.tile,
      awful.layout.suit.floating,
			awful.layout.suit.spiral.dwindle,
			awful.layout.suit.max
	    })
	end
)

screen.connect_signal(
	'request::desktop_decoration',
	function(s)
		for i, tag in pairs(tags) do
			awful.tag.add(
				i,
				{
					icon = tag.icon,
					icon_only = true,
					layout = awful.layout.suit.tile,
					gap_single_client = false,
					gap = beautiful.useless_gap,
					screen = s,
					default_app = tag.default_app,
					selected = i == 1
				}
			)
		end
	end
)

tag.connect_signal(
	'property::layout',
	function(t)
		local currentLayout = awful.tag.getproperty(t, 'layout')
		if (currentLayout == awful.layout.suit.max) then
			t.gap = 0
		else
			t.gap = beautiful.useless_gap
		end
	end
)
