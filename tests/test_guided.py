"""Guided setup uses real temporary deployments; system operations stay mocked."""
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
from dotawesome import cli, guided
from dotawesome.components import Component
from dotawesome.manager import Error, Manager


class GuidedTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.home = Path(self.tmp.name) / 'home'
        self.home.mkdir()
        self.manager = Manager(ROOT, self.home, Path(self.tmp.name) / 'build', runner=lambda *a, **k: '')
        self.installed = {'tmux'}
        self.installations = []
        self.addCleanup(patch.stopall)
        patch.object(guided.Packages, 'installed', side_effect=lambda: set(self.installed)).start()
        def install(groups):
            self.installations.append(groups)
            self.installed.update(p for group in groups.values() for p in group)
        patch.object(guided.Packages, 'install_many', side_effect=install).start()
        patch.object(Component, 'check', return_value={'errors': [], 'warnings': [], 'packages': ['tmux']}).start()
        patch.object(Component, 'validate', return_value=True).start()
        patch.object(guided, 'authenticate', return_value=[]).start()
        patch.object(guided.shutil, 'which', side_effect=lambda name: '/usr/bin/' + name).start()
        self.prompts = []

    def invoke(self, *answers, terminal=True):
        replies = iter(answers)
        def ask(prompt):
            self.prompts.append(prompt)
            try:
                return next(replies)
            except StopIteration:
                self.fail(f'unexpected prompt: {prompt}')
        output = io.StringIO()
        with contextlib.redirect_stdout(output), contextlib.redirect_stderr(output):
            result = guided.main(ROOT, manager=self.manager, ask=ask, terminal=terminal)
        return result, output.getvalue()

    def setup_tmux(self):
        code, output = self.invoke('1', 'desktop', 'tmux', 'yes', 'yes')
        self.assertEqual(code, 0, output)

    def test_no_arguments_open_guided_entrypoint(self):
        with patch.object(sys, 'argv', ['dotawesome']), patch.object(guided, 'main', return_value=0) as run:
            self.assertEqual(cli.main(ROOT), 0)
        run.assert_called_once()

    def test_noninteractive_entrypoint_explains_explicit_commands_without_changes(self):
        code, output = self.invoke(terminal=False)
        self.assertEqual(code, 1)
        self.assertIn('terminal', output)
        self.assertEqual(list(self.home.rglob('*')), [])

    def test_status_and_exit_leave_fresh_home_untouched(self):
        for answers in [('0',), ('2', 'tmux')]:
            code, output = self.invoke(*answers)
            self.assertEqual(code, 0, output)
            self.assertIn('tmux', output)
            self.assertEqual(list(self.home.rglob('*')), [])
            self.assertFalse(self.manager.build_root.exists())

    def test_declining_setup_preserves_home_and_does_not_install(self):
        code, output = self.invoke('1', 'desktop', 'tmux', 'no')
        self.assertEqual(code, 0, output)
        self.assertEqual(list(self.home.rglob('*')), [])
        self.assertEqual(self.installations, [])

    def test_setup_deploys_and_remembers_selection_then_skips_current_files(self):
        self.setup_tmux()
        target = self.home / '.tmux.conf'
        before = target.stat().st_mtime_ns
        code, output = self.invoke('1', '', 'yes')
        self.assertEqual(code, 0, output)
        self.assertIn('tmux: current', output)
        self.assertEqual(before, target.stat().st_mtime_ns)
        preferences = self.manager.state / 'preferences.json'
        self.assertEqual(preferences.stat().st_mode & 0o777, 0o600)
        self.assertEqual(json.loads(preferences.read_text()), {'machine': 'desktop', 'components': ['tmux']})
        self.assertEqual(self.installations, [])

    def test_missing_packages_offer_installation_without_a_flag(self):
        self.installed.clear()
        code, output = self.invoke('1', 'desktop', 'tmux', 'yes', 'yes', 'yes')
        self.assertEqual(code, 0, output)
        self.assertIn('full system upgrade', output)
        self.assertEqual(self.installations, [{'tmux': ['tmux']}])
        self.assertTrue((self.home / '.tmux.conf').exists())

    def test_declined_packages_defer_affected_component(self):
        self.installed.clear()
        code, output = self.invoke('1', 'desktop', 'tmux', 'yes', 'no')
        self.assertEqual(code, 0, output)
        self.assertIn('deferred', output)
        self.assertFalse((self.home / '.tmux.conf').exists())
        self.assertEqual(self.installations, [])

    def test_local_edits_can_be_kept_without_overwriting_them(self):
        self.setup_tmux()
        target = self.home / '.tmux.conf'
        target.write_text('# my edit\n')
        code, output = self.invoke('1', '', 'yes', 'k')
        self.assertEqual(code, 0, output)
        self.assertEqual(target.read_text(), '# my edit\n')
        self.assertIn('local edits kept', output)

    def test_local_edits_are_backed_up_before_explicit_restore_and_update(self):
        self.setup_tmux()
        target = self.home / '.tmux.conf'
        target.write_text('# my edit\n')
        code, output = self.invoke('1', '', 'yes', 'r', 'yes')
        self.assertEqual(code, 0, output)
        self.assertNotEqual(target.read_text(), '# my edit\n')
        backups = list((self.manager.state / 'local-edits').glob('*/files.json'))
        self.assertEqual(len(backups), 1)
        self.assertEqual(backups[0].stat().st_mode & 0o777, 0o600)
        self.assertIn('IyBteSBlZGl0Cg==', backups[0].read_text())

    def test_running_niri_is_deferred_while_other_components_are_applied(self):
        with patch.object(guided.niri, 'outside_session', side_effect=Error('niri.service is active')):
            code, output = self.invoke('1', 'desktop', 'tmux niri', 'yes', 'yes')
        self.assertEqual(code, 0, output)
        self.assertTrue((self.home / '.tmux.conf').exists())
        self.assertNotIn('niri', self.manager.records())
        self.assertIn('Niri', output)
        self.assertIn('deferred', output)

    def test_current_niri_is_not_reported_as_pending_in_running_session(self):
        from dotawesome import niri
        files = niri.render(self.manager, niri.profile(ROOT, 'desktop'))
        self.manager.deploy('niri', files, 'snapshot', 'desktop')
        self.manager.save_json(self.manager.state / 'preferences.json', {'machine': 'desktop', 'components': ['niri']})
        with patch.object(guided.niri, 'outside_session', side_effect=Error('niri.service is active')):
            code, output = self.invoke('1', '', 'yes')
        self.assertEqual(code, 0, output)
        self.assertIn('niri: current', output)
        self.assertNotIn('Niri deferred', output)
        self.assertEqual(self.installations, [])

    def test_restore_menu_restores_original_and_retains_application_state(self):
        original = self.home / '.tmux.conf'
        original.write_text('# before dotawesome\n')
        self.setup_tmux()
        state = self.home / '.tmux-app-state'
        state.write_text('session data')
        code, output = self.invoke('3', 'tmux', 'yes')
        self.assertEqual(code, 0, output)
        self.assertEqual(original.read_text(), '# before dotawesome\n')
        self.assertEqual(state.read_text(), 'session data')
        self.assertNotIn('tmux', self.manager.records())

    def test_failed_native_validation_preserves_destination_and_returns_failure(self):
        target = self.home / '.tmux.conf'
        target.write_text('# original\n')
        with patch.object(Component, 'validate', side_effect=Error('invalid native config')):
            code, output = self.invoke('1', 'desktop', 'tmux', 'yes', 'yes')
        self.assertEqual(code, 1, output)
        self.assertEqual(target.read_text(), '# original\n')
        self.assertNotIn('tmux', self.manager.records())
        self.assertIn('invalid native config', output)

    def test_package_failure_stops_before_any_config_is_deployed(self):
        self.installed.clear()
        with patch.object(guided.Packages, 'install_many', side_effect=Error('package update failed')):
            code, output = self.invoke('1', 'desktop', 'tmux', 'yes', 'yes')
        self.assertEqual(code, 1, output)
        self.assertFalse((self.home / '.tmux.conf').exists())

    def test_interrupt_leaves_completed_work_resumable(self):
        def interrupt(prompt):
            raise KeyboardInterrupt()
        with contextlib.redirect_stdout(io.StringIO()):
            code = guided.main(ROOT, manager=self.manager, ask=interrupt, terminal=True)
        self.assertEqual(code, 130)
        self.assertEqual(list(self.home.rglob('*')), [])

    def test_saved_niri_machine_profile_is_used_when_shared_config_is_absent(self):
        from dotawesome import niri
        self.manager.deploy('niri', niri.render(self.manager, niri.profile(ROOT, 'g14')), 'snapshot', 'g14')
        code, output = self.invoke('1', 'tmux', 'yes', 'yes')
        self.assertEqual(code, 0, output)
        self.assertIn('Machine: g14', output)
        self.assertFalse(any(prompt.startswith('Machine [') for prompt in self.prompts))

    def test_session_recovery_is_refused_before_mutation_from_a_running_niri(self):
        target = self.home / 'interrupted-file'
        target.write_text('interrupted content')
        from dotawesome.manager import encode
        self.manager.save_json(self.manager.state / 'transaction.json', {
            'records': {}, 'reload_units': True,
            'changes': {str(target): {'before': encode(b'original', 0o644),
                                      'after': self.manager.inspect(target)}},
        })
        with patch.object(guided.niri, 'outside_session', side_effect=Error('niri.service is active')):
            code, output = self.invoke('1', 'desktop', 'tmux', 'yes', 'yes')
        self.assertEqual(code, 1, output)
        self.assertEqual(target.read_text(), 'interrupted content')
        self.assertTrue((self.manager.state / 'transaction.json').exists())
        self.assertFalse((self.manager.state / 'preferences.json').exists())
        self.assertEqual(self.installations, [])

    def test_restoration_does_not_require_component_source_to_still_exist(self):
        original = self.home / '.tmux.conf'
        original.write_text('# original\n')
        self.setup_tmux()
        with patch.object(Component, 'render', side_effect=Error('source unavailable')):
            code, output = self.invoke('3', 'tmux', 'yes')
        self.assertEqual(code, 0, output)
        self.assertEqual(original.read_text(), '# original\n')


class AuthenticationTests(unittest.TestCase):
    def test_ready_logins_do_not_prompt_or_print_captured_credentials(self):
        result = type('Result', (), {'returncode': 0, 'stdout': 'private session data', 'stderr': ''})()
        out = io.StringIO()
        with patch.object(guided.shutil, 'which', return_value='/tool'), patch.object(guided.subprocess, 'run', return_value=result), contextlib.redirect_stdout(out):
            remaining = guided.authenticate(lambda prompt: self.fail(prompt))
        self.assertEqual(remaining, [])
        self.assertNotIn('private session data', out.getvalue())

    def test_missing_auth_tools_offer_guidance_without_launching_anything(self):
        with patch.object(guided.shutil, 'which', return_value=None), patch.object(guided.subprocess, 'run') as run:
            remaining = guided.authenticate(lambda prompt: self.fail(prompt))
        self.assertEqual(len(remaining), 2)
        run.assert_not_called()

    def test_declining_login_only_runs_read_only_checks(self):
        result = type('Result', (), {'returncode': 1})()
        with patch.object(guided.shutil, 'which', return_value='/tool'), patch.object(guided.subprocess, 'run', return_value=result) as run, contextlib.redirect_stdout(io.StringIO()):
            remaining = guided.authenticate(lambda prompt: 'no')
        self.assertEqual(len(remaining), 2)
        self.assertEqual([call.args[0] for call in run.call_args_list],
                         [['gh', 'auth', 'status'], ['aws', 'sts', 'get-caller-identity', '--profile', 'cof-tf']])

    def test_opted_in_login_is_rechecked(self):
        ready = type('Result', (), {'returncode': 0})()
        missing = type('Result', (), {'returncode': 1})()
        with patch.object(guided.shutil, 'which', return_value='/tool'), patch.object(guided.subprocess, 'run', side_effect=[missing, ready, ready, ready]) as run, contextlib.redirect_stdout(io.StringIO()):
            remaining = guided.authenticate(lambda prompt: 'yes')
        self.assertEqual(remaining, [])
        self.assertEqual(run.call_args_list[1].args[0], ['gh', 'auth', 'login'])
        self.assertEqual(run.call_args_list[2].args[0], ['gh', 'auth', 'status'])
