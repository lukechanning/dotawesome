"""Shared configuration ownership and secure identity recovery."""
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome import shared, niri
from dotawesome.manager import Manager
from dotawesome.components import COMPONENTS


class SharedTests(unittest.TestCase):
    def test_ownership_rejects_component_targets_and_ancestors(self):
        for path in ('.config/niri/config.kdl', '.config/nvim/init.lua', '.tmux.conf',
                     '.config/dotawesome/niri/profile.json', '.config/systemd/user/niri.service.d', '.config'):
            with self.subTest(path=path), self.assertRaises(ValueError):
                shared.check_ownership([path])
        shared.check_ownership(['.gitconfig', '.aws/config', '.config/fish/functions/vpn.fish'])

    def test_boundary_covers_every_component_rendered_destination(self):
        with tempfile.TemporaryDirectory() as tmp:
            manager = Manager(ROOT, Path(tmp))
            for name, component in COMPONENTS.items():
                profile = niri.profile(ROOT, 'g14') if name == 'niri' else None
                for destination in component.render(manager, profile):
                    with self.subTest(destination=destination), self.assertRaises(ValueError):
                        shared.check_ownership([str(destination.relative_to(manager.home))])

    def test_owned_directories_rejected_but_shared_parents_allowed(self):
        shared.check_ownership([], directories=['.config', '.aws', '.config/fish'])
        for name in ('.config/niri', '.config/nvim', '.config/dotawesome/niri'):
            with self.assertRaises(ValueError):
                shared.check_ownership([], directories=[name])

    def test_desktop_uses_shared_apps_without_laptop_hardware(self):
        with tempfile.TemporaryDirectory() as tmp:
            manager = Manager(ROOT, Path(tmp))
            profile = niri.profile(ROOT, 'desktop')
            self.assertEqual(profile['apps'], niri.profile(ROOT, 'g14')['apps'])
            files = niri.render(manager, profile)
            text = files[manager.config / 'niri/config.kdl'][0].decode()
            for forbidden in ('eDP-', '/dev/dri', 'render-drm-device', '/home/luke', 'Path.jpg'):
                self.assertNotIn(forbidden, text)
            self.assertNotIn('laptop', profile)
            self.assertNotIn('wallpaper', profile)

    def test_recovery_validates_before_publishing_and_preserves_existing(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            key = root / 'original'
            subprocess.run(['age-keygen', '-o', str(key)], check=True, capture_output=True)
            recipient = subprocess.check_output(['age-keygen', '-y', str(key)], text=True).strip()
            target = root / 'local/key.txt'
            modes = []
            def decrypt(command, **kwargs):
                modes.append(os.fstat(kwargs['stdout'].fileno()).st_mode & 0o777)
                kwargs['stdout'].write(key.read_bytes())
            shared.recover_identity(target, root / 'recovery.age', recipient, runner=decrypt)
            self.assertEqual(modes, [0o600])
            self.assertEqual(target.stat().st_mode & 0o777, 0o600)
            shared.recover_identity(target, root / 'recovery.age', recipient, runner=lambda *a, **k: self.fail('repeat decrypt'))
            before = target.read_bytes()
            with self.assertRaises(ValueError):
                shared.recover_identity(target, root / 'recovery.age', 'age1wrong', runner=decrypt)
            self.assertEqual(target.read_bytes(), before)

    def test_failed_or_interrupted_recovery_leaves_no_identity(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            target = root / 'local/key.txt'
            for failure in (subprocess.CalledProcessError(1, 'age'), KeyboardInterrupt(), None):
                def decrypt(command, **kwargs):
                    kwargs['stdout'].write(b'invalid partial identity')
                    if failure:
                        raise failure
                with self.assertRaises((subprocess.CalledProcessError, KeyboardInterrupt, ValueError)):
                    shared.recover_identity(target, root / 'recovery.age', 'age1wrong', runner=decrypt)
                self.assertFalse(target.exists())
                self.assertEqual(list(target.parent.iterdir()), [])
