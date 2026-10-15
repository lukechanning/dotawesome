import importlib.util
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome.manager import Manager, Error


class DeploymentTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.home = self.root / 'home'
        self.home.mkdir()
        self.manager = Manager(ROOT, self.home, self.root / 'build', runner=lambda *a, **k: None)
        self.files = {self.home / '.config/example': (b'new', 0o644)}

    def test_check_and_dry_run_leave_home_and_build_untouched(self):
        before = list(self.home.rglob('*'))
        self.manager.preview('niri', self.files)
        self.assertEqual(before, list(self.home.rglob('*')))
        self.assertFalse(self.manager.build_root.exists())

    def test_repeat_activation_preserves_original_backup(self):
        p = next(iter(self.files)); p.parent.mkdir(); p.write_text('original')
        self.manager.deploy('niri', self.files, 'first')
        self.manager.deploy('niri', {p: (b'updated', 0o644)}, 'second')
        self.manager.deactivate('niri')
        self.assertEqual(p.read_text(), 'original')

    def test_modified_file_refuses_update_and_deactivation(self):
        self.manager.deploy('niri', self.files, 'one')
        p = next(iter(self.files)); p.write_text('manual')
        for operation in [lambda: self.manager.deploy('niri', self.files, 'two'), lambda: self.manager.deactivate('niri')]:
            with self.assertRaisesRegex(Error, 'modified'): operation()
        self.assertEqual(p.read_text(), 'manual')

    def test_component_ownership_isolation_and_overlap(self):
        self.manager.deploy('niri', self.files, 'one')
        with self.assertRaisesRegex(Error, 'owned'): self.manager.deploy('test-only', self.files, 'two')
        other = self.home / '.config/other'
        self.manager.deploy('test-only', {other: (b'other', 0o644)}, 'two')
        self.manager.deactivate('niri')
        self.assertEqual(other.read_bytes(), b'other')

    def test_reload_failure_rolls_back_files_and_records(self):
        p = next(iter(self.files)); p.parent.mkdir(); p.write_text('original')
        self.manager.runner = lambda *a, **k: (_ for _ in ()).throw(Error('reload failed'))
        with self.assertRaisesRegex(Error, 'reload failed'): self.manager.deploy('niri', self.files, 'one')
        self.assertEqual(p.read_text(), 'original')
        self.assertEqual(self.manager.records(), {})

    def test_mid_copy_failure_restores_completed_changes(self):
        p = next(iter(self.files)); p.parent.mkdir(); p.write_text('original')
        second = self.home / '.config/second'
        real = self.manager.write
        def fail(path, value):
            if path == second and value is not None: raise OSError('disk full')
            real(path, value)
        with patch.object(self.manager, 'write', side_effect=fail):
            with self.assertRaisesRegex(OSError, 'disk full'):
                self.manager.deploy('niri', {**self.files, second: (b'two', 0o644)}, 'one')
        self.assertEqual(p.read_text(), 'original')
        self.assertFalse(second.exists())

    def test_symlink_original_restored_without_touching_target(self):
        p = next(iter(self.files)); p.parent.mkdir(); target = self.home / 'target'; target.write_text('safe'); p.symlink_to(target)
        self.manager.deploy('niri', self.files, 'one'); self.manager.deactivate('niri')
        self.assertTrue(p.is_symlink()); self.assertEqual(target.read_text(), 'safe')

    def test_directories_and_symlink_parents_refused(self):
        p = next(iter(self.files)); p.parent.mkdir(); p.mkdir()
        with self.assertRaises(Error): self.manager.preview('niri', self.files)
        with self.assertRaises(Error): self.manager.preview('niri', {p / 'nested': (b'x', 0o644), p: (b'x', 0o644)})
        link = self.home / 'link'; link.symlink_to(self.root, target_is_directory=True)
        with self.assertRaisesRegex(Error, 'symlink'): self.manager.preview('niri', {link / 'outside': (b'x', 0o644)})

    def test_snapshot_deterministic_and_tampering_detected(self):
        first = self.manager.snapshot('niri', self.files)
        self.assertEqual(first, self.manager.snapshot('niri', self.files))
        f = first / '0'; f.chmod(0o644); f.write_text('tampered')
        with self.assertRaisesRegex(Error, 'snapshot'): self.manager.snapshot('niri', self.files)

    def test_cli_unknown_component_and_required_component(self):
        for args in [('build', 'awesome'), ('activate',), ('deactivate', 'everything')]:
            r = subprocess.run([str(ROOT / 'bin/dotawesome'), *args], capture_output=True, text=True)
            self.assertNotEqual(r.returncode, 0)


if __name__ == '__main__': unittest.main()

class RunnerTests(unittest.TestCase):
    def test_successful_command_preserves_stderr_diagnostics(self):
        from dotawesome.manager import run
        self.assertIn('WARN', run([sys.executable, '-c', 'import sys; print("WARN unknown setting", file=sys.stderr)']))
