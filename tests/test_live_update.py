"""Live deployment may change appearance, never session behaviour or user edits."""
from pathlib import Path
import json
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome import niri
from dotawesome.manager import Error, Manager


class LiveUpdateTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.manager = Manager(ROOT, Path(self.tmp.name) / 'home', runner=lambda *a: '')
        self.files = niri.render(self.manager, niri.profile(ROOT, 'g14'))
        self.manager.deploy('niri', self.files, 'previous')
        self.config = self.manager.config / 'niri/config.kdl'
        self.shell = self.manager.config / 'dotawesome/niri/noctalia/noctalia/config.toml'

    def changed(self, path, before, after):
        data, mode = self.files[path]
        self.assertIn(before.encode(), data)
        return self.files | {path: (data.replace(before.encode(), after.encode()), mode)}

    def test_live_appearance_and_palette_addition_are_allowed(self):
        files = self.changed(self.shell, 'background_opacity = 0.0', 'background_opacity = 0.1')
        data, mode = files[self.config]
        files[self.config] = (data.replace(b'geometry-corner-radius 6', b'geometry-corner-radius 10'), mode)
        niri.validate_live_update(self.manager, files)

        palette = self.shell.parent / 'palettes/Architectural-Glass.json'
        old = {p: value for p, value in self.files.items() if p != palette}
        self.manager.deploy('niri', old, 'without-palette')
        niri.validate_live_update(self.manager, files)

    def test_shell_blur_rule_can_change_without_changing_its_match(self):
        data, mode = self.files[self.config]
        rule = b'\nlayer-rule {\n match namespace="^noctalia-"\n background-effect { blur false; }\n}\n'
        old = self.files | {self.config: (data + rule, mode)}
        self.manager.deploy('niri', old, 'old-blur-policy')
        niri.validate_live_update(self.manager, self.files)

    def test_output_shortcut_launcher_and_session_changes_require_logout(self):
        cases = [
            (self.config, '2880x1800@60.001', '2880x1800@120'),
            (self.config, 'scale 1.6', 'scale 1.5'),
            (self.config, 'Mod+H', 'Mod+Left'),
            (self.config, 'lv2:lsgt_switch', 'ctrl:nocaps'),
            (self.config, '"alacritty"', '"other-terminal"'),
            (self.shell, '[calendar]\nenabled = true', '[calendar]\nenabled = false'),
            (self.shell, 'hidden_tabs = ["weather", "screen-time"]', 'hidden_tabs = ["weather", "calendar", "screen-time"]'),
            (self.shell, 'polkit_agent = true', 'polkit_agent = false'),
            (self.shell, 'enable_builtin_templates = false', 'enable_builtin_templates = true'),
            (self.shell, 'timeout = 600', 'timeout = 30'),
            (self.shell, 'panel-open control-center audio', 'session-shutdown'),
            (self.shell, 'Path.jpg', 'Different.jpg'),
            (self.manager.config / 'xdg-desktop-portal/niri-portals.conf', 'kwallet', 'gnome-keyring'),
        ]
        for path, before, after in cases:
            with self.subTest(change=before):
                with self.assertRaisesRegex(Error, 'live update'):
                    niri.validate_live_update(self.manager, self.changed(path, before, after))

    def test_quoted_command_text_is_not_treated_as_appearance_nodes(self):
        old = self.changed(self.config, '"alacritty"', '"echo" "focus-ring { active-color blue; }"')
        self.manager.deploy('niri', old, 'quoted-command')
        new = dict(old)
        data, mode = new[self.config]
        new[self.config] = (data.replace(b'active-color blue', b'active-color red'), mode)
        with self.assertRaisesRegex(Error, 'live update'):
            niri.validate_live_update(self.manager, new)

    def test_modified_files_and_first_install_are_refused(self):
        self.config.write_bytes(self.config.read_bytes() + b'// manual edit\n')
        with self.assertRaisesRegex(Error, 'modified'):
            niri.validate_live_update(self.manager, self.files)
        clean = Manager(ROOT, Path(self.tmp.name) / 'new-home')
        with self.assertRaisesRegex(Error, 'existing deployment'):
            niri.validate_live_update(clean, niri.render(clean, niri.profile(ROOT, 'g14')))

    def test_profile_metadata_may_refresh_but_work_apps_must_not_change(self):
        path = self.manager.config / 'dotawesome/niri/profile.json'
        profile = json.loads(self.files[path][0])
        profile['laptop']['mode'] = '2880x1800@120'
        metadata = self.files | {path: ((json.dumps(profile) + '\n').encode(), 0o644)}
        niri.validate_live_update(self.manager, metadata)
        profile['apps'][0]['command'] = ['another-browser']
        with self.assertRaisesRegex(Error, 'live update'):
            niri.validate_live_update(self.manager, metadata | {path: (json.dumps(profile).encode(), 0o644)})

    def test_rendered_output_change_is_refused_even_with_metadata_refresh(self):
        profile = niri.profile(ROOT, 'g14')
        profile['laptop']['scale'] = 1.5
        with self.assertRaisesRegex(Error, 'live update'):
            niri.validate_live_update(self.manager, niri.render(self.manager, profile))

    def test_recovery_and_new_service_files_are_refused(self):
        files = dict(self.files)
        files[self.manager.config / 'systemd/user/new.service'] = (b'[Service]\n', 0o644)
        with self.assertRaisesRegex(Error, 'live update'):
            niri.validate_live_update(self.manager, files)
        (self.manager.state / 'transaction.json').write_text('{}')
        with self.assertRaisesRegex(Error, 'interrupted'):
            niri.validate_live_update(self.manager, self.files)

    def test_workspace_visibility_and_group_changes_require_logout(self):
        for before, after in [
            ('hide_empty_workspaces = true', 'hide_empty_workspaces = false'),
            ('end = ["group:metrics", "group:status"]', 'end = ["group:status", "group:metrics"]'),
        ]:
            with self.subTest(change=before):
                files = self.changed(self.shell, before, after)
                with self.assertRaisesRegex(Error, 'live update'):
                    niri.validate_live_update(self.manager, files)
