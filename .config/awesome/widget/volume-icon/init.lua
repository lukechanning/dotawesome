local awful = require('awful')
local wibox = require('wibox')
local awful = require('awful')
local gears = require('gears')
local naughty = require('naughty')
local watch = awful.widget.watch
local apps = require('configuration.apps')
local clickable_container = require('widget.clickable-container')
local dpi = require('beautiful').xresources.apply_dpi
local config_dir = gears.filesystem.get_configuration_dir()
local widget_icon_dir = config_dir .. 'widget/volume-icon/icons/'

local return_button = function()

  local volume_icon_name="audio-volume-high"

	local volume_imagebox = wibox.widget {
		nil,
		{
			id = 'icon',
			image = widget_icon_dir .. volume_icon_name .. '.svg',
			widget = wibox.widget.imagebox,
			resize = true
		},
		nil,
		expand = 'none',
		layout = wibox.layout.align.vertical
	}

	local battery_percentage_text = wibox.widget {
		id = 'percent_text',
		text = '100%',
		font = 'Inter Bold 11',
		align = 'center',
		valign = 'center',
		visible = false,
		widget = wibox.widget.textbox
	}

  local function set_icon(name)
		volume_imagebox.icon:set_image(gears.surface.load_uncached(widget_icon_dir .. volume_icon_name .. '.svg'))
    return
  end

  local function parse_volume(stdout)
    local level = string.match(stdout, "(%d?%d?%d)%%")
    if stdout:find("%[off%]") then
      volume_icon_name="audio-volume-muted"
      return set_icon(volume_icon_name)
    end
    level = tonumber(string.format("% 3d", level))

    if (level >= 0 and level < 50) then
        volume_icon_name="audio-volume-muted-symbolic"
    elseif (level < 75) then
        volume_icon_name="audio-volume-medium"
    else
        volume_icon_name="audio-volume-high"
    end
    return set_icon(volume_icon_name)
  end

	local video_widget = wibox.widget {
		layout = wibox.layout.fixed.horizontal,
		spacing = dpi(0),
		volume_imagebox,
		battery_percentage_text
	}

	local volume_icon = wibox.widget {
		{
			video_widget,
			margins = dpi(7),
			widget = wibox.container.margin
		},
		widget = clickable_container
	}
	volume_icon:buttons(
		gears.table.join(
			awful.button(
				{},
				1,
				nil,
				function()
					awful.spawn(apps.default.power_manager , false)
				end
			)
		)
	)

  -- watch audio signal
	watch(
		[[sh -c "
      amixer sget Master
		"]],
		5,
		function(widget, stdout)
			return parse_volume(stdout)
		end
	)

	return volume_icon
end

return return_button


