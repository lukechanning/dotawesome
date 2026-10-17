"""Desktop contract: requested daily controls, no extra desktop services."""
from pathlib import Path
import re
import sys
import tempfile
import tomllib
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome.manager import Manager
from dotawesome import niri


class LeanDesktopTests(unittest.TestCase):
    def setUp(self):
        self.shell = tomllib.loads((ROOT / 'components/niri/noctalia.toml').read_text())
        self.base = (ROOT / 'components/niri/base.kdl').read_text()

    def test_daily_controls_are_visible_including_bluetooth(self):
        end = self.shell['bar']['default']['end']
        for widget in ('network', 'battery', 'cpu', 'ram', 'notifications', 'clipboard', 'tray', 'volume', 'bluetooth'):
            self.assertIn(widget, end)
        self.assertEqual(self.shell['widget']['bluetooth']['actions']['left'], 'panel-open control-center bluetooth')
        self.assertNotIn('taskbar', end)
        self.assertEqual(self.shell['control_center']['hidden_tabs'], ['weather', 'calendar', 'screen-time'])
        self.assertEqual(self.shell['widget']['network']['actions']['left'], 'panel-open control-center network')
        self.assertTrue(self.shell['widget']['notifications']['hide_when_no_unread'])

    def test_battery_toggles_caffeine_and_screenshot_has_keyboard_shortcut(self):
        self.assertEqual(self.shell['widget']['battery']['actions']['middle'], 'caffeine-toggle')
        self.assertIn('caffeine', self.shell['bar']['default']['end'])
        with tempfile.TemporaryDirectory() as tmp:
            m = Manager(ROOT, Path(tmp) / 'home')
            config = niri.render(m, niri.profile(ROOT, 'generic'))[m.config / 'niri/config.kdl'][0].decode()
        self.assertRegex(config, r'Mod\+Shift\+S\s*\{\s*screenshot;\s*\}')
        self.assertTrue(self.shell['lockscreen']['lock_before_suspend'])

    def test_world_clocks_expand_on_hover_without_a_plugin(self):
        groups = self.shell['bar']['default']['capsule_group']
        clocks = next(g for g in groups if g['id'] == 'world-clocks')
        self.assertTrue(clocks['accordion'])
        self.assertIn('group:world-clocks', self.shell['bar']['default']['center'])
        zones = [self.shell['widget'][name]['timezone'] for name in clocks['members']]
        self.assertEqual(zones, ['Europe/Amsterdam', 'America/New_York', 'America/Los_Angeles', 'Pacific/Auckland'])
        self.assertEqual(self.shell['plugins']['enabled'], [])
        self.assertEqual(self.shell['plugins']['auto_update'], 'none')

    def test_media_is_available_in_control_center_without_bar_controls(self):
        end = self.shell['bar']['default']['end']
        for widget in ('media-prev', 'media-play', 'media-next', 'media'):
            self.assertNotIn(widget, end)
        self.assertNotIn('media', self.shell['control_center']['hidden_tabs'])

    def test_glass_is_shell_only_and_statistics_avoid_extra_polling(self):
        self.assertGreater(self.shell['shell']['corner_radius_scale'], 0)
        self.assertTrue(self.shell['shell']['animation']['enabled'])
        self.assertTrue(self.shell['bar']['default']['compositor_blur'])
        self.assertLess(self.shell['bar']['default']['background_opacity'], 1.0)
        self.assertEqual(self.shell['shell']['panel']['transparency_mode'], 'glass')
        self.assertTrue(self.shell['shell']['panel']['shadow'])
        self.assertNotIn('opacity ', self.base)
        self.assertNotIn('blur false', self.base)
        # A forced xray effect on the full-screen panel input layer hides apps.
        self.assertNotIn('background-effect { xray true;', self.base)
        self.assertNotIn('animations { off; }', self.base)
        for stat in ('cpu', 'ram'):
            self.assertEqual(self.shell['widget'][stat]['visualization'], 'none')
        monitor = self.shell['system']['monitor']
        self.assertEqual(monitor['gpu_poll_seconds'], 0)
        self.assertEqual(monitor['disk_poll_seconds'], 0)
        self.assertEqual(monitor['network_poll_seconds'], 0)
        self.assertGreaterEqual(monitor['cpu_poll_seconds'], 3)
        self.assertGreaterEqual(monitor['memory_poll_seconds'], 3)

    def test_native_switcher_and_work_layout_have_explicit_bindings(self):
        self.assertIn('recent-windows {', self.base)
        self.assertRegex(self.base, r'Alt\+Tab\s*\{ next-window; \}')
        self.assertRegex(self.base, r'Alt\+Shift\+Tab\s*\{ previous-window; \}')
        with tempfile.TemporaryDirectory() as tmp:
            m = Manager(ROOT, Path(tmp) / 'home')
            config = niri.render(m, niri.profile(ROOT, 'g14'))[m.config / 'niri/config.kdl'][0].decode()
        self.assertRegex(config, r'Mod\+Shift\+Return\s*\{[^}]*work.py')
        self.assertRegex(config, r'Mod\+N\s*\{[^}]*"notifications"')
        self.assertRegex(config, r'Mod\+B\s*\{[^}]*"network"')
