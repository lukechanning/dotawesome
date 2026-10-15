"""Package lifecycle tests use a fake pacman and temporary private state."""
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome.manager import Manager, Error
from dotawesome.packages import Packages


class PackageTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.installed = {'plasma', 'alacritty', 'old-dependency'}
        self.removable = {'niri', 'noctalia', 'new-dependency', 'old-dependency'}
        self.calls = []
        def query(args, **kwargs):
            self.calls.append(args)
            if args == ['systemctl', '--user', 'daemon-reload']: return ''
            if args == ['pacman', '-Qq']: return '\n'.join(sorted(self.installed))
            if args[:3] == ['pacman', '-Rsu', '--print-format']: return '\n'.join(sorted(self.removable & self.installed))
            raise AssertionError(args)
        self.manager = Manager(ROOT, Path(self.tmp.name) / 'home', runner=query)
        self.packages = Packages(self.manager)
        guard = patch('dotawesome.packages.subprocess.run', side_effect=AssertionError('unmocked package mutation'))
        guard.start(); self.addCleanup(guard.stop)

    def install(self, fail=False):
        def execute(args):
            self.calls.append(args)
            self.installed.update({'niri', 'noctalia', 'new-dependency', 'upgrade-addition'})
            return type('Result', (), {'returncode': int(fail)})()
        with patch('dotawesome.packages.subprocess.run', side_effect=execute):
            self.packages.install('niri', ['niri', 'noctalia', 'alacritty'])

    def test_tracks_additions_and_preserves_preexisting_and_upgrade_packages(self):
        self.install()
        plan = self.packages.plan('niri')
        self.assertEqual(plan, ['new-dependency', 'niri', 'noctalia'])
        self.assertNotIn('old-dependency', plan)
        self.assertNotIn('upgrade-addition', plan)
        self.assertIn(['sudo', 'pacman', '-Syu', '--needed', 'niri', 'noctalia', 'alacritty'], self.calls)

    def test_repeat_install_keeps_first_ownership(self):
        self.install(); self.install()
        self.assertIn('niri', self.packages.plan('niri'))

    def test_failure_retains_package_record_for_later_cleanup(self):
        with self.assertRaises(Error): self.install(fail=True)
        self.assertIn('niri', self.packages.plan('niri'))

    def test_plan_is_read_only_with_no_provenance(self):
        self.assertEqual(self.packages.plan('niri'), [])
        self.assertFalse(self.manager.state.exists())
        self.assertEqual(self.calls, [])

    def test_shared_root_is_kept_when_pacman_marks_it_required(self):
        self.install(); self.removable = {'noctalia'}
        self.assertEqual(self.packages.plan('niri'), ['noctalia'])

    def test_removal_uses_only_tracked_names_without_recursive_or_force_flags(self):
        self.install()
        def execute(args):
            self.calls.append(args); self.installed.difference_update(args[3:])
            return type('Result', (), {'returncode': 0})()
        with patch('dotawesome.packages.subprocess.run', side_effect=execute):
            self.packages.remove('niri')
        self.assertIn(['sudo', 'pacman', '-Ru', 'new-dependency', 'niri', 'noctalia'], self.calls)
        self.assertIn('old-dependency', self.installed)
        self.assertIn('upgrade-addition', self.installed)
        self.assertEqual(self.packages.plan('niri'), [])

    def test_failed_or_cancelled_removal_can_be_retried(self):
        self.install()
        with patch('dotawesome.packages.subprocess.run', return_value=type('Result', (), {'returncode': 1})()):
            with self.assertRaises(Error): self.packages.remove('niri')
        self.assertIn('niri', self.packages.plan('niri'))

    def test_other_component_requirements_protected(self):
        self.install()
        self.packages.require('other', ['noctalia'])
        self.assertNotIn('noctalia', self.packages.plan('niri'))

    def test_interrupted_install_is_not_retroactively_attributed(self):
        self.manager.state.mkdir(parents=True)
        self.manager.save_json(self.manager.state / 'packages.json', {'niri': {'required': ['niri'], 'added': [], 'roots': [], 'pending': True}})
        with self.assertRaisesRegex(Error, 'interrupted'): self.packages.plan('niri')

    def test_removed_package_is_not_claimed_if_reinstalled_later(self):
        self.install()
        def execute(args):
            self.installed.difference_update(args[3:])
            return type('Result', (), {'returncode': 0})()
        with patch('dotawesome.packages.subprocess.run', side_effect=execute):
            self.packages.remove('niri')
        self.installed.add('niri')
        self.assertEqual(self.packages.plan('niri'), [])

    def test_refuses_package_removal_if_component_reactivated(self):
        self.install()
        self.manager.deploy('niri', {self.manager.home / 'config': (b'x', 0o644)}, 'snapshot')
        with patch('dotawesome.packages.subprocess.run') as execute:
            with self.assertRaisesRegex(Error, 'deactivate'):
                self.packages.remove('niri')
            execute.assert_not_called()

    def test_no_removable_targets_handles_pacman_nothing_to_do_message(self):
        self.install()
        real = self.manager.runner
        def query(args, **kwargs):
            if '-Rsu' in args: return ' there is nothing to do'
            return real(args)
        self.manager.runner = query
        self.assertEqual(self.packages.plan('niri'), [])
