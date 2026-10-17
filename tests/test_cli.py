"""Command boundaries: every fake live path is under a temporary home."""
import contextlib
import io
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome import cli, niri
from dotawesome.manager import Manager, Error, encode


class CLITests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(); self.addCleanup(self.temp.cleanup)
        self.home = Path(self.temp.name) / 'home'; self.home.mkdir()
        self.calls = []
        def runner(args, **kw): self.calls.append(args); return ''
        self.manager = Manager(ROOT, self.home, Path(self.temp.name) / 'build', runner=runner)

    def invoke(self, *args):
        out = io.StringIO()
        with patch.object(sys, 'argv', ['dotawesome', *args]), patch.object(cli, 'Manager', return_value=self.manager), contextlib.redirect_stdout(out), contextlib.redirect_stderr(out):
            code = cli.main(ROOT)
        return code, out.getvalue()

    def test_check_dry_run_and_missing_dependency_activation_are_read_only(self):
        with patch.object(niri, 'check', return_value={'errors': ['missing niri'], 'warnings': [], 'packages': ['niri']}):
            self.assertEqual(self.invoke('check', 'niri', '--profile', 'generic')[0], 1)
            code, output = self.invoke('activate', 'niri', '--profile', 'generic', '--dry-run', '--install-packages')
            self.assertEqual(code, 0); self.assertIn('sudo pacman -Syu --needed niri', output)
            with patch.object(niri, 'outside_session'):
                self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic')[0], 1)
        self.assertEqual(list(self.home.rglob('*')), [])
        self.assertFalse(self.manager.build_root.exists()); self.assertEqual(self.calls, [])

    def test_activation_validates_before_deploy_and_only_reloads_units(self):
        with patch.object(niri, 'check', return_value={'errors': [], 'warnings': [], 'packages': []}), patch.object(niri, 'outside_session'), patch.object(niri, 'validate', return_value=True) as validator:
            self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic')[0], 0)
            validator.assert_called_once()
        self.assertEqual(self.calls, [['systemctl', '--user', 'daemon-reload']])
        self.assertIn('niri', self.manager.records())
        self.assertEqual(self.invoke('status', 'niri')[0], 0)
        with patch.object(niri, 'outside_session'):
            self.assertEqual(self.invoke('deactivate', 'niri')[0], 0)
        self.assertFalse((self.manager.config / 'niri/config.kdl').exists())

    def test_failed_native_validation_never_deploys(self):
        with patch.object(niri, 'check', return_value={'errors': [], 'warnings': [], 'packages': []}), patch.object(niri, 'outside_session'), patch.object(niri, 'validate', side_effect=Error('bad config')):
            self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic')[0], 1)
        self.assertEqual(list(self.home.rglob('*')), []); self.assertEqual(self.calls, [])

    def test_package_install_uses_full_arch_upgrade_and_rechecks(self):
        result = type('Result', (), {'returncode': 0})()
        with patch.object(niri, 'check', return_value={'errors': [], 'warnings': [], 'packages': ['niri', 'noctalia']}) as check, patch.object(niri, 'outside_session'), patch.object(niri, 'validate', return_value=True), patch.object(cli.subprocess, 'run', return_value=result) as run:
            self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic', '--install-packages')[0], 0)
            run.assert_called_once_with(['sudo', 'pacman', '-Syu', '--needed', 'niri', 'noctalia'])
            self.assertEqual(check.call_count, 2)

    def test_deactivation_refuses_live_niri(self):
        with patch.dict(os.environ, {'NIRI_SOCKET': '/tmp/example'}):
            self.assertEqual(self.invoke('deactivate', 'niri')[0], 1)
        self.assertFalse(self.manager.state.exists())

    def test_live_activation_keeps_session_running_and_validates_first(self):
        files = niri.render(self.manager, niri.profile(ROOT, 'generic'))
        self.manager.deploy('niri', files, 'initial')
        self.calls.clear()
        with patch.dict(os.environ, {'NIRI_SOCKET': '/tmp/example'}), patch.object(niri, 'check', return_value={'errors': [], 'warnings': [], 'packages': []}), patch.object(niri, 'validate', return_value=True) as validator, patch.object(niri, 'outside_session') as logout, patch('dotawesome.cli.Packages.require') as requirements:
            code, output = self.invoke('activate', 'niri', '--profile', 'generic', '--live')
            self.assertEqual(code, 0, output)
            validator.assert_called_once()
            logout.assert_not_called()
            requirements.assert_not_called()
        self.assertEqual(self.calls, [['systemctl', '--user', 'daemon-reload']])
        self.assertIn('live', output.lower())

    def test_live_activation_refuses_packages_without_any_changes(self):
        with patch('dotawesome.cli.Packages.install') as install:
            code, output = self.invoke('activate', 'niri', '--profile', 'generic', '--live', '--install-packages')
        self.assertEqual(code, 1, output)
        install.assert_not_called()
        self.assertFalse(self.manager.state.exists())

    def test_interrupted_transaction_restores_original_before_next_mutation(self):
        p = self.home / 'original'; p.write_text('original')
        original = self.manager.inspect(p)
        self.manager.state.mkdir(parents=True)
        after = encode(b'new', 0o644); self.manager.write(p, after)
        self.manager.save_json(self.manager.state / 'transaction.json', {'records': {}, 'changes': {str(p): {'before': original, 'after': after}}})
        self.manager.deactivate('niri')
        self.assertEqual(p.read_text(), 'original')
        self.assertFalse((self.manager.state / 'transaction.json').exists())

    def test_interrupted_transaction_refuses_post_crash_edits(self):
        p = self.home / 'original'; p.write_text('edited after crash')
        self.manager.state.mkdir(parents=True)
        self.manager.save_json(self.manager.state / 'transaction.json', {'records': {}, 'changes': {str(p): {'before': None, 'after': encode(b'new', 0o644)}}})
        with self.assertRaisesRegex(Error, 'unexpected edit'): self.manager.deactivate('niri')
        self.assertEqual(p.read_text(), 'edited after crash')

    def test_status_reports_pending_checkout_changes(self):
        with patch.object(niri, 'check', return_value={'errors': [], 'warnings': [], 'packages': []}), patch.object(niri, 'outside_session'), patch.object(niri, 'validate', return_value=True):
            self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic')[0], 0)
        original = niri.render
        def changed(manager, profile):
            files = original(manager, profile)
            p = manager.config / 'niri/config.kdl'; files[p] = (b'changed config', 0o644)
            return files
        with patch.object(niri, 'render', side_effect=changed):
            self.assertIn('checkout differs', self.invoke('status', 'niri')[1])

    def test_activate_recovers_interrupted_update_before_edit_check(self):
        with patch.object(niri, 'check', return_value={'errors': [], 'warnings': [], 'packages': []}), patch.object(niri, 'outside_session'), patch.object(niri, 'validate', return_value=True):
            self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic')[0], 0)
            p = self.manager.config / 'niri/config.kdl'; before = self.manager.inspect(p); after = encode(b'interrupted update', 0o644)
            records = self.manager.records()
            self.manager.save_json(self.manager.state / 'transaction.json', {'records': records, 'changes': {str(p): {'before': before, 'after': after}}})
            self.manager.write(p, after)
            self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic')[0], 0)
            self.assertEqual(self.manager.inspect(p), before)

    def test_deactivation_package_dry_run_does_not_write_or_execute(self):
        self.manager.deploy('niri', {self.home / 'config': (b'managed', 0o644)}, 'snapshot')
        before = {p: p.read_bytes() for p in self.home.rglob('*') if p.is_file()}
        self.calls.clear()
        with patch('dotawesome.cli.Packages.plan', return_value=['niri', 'noctalia']), patch.object(cli.subprocess, 'run') as execute:
            code, output = self.invoke('deactivate', 'niri', '--remove-packages', '--dry-run')
        self.assertEqual(code, 0)
        self.assertIn('sudo pacman -Ru niri noctalia', output)
        self.assertEqual(before, {p: p.read_bytes() for p in self.home.rglob('*') if p.is_file()})
        execute.assert_not_called(); self.assertEqual(self.calls, [])

    def test_package_removal_happens_after_config_restoration(self):
        self.manager.deploy('niri', {self.home / 'config': (b'managed', 0o644)}, 'snapshot')
        def remove(component):
            self.assertNotIn(component, self.manager.records())
            self.assertFalse((self.home / 'config').exists())
            return ['niri']
        with patch('dotawesome.cli.Packages.plan', return_value=['niri']), patch('dotawesome.cli.Packages.remove', side_effect=remove) as cleanup, patch.object(niri, 'outside_session'):
            self.assertEqual(self.invoke('deactivate', 'niri', '--remove-packages')[0], 0)
            cleanup.assert_called_once_with('niri')

    def test_package_provenance_survives_config_validation_failure(self):
        with patch.object(niri, 'check', return_value={'errors': [], 'warnings': [], 'packages': ['niri']}), patch.object(niri, 'outside_session'), patch.object(niri, 'validate', side_effect=Error('bad config')), patch('dotawesome.cli.Packages.install') as install:
            self.assertEqual(self.invoke('activate', 'niri', '--profile', 'generic', '--install-packages')[0], 1)
            install.assert_called_once_with('niri', ['niri'])
        with patch('dotawesome.cli.Packages.plan', return_value=['niri']), patch('dotawesome.cli.Packages.remove', return_value=['niri']) as remove, patch.object(niri, 'outside_session'):
            self.assertEqual(self.invoke('deactivate', 'niri', '--remove-packages')[0], 0)
            remove.assert_called_once()
