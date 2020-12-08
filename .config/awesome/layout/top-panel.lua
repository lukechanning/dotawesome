local awful = require('awful')
local beautiful = require('beautiful')
local wibox = require('wibox')
local gears = require('gears')
local icons = require('theme.icons')
local dpi = beautiful.xresources.apply_dpi
local clickable_container = require('widget.clickable-container')
local task_list = require('widget.task-list')
local tag_list = require('widget.tag-list')

local top_panel = function(s, offset)

	local offsetx = 0
	if offset == true then
		offsetx = 0 
	end

	local panel = wibox
	{
		ontop = true,
		screen = s,
		type = 'dock',
		height = dpi(28),
		width = s.geometry.width - offsetx,
		x = s.geometry.x + offsetx,
		y = s.geometry.y,
		stretch = false,
		bg = beautiful.background,
		fg = beautiful.fg_normal
	}

	panel:struts
	{
		top = dpi(28)
	}

	panel:connect_signal(
		'mouse::enter',
		function() 
			local w = mouse.current_wibox
			if w then
				w.cursor = 'left_ptr'
			end
		end
	)

	s.systray = wibox.widget {
		visible = false,
		base_size = dpi(20),
		horizontal = true,
		screen = 'primary',
		widget = wibox.widget.systray
	}

  local build_widget = function(widget)
		return wibox.widget {
			{
				widget,
				border_width = dpi(1),
        		border_color = beautiful.groups_title_bg,
				bg = beautiful.transparent,
				shape = function(cr, w, h)
					gears.shape.rounded_rect(cr, w, h, dpi(12))
				end,
				widget = wibox.container.background
			},
			top = dpi(2),
			bottom = dpi(2),
			widget = wibox.container.margin
		}
	end

	local clock 			  = require('widget.clock')(s)
	local layout_box 		= require('widget.layoutbox')(s)
	local add_button 		= require('widget.open-default-app')(s)
  s.dashboard_toggle  = require('widget.dashboard-toggle')()
	s.tray_toggler  		= require('widget.tray-toggle')
	s.updater 			    = require('widget.package-updater')()
	s.battery     			= require('widget.battery')()
	s.network       		= require('widget.network')()
  s.volume            = require('widget.volume-icon')()
	s.info_center_toggle		= require('widget.info-center-toggle')()

	panel : setup {
		layout = wibox.layout.align.horizontal,
		expand = 'none',
		{
			layout = wibox.layout.fixed.horizontal,
      spacing = dpi(5),
      s.dashboard_toggle,
      build_widget(tag_list(s)),
			task_list(s),
			add_button
		}, 
		nil,
		{
			layout = wibox.layout.fixed.horizontal,
			spacing = dpi(5),
			{
				s.systray,
				margins = dpi(5),
				widget = wibox.container.margin
			},
			s.tray_toggler,
			s.updater,
      s.volume,
			s.network,
			s.battery,
			layout_box,
      clock,
			s.info_center_toggle
		}
	}

	return panel
end


return top_panel
