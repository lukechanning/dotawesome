import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome.manager import Manager, Error
from dotawesome import niri


class NiriTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(); self.addCleanup(self.temp.cleanup)
        self.manager = Manager(ROOT, Path(self.temp.name) / 'home', Path(self.temp.name) / 'build')

    def test_shared_aoc_scale_in_g14_and_desktop_without_pinning_modes(self):
        for name in ('g14', 'desktop'):
            with self.subTest(profile=name):
                files = niri.render(self.manager, niri.profile(ROOT, name))
                config = files[self.manager.config / 'niri/config.kdl'][0].decode()
                self.assertIn('output "PNP(AOC) 27B36X XI7S4HA042988" {\n    scale 0.85\n}', config)
                for key, direction in [('H', 'left'), ('J', 'down'), ('K', 'up'), ('L', 'right')]:
                    self.assertIn(f'Mod+Ctrl+{key} {{ focus-monitor-{direction}; }}', config)
                    self.assertIn(f'Mod+Ctrl+Shift+{key} {{ move-window-to-monitor-{direction}; }}', config)

    def test_shared_monitor_is_preferred_for_both_numbered_workspaces(self):
        for name in ('g14', 'desktop'):
            files = niri.render(self.manager, niri.profile(ROOT, name))
            config = files[self.manager.config / 'niri/config.kdl'][0].decode()
            for workspace in ('One', 'Two'):
                self.assertIn(f'workspace "{workspace}" {{ open-on-output "PNP(AOC) 27B36X XI7S4HA042988"; }}', config)

    def test_iso_key_matches_kde_shift_remap_in_every_profile(self):
        for name in ('generic', 'g14', 'desktop'):
            with self.subTest(profile=name):
                files = niri.render(self.manager, niri.profile(ROOT, name))
                config = files[self.manager.config / 'niri/config.kdl'][0].decode()
                self.assertIn('options "lv2:lsgt_switch"', config)

    def test_monitor_shortcuts_move_individual_windows_and_preserve_tiling_controls(self):
        files = niri.render(self.manager, niri.profile(ROOT, 'g14'))
        config = files[self.manager.config / 'niri/config.kdl'][0].decode()
        for key, direction in [('H', 'left'), ('J', 'down'), ('K', 'up'), ('L', 'right')]:
            self.assertIn(f'Mod+Ctrl+{key} {{ focus-monitor-{direction}; }}', config)
            self.assertIn(f'Mod+Ctrl+Shift+{key} {{ move-window-to-monitor-{direction}; }}', config)
        self.assertIn('Mod+Shift+H { move-column-left; }', config)
        self.assertIn('Mod+Shift+L { move-column-right; }', config)

    def test_g14_routes_secret_portal_to_existing_kde_wallet(self):
        import configparser
        files = niri.render(self.manager, niri.profile(ROOT, 'g14'))
        portal = configparser.ConfigParser()
        portal.read_string(files[self.manager.config / 'xdg-desktop-portal/niri-portals.conf'][0].decode())
        self.assertEqual(portal['preferred']['org.freedesktop.impl.portal.Secret'], 'kwallet')
        self.assertEqual(portal['preferred']['org.freedesktop.impl.portal.FileChooser'], 'gtk')

    def test_generic_and_g14_render_without_unknown_external(self):
        for name in ['generic', 'g14']:
            profile = niri.profile(ROOT, name)
            files = niri.render(self.manager, profile)
            config = files[self.manager.config / 'niri/config.kdl'][0].decode()
            self.assertIn('workspace "One"', config)
            self.assertNotIn('1920x1080', config)
            self.assertNotIn('spawn-at-startup "noctalia"', config)
            self.assertIn('dotawesome-niri-noctalia.service', '\n'.join(map(str, files)))
        self.assertIn('scale 1.6', config)
        self.assertIn('output "eDP-2"', config)

    def test_minimal_loadout_and_workspace_order(self):
        p = niri.profile(ROOT, 'g14')
        self.assertEqual([(a['name'], a['workspace'], a['width']) for a in p['apps']],
                         [('Vivaldi', 'One', '100%'), ('Alacritty', 'One', '100%')])
        files = niri.render(self.manager, p)
        config = files[self.manager.config / 'niri/config.kdl'][0].decode()
        import re
        self.assertEqual(re.findall(r'^workspace "([^\"]+)"', config, re.M), ['One', 'Two'])
        for key, name in [(1, 'One'), (2, 'Two')]:
            self.assertIn(f'Mod+{key} {{ focus-workspace "{name}";', config)
            self.assertIn(f'Mod+Shift+{key} {{ move-window-to-workspace "{name}";', config)
        self.assertNotIn('Mod+3 {', config)
        self.assertNotIn('Mod+Shift+3 {', config)

    def test_g14_pins_vivaldi_wallet_for_all_launch_paths_only_when_enabled(self):
        flags = self.manager.config / 'vivaldi-stable.conf'
        files = niri.render(self.manager, niri.profile(ROOT, 'g14'))
        self.assertEqual(files[flags][0], b'--password-store=kwallet6\n')
        self.assertNotIn(flags, niri.render(self.manager, niri.profile(ROOT, 'generic')))

    def test_g14_selects_amd_renderer_without_disabling_nvidia(self):
        p = niri.profile(ROOT, 'g14')
        config = niri.render(self.manager, p)[self.manager.config / 'niri/config.kdl'][0].decode()
        self.assertIn('render-drm-device "/dev/dri/by-path/pci-0000:65:00.0-render"', config)
        self.assertNotIn('ignore-drm-device', config)
        generic = niri.render(self.manager, niri.profile(ROOT, 'generic'))
        self.assertNotIn('render-drm-device', generic[self.manager.config / 'niri/config.kdl'][0].decode())

    def test_signal_wallet_is_optional_and_independent_of_vivaldi(self):
        flags = self.manager.config / 'signal-desktop-flags.conf'
        p = niri.profile(ROOT, 'g14')
        self.assertEqual(niri.render(self.manager, p)[flags][0], b'--password-store=kwallet6\n')
        self.assertNotIn(flags, niri.render(self.manager, niri.profile(ROOT, 'generic')))
        p.pop('vivaldi_password_store')
        files = niri.render(self.manager, p)
        self.assertIn(flags, files)
        self.assertNotIn(self.manager.config / 'vivaldi-stable.conf', files)
        p['signal_password_store'] = 'basic'
        with self.assertRaisesRegex(Error, 'signal_password_store'):
            niri.render(self.manager, p)

    def test_external_requires_captured_identity_and_advertised_mode(self):
        p = niri.profile(ROOT, 'g14')
        p['external'] = {'identity': 'HDMI-A-1', 'mode': '1920x1080@144', 'scale': .85}
        with self.assertRaises(Error): niri.render(self.manager, p)
        p['external'] = {'identity': 'Acme Work ABC123', 'mode': '1920x1080@144', 'scale': .85,
                         'verified_connections': ['dock-hdmi', 'direct-hdmi'], 'advertised_modes': ['1920x1080@144']}
        config = niri.render(self.manager, p)[self.manager.config / 'niri/config.kdl'][0].decode()
        self.assertIn('output "Acme Work ABC123"', config)
        self.assertIn('scale 0.85', config)
        self.assertIn('focus-at-startup', config)
        self.assertIn('open-on-output "Acme Work ABC123"', config)

    def test_native_validation_is_required_and_warnings_fail(self):
        p = niri.profile(ROOT, 'generic'); files = niri.render(self.manager, p)
        with patch('shutil.which', return_value=None):
            with self.assertRaisesRegex(Error, 'missing'): niri.validate(self.manager, files)
        with patch('shutil.which', return_value='/usr/bin/tool'), patch.object(self.manager, 'runner', return_value='WARN unknown setting'):
            with self.assertRaisesRegex(Error, 'warning'): niri.validate(self.manager, files)

    def test_custom_palette_is_managed_and_original_restored(self):
        import tomllib
        files = niri.render(self.manager, niri.profile(ROOT, 'g14'))
        shell_dir = self.manager.config / 'dotawesome/niri/noctalia/noctalia'
        shell = tomllib.loads(files[shell_dir / 'config.toml'][0].decode())
        self.assertEqual(shell['theme']['source'], 'custom')
        self.assertEqual(shell['theme']['custom_palette'], 'Graphite-Jade')
        palette = shell_dir / 'palettes' / (shell['theme']['custom_palette'] + '.json')
        self.assertIn(palette, files)
        dark = json.loads(files[palette][0])['dark']
        # Native Noctalia's runtime parser requires this even with templates off.
        self.assertIn('terminal', dark)
        self.assertEqual(len(dark['terminal']['normal']), 8)
        self.assertEqual(len(dark['terminal']['bright']), 8)
        palette.parent.mkdir(parents=True)
        palette.write_bytes(b'original palette')
        with patch.object(self.manager, 'runner', return_value=''):
            snapshot = self.manager.snapshot('niri', files)
            self.manager.deploy('niri', files, snapshot)
            self.assertEqual(palette.read_bytes(), files[palette][0])
            self.manager.deactivate('niri')
        self.assertEqual(palette.read_bytes(), b'original palette')

    def test_native_validation_stages_custom_palette_beside_config(self):
        import tomllib
        files = niri.render(self.manager, niri.profile(ROOT, 'generic'))
        validated = []
        def runner(command):
            if command[:3] == ['noctalia', 'config', 'validate']:
                config = Path(command[3])
                shell = tomllib.loads(config.read_text())
                self.assertEqual(shell['theme']['source'], 'custom')
                palette = config.parent / 'palettes' / (shell['theme']['custom_palette'] + '.json')
                self.assertTrue(palette.is_file())
                self.assertIn('dark', json.loads(palette.read_text()))
                validated.append(config)
            return ''
        with patch('shutil.which', return_value='/usr/bin/tool'), patch.object(self.manager, 'runner', side_effect=runner):
            niri.validate(self.manager, files)
        self.assertEqual(len(validated), 1)

    def test_preflight_missing_deps_and_optional_apps(self):
        with patch('shutil.which', return_value=None):
            report = niri.check(self.manager, niri.profile(ROOT, 'g14'))
        self.assertTrue(report['errors'])
        self.assertTrue(any('Vivaldi' in line for line in report['warnings']))

    def test_profile_rejects_invalid_command(self):
        p = niri.profile(ROOT, 'generic'); p['apps'][0]['command'] = 'alacritty; surprise'
        with self.assertRaises(Error): niri.render(self.manager, p)


if __name__ == '__main__': unittest.main()

class ShellPolicyTests(unittest.TestCase):
    def test_shell_uses_documented_stat_and_disables_transitions(self):
        import tomllib
        shell = tomllib.loads((ROOT / 'components/niri/noctalia.toml').read_text())
        self.assertEqual(shell['widget']['ram']['stat'], 'ram_pct')
        self.assertEqual(shell['lockscreen']['transition'], [])
        self.assertEqual(shell['wallpaper']['transition'], [])
        self.assertEqual(shell['widget']['media']['max_length'], 160)
        self.assertTrue(shell['widget']['media']['hide_album_art'])

class StartupAuditTests(unittest.TestCase):
    def test_system_autostart_kde_scope_and_user_hidden_override(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            m = Manager(ROOT, root / 'home', root / 'build')
            system = root / 'system/autostart'; system.mkdir(parents=True)
            user = m.config / 'autostart'; user.mkdir(parents=True)
            (system / 'polkit.desktop').write_text('[Desktop Entry]\nExec=/usr/lib/polkit-kde-authentication-agent-1\nOnlyShowIn=KDE;\n')
            (system / 'shell.desktop').write_text('[Desktop Entry]\nExec=noctalia --daemon\n')
            self.assertEqual(len(niri.startup_conflicts(m, [system.parent])), 1)
            (user / 'shell.desktop').write_text('[Desktop Entry]\nHidden=true\n')
            self.assertEqual(niri.startup_conflicts(m, [system.parent]), [])

class PrerequisiteTests(unittest.TestCase):
    def test_packaged_session_and_pam_are_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            manager = Manager(ROOT, Path(tmp) / 'home', runner=lambda *a, **k: '')
            with patch.object(Path, 'is_file', return_value=False):
                report = niri.check(manager, niri.profile(ROOT, 'generic'))
            self.assertTrue(any('packaged Niri session' in error for error in report['errors']))
            self.assertTrue(any('PAM login' in error for error in report['errors']))

class WallpaperTests(unittest.TestCase):
    def test_g14_reuses_local_kde_wallpapers_without_copying_images(self):
        import tomllib
        with tempfile.TemporaryDirectory() as tmp:
            manager = Manager(ROOT, Path(tmp) / 'home')
            files = niri.render(manager, niri.profile(ROOT, 'g14'))
            shell = tomllib.loads(files[manager.config / 'dotawesome/niri/noctalia/noctalia/config.toml'][0].decode())
            self.assertEqual(shell['wallpaper']['default']['path'], str(manager.home / 'Pictures/Path.jpg'))
            self.assertEqual([f['path'] for f in shell['wallpaper']['favorite']],
                             [str(manager.home / 'Pictures/Path.jpg'), str(manager.home / 'Pictures/OneStandsOut.jpg')])
            self.assertFalse(shell['wallpaper']['automation']['enabled'])
            self.assertFalse(shell['theme']['templates']['enable_builtin_templates'])
            self.assertFalse(any(p.suffix.lower() in ('.jpg', '.png') for p in files))

    def test_generic_retains_plain_background(self):
        import tomllib
        with tempfile.TemporaryDirectory() as tmp:
            manager = Manager(ROOT, Path(tmp) / 'home')
            files = niri.render(manager, niri.profile(ROOT, 'generic'))
            shell = tomllib.loads(files[manager.config / 'dotawesome/niri/noctalia/noctalia/config.toml'][0].decode())
            self.assertNotIn('default', shell['wallpaper'])

    def test_preflight_reports_missing_local_wallpapers(self):
        with tempfile.TemporaryDirectory() as tmp:
            manager = Manager(ROOT, Path(tmp) / 'home', runner=lambda *a, **k: '')
            with patch('shutil.which', return_value=None):
                report = niri.check(manager, niri.profile(ROOT, 'g14'))
            self.assertTrue(any('Path.jpg' in warning for warning in report['warnings']))
            self.assertTrue(any('OneStandsOut.jpg' in warning for warning in report['warnings']))
