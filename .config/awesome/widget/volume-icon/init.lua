local awful = require('awful')
local wibox = require('wibox')
local awful = require('awful')
local gears = require('gears')
local watch = awful.widget.watch
local apps = require('configuration.apps')
local clickable_container = require('widget.clickable-container')
local dpi = require('beautiful').xresources.apply_dpi
local config_dir = gears.filesystem.get_configuration_dir()
local widget_icon_dir = config_dir .. 'widget/volume-icon/icons/'

local volume_icon_name="audio-volume-high"
local last_active_state="audio-volume-high"

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

local function set_icon(name)
  volume_imagebox.icon:set_image(gears.surface.load_uncached(widget_icon_dir .. volume_icon_name .. '.svg'))
  return
end

local function mute_icon()
  if volume_icon_name == "audio-volume-muted" then
   volume_icon_name = last_active_state 
  else
   last_active_state = volume_icon_name
   volume_icon_name="audio-volume-muted"
  end
  return set_icon(volume_icon_name)
end

local function parse_volume(stdout)
  local level = string.match(stdout, "(%d?%d?%d)%%")
  -- leave in place to handle init audio 
  if stdout:find("%[off%]") then
    volume_icon_name="audio-volume-muted"
    return set_icon(volume_icon_name)
  end
  level = tonumber(string.format("% 3d", level))

  if (level >= 0 and level < 50) then
      volume_icon_name="audio-volume-low"
  elseif (level < 75) then
      volume_icon_name="audio-volume-medium"
  else
      volume_icon_name="audio-volume-high"
  end
  return set_icon(volume_icon_name)
end

-- get current settings when we launch
local function _init()
  awful.spawn.easy_async_with_shell(
    [[bash -c "amixer -D pulse sget Master"]],
    function(stdout)
      parse_volume(stdout) 
    end
  )
end
_init()

local video_widget = wibox.widget {
  layout = wibox.layout.fixed.horizontal,
  spacing = dpi(0),
  volume_imagebox,
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
			  awesome.emit_signal('module::volume_osd:show', true)
      end
    )
  )
)

function volume_icon:send_mute()
  mute_icon()
end

function volume_icon:update_icon(stdout)
  parse_volume(stdout)
end

-- signal will come from global keys
awesome.connect_signal(
	'volume::icon',
	function()
    awful.spawn.easy_async_with_shell(
      [[bash -c "amixer -D pulse sget Master"]],
      function(stdout)
        volume_icon:update_icon(stdout) 
      end
    )
	end
)

-- mute signal will come from global keys
awesome.connect_signal(
	'volume::mute',
  function()
    volume_icon:send_mute()
  end
)

local return_button = function()
  return volume_icon
end

return return_button
