"""Real chezmoi/age integration in disposable checkouts and private XDG homes."""
import os
from pathlib import Path
import pty
import re
import select
import shutil
import subprocess
import sys
import tempfile
import time
import tomllib
import unittest

ROOT = Path(__file__).resolve().parents[1]


def terminal(command, env, answers):
    pid, fd = pty.fork()
    if pid == 0:
        try:
            os.execvpe(command[0], command, env)
        except OSError:
            os._exit(127)
    output = b''
    pending = b''
    deadline = time.monotonic() + 30
    try:
        while time.monotonic() < deadline:
            ready, _, _ = select.select([fd], [], [], .1)
            if ready:
                try:
                    chunk = os.read(fd, 65536)
                except OSError:
                    break
                if not chunk:
                    break
                output += chunk
                pending += chunk
                prompt = rb'(?:Enter passphrase[^\r\n]*|Confirm passphrase|Apply shared configuration\? \[yes/no\]):'
                if answers and re.search(prompt, pending):
                    os.write(fd, (answers.pop(0) + '\n').encode())
                    pending = b''
            done, status = os.waitpid(pid, os.WNOHANG)
            if done:
                return os.waitstatus_to_exitcode(status), output
        else:
            os.kill(pid, 9)
            os.waitpid(pid, 0)
            raise AssertionError('terminal command timed out')
        _, status = os.waitpid(pid, 0)
        return os.waitstatus_to_exitcode(status), output
    finally:
        os.close(fd)


@unittest.skipUnless(shutil.which('chezmoi'), 'chezmoi required for integration (bin/ci requires it)')
class BootstrapTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.work = Path(self.tmp.name)
        self.repo = self.work / 'repo'
        self.repo.mkdir()
        for name in ('home', 'bin', 'lib', 'components', 'hooks'):
            shutil.copytree(ROOT / name, self.repo / name)
        shutil.copy2(ROOT / '.chezmoiroot', self.repo)
        subprocess.run(['git', 'init', '-q', str(self.repo)], check=True)
        self.home = self.work / 'user'
        self.home.mkdir()
        self.env = dict(os.environ, HOME=str(self.home), TERM='dumb', NO_COLOR='1',
                        XDG_CONFIG_HOME=str(self.home / '.config'), XDG_DATA_HOME=str(self.home / '.local/share'),
                        XDG_STATE_HOME=str(self.home / '.local/state'), XDG_CACHE_HOME=str(self.home / '.cache'))
        self.key = self.work / 'disposable-key'
        subprocess.run(['age-keygen', '-o', str(self.key)], check=True, capture_output=True)
        recipient = subprocess.check_output(['age-keygen', '-y', str(self.key)], text=True).strip()
        (self.repo / 'encryption').mkdir()
        (self.repo / 'encryption/recipient.txt').write_text(recipient + '\n')
        code, _ = terminal(['age', '-p', '-o', str(self.repo / 'encryption/identity.age'), str(self.key)], self.env,
                           ['disposable test passphrase', 'disposable test passphrase'])
        self.assertEqual(code, 0)

    def bootstrap(self, *args, answer='no\n'):
        return subprocess.run([str(self.repo / 'bin/init'), *args], env=self.env,
                              input=answer, capture_output=True, text=True)

    def recover(self):
        # Bootstrap recovers before diff; decline application at the final prompt.
        code, output = terminal([str(self.repo / 'bin/init'), '--machine', 'g14'], self.env,
                                ['disposable test passphrase', 'no'])
        self.assertEqual(code, 0, output.decode())
        return self.home / '.config/chezmoi/key.txt'

    def test_fresh_recovery_preview_apply_repeat_and_drift_for_both_machines(self):
        key = self.recover()
        self.assertEqual(key.stat().st_mode & 0o777, 0o600)
        self.assertFalse((self.home / '.gitconfig').exists())
        hook = self.repo / '.git/hooks/pre-commit'
        self.assertEqual(hook.read_bytes(), (self.repo / 'hooks/pre-commit').read_bytes())
        self.assertTrue(os.access(hook, os.X_OK))
        for machine in ('g14', 'desktop'):
            result = self.bootstrap('--machine', machine, answer='yes\n')
            if machine == 'desktop':
                # Changing an initialized machine is deliberately refused.
                self.assertNotEqual(result.returncode, 0)
                config = self.home / '.config/chezmoi/chezmoi.toml'
                config.write_text(config.read_text().replace('g14', 'desktop'))
                result = self.bootstrap(answer='yes\n')
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('Preview', result.stdout)
            self.assertTrue((self.home / '.gitconfig').is_file())
            self.assertFalse((self.home / '.config/fish/config.fish').exists())
            self.assertFalse((self.home / '.aws/credentials').exists())
            for operation in ('status', 'diff'):
                check = subprocess.run(['chezmoi', operation], env=self.env, capture_output=True, text=True)
                self.assertEqual(check.returncode, 0, check.stderr)
                self.assertEqual(check.stdout, '')
            again = self.bootstrap()
            self.assertEqual(again.returncode, 0, again.stderr)
            data = tomllib.loads((self.home / '.config/chezmoi/chezmoi.toml').read_text())
            self.assertEqual(data['data']['machine'], machine)
            (self.home / '.gitconfig').write_text('# local drift\n')
            drift = subprocess.check_output(['chezmoi', 'diff'], env=self.env, text=True)
            self.assertIn('local drift', drift)

    def test_wrong_passphrase_does_not_publish_key_or_apply_files(self):
        code, _ = terminal([str(self.repo / 'bin/init'), '--machine', 'desktop'], self.env, ['wrong'])
        self.assertNotEqual(code, 0)
        self.assertFalse((self.home / '.config/chezmoi/key.txt').exists())
        self.assertFalse((self.home / '.gitconfig').exists())
        self.assertEqual(list((self.home / '.config/chezmoi').glob('.identity-*')), [])

    def test_encrypted_disposable_credentials_deploy_privately(self):
        self.recover()
        credentials = b'[disposable-test]\naws_access_key_id = disposable-test-only\naws_secret_access_key = disposable-test-only\n'
        target = self.repo / 'home/private_dot_aws/encrypted_private_credentials.age'
        subprocess.run(['age', '-r', (self.repo / 'encryption/recipient.txt').read_text().strip(), '-o', str(target)],
                       input=credentials, check=True)
        result = self.bootstrap(answer='yes\n')
        self.assertEqual(result.returncode, 0, result.stderr)
        destination = self.home / '.aws/credentials'
        self.assertEqual(destination.read_bytes(), credentials)
        self.assertEqual(destination.stat().st_mode & 0o777, 0o600)
        self.assertEqual(subprocess.check_output(['chezmoi', 'status'], env=self.env), b'')

    def test_overlapping_source_refused_before_apply(self):
        self.recover()
        source = self.repo / 'home/dot_config/niri'
        source.mkdir()
        (source / 'config.kdl').write_text('// overlap')
        result = self.bootstrap(answer='yes\n')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('overlap', result.stderr)
        self.assertFalse((self.home / '.gitconfig').exists())

    def test_enrollment_verifies_recovery_before_publishing_and_refuses_overwrite(self):
        (self.repo / 'encryption/recipient.txt').unlink()
        (self.repo / 'encryption/identity.age').unlink()
        code, output = terminal([str(self.repo / 'bin/enroll-age')], self.env,
                                ['enrollment test passphrase', 'enrollment test passphrase', 'enrollment test passphrase'])
        self.assertEqual(code, 0, output.decode())
        identity = self.home / '.config/chezmoi/key.txt'
        self.assertEqual(identity.stat().st_mode & 0o777, 0o600)
        before = identity.read_bytes()
        result = subprocess.run([str(self.repo / 'bin/enroll-age')], env=self.env, capture_output=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(identity.read_bytes(), before)
        public = subprocess.check_output(['age-keygen', '-y', str(identity)], text=True)
        self.assertEqual(public, (self.repo / 'encryption/recipient.txt').read_text())

    def test_desktop_fresh_initialization_backup_and_retained_init(self):
        key = self.home / '.config/chezmoi/key.txt'
        key.parent.mkdir(parents=True)
        shutil.copyfile(self.key, key)
        key.chmod(0o600)
        original = self.home / '.gitconfig'
        original.write_text('# adopted original\n')
        result = self.bootstrap('--machine', 'desktop', answer='yes\n')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.home / '.config').stat().st_mode & 0o777, 0o755)
        self.assertEqual((self.home / '.config/alacritty/alacritty.toml').stat().st_mode & 0o777, 0o644)
        snapshots = list((self.home / '.local/state/dotawesome/adoption').glob('shared-*'))
        self.assertEqual(len(snapshots), 1)
        self.assertEqual((snapshots[0] / '.gitconfig').read_text(), '# adopted original\n')
        self.assertEqual((snapshots[0] / '.gitconfig').stat().st_mode & 0o777, 0o600)
        result = subprocess.run(['chezmoi', 'init', '--no-tty'], env=self.env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        data = tomllib.loads((self.home / '.config/chezmoi/chezmoi.toml').read_text())
        self.assertEqual(data['data']['machine'], 'desktop')
        self.assertEqual(Path(data['sourceDir']), self.repo)
        self.assertEqual(data['age']['recipient'], (self.repo / 'encryption/recipient.txt').read_text().strip())

    def test_mismatched_age_configuration_is_refused(self):
        self.recover()
        config = self.home / '.config/chezmoi/chezmoi.toml'
        recipient = (self.repo / 'encryption/recipient.txt').read_text().strip()
        config.write_text(config.read_text().replace(recipient, 'age1wrong'))
        result = self.bootstrap(answer='yes\n')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('age configuration', result.stderr)
        self.assertFalse((self.home / '.gitconfig').exists())

    def test_init_preserves_existing_hooks_and_stops_before_shared_setup(self):
        hook = self.repo / '.git/hooks/pre-commit'
        original = b'#!/bin/sh\necho custom hook\n'
        hook.write_bytes(original)
        result = self.bootstrap('--machine', 'desktop', answer='yes\n')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('existing pre-commit hook preserved', result.stderr)
        self.assertEqual(hook.read_bytes(), original)
        self.assertFalse((self.home / '.config/chezmoi').exists())

    def test_init_preserves_configured_hooks_directory(self):
        subprocess.run(['git', '-C', str(self.repo), 'config', 'core.hooksPath', 'custom-hooks'], check=True)
        result = self.bootstrap('--machine', 'desktop', answer='yes\n')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('core.hooksPath', result.stderr)
        self.assertFalse((self.home / '.config/chezmoi').exists())

    def test_explicit_existing_hook_setup_can_be_kept(self):
        self.recover()
        hook = self.repo / '.git/hooks/pre-commit'
        original = b'#!/bin/sh\nexec hooks/pre-commit\n'
        hook.write_bytes(original)
        result = self.bootstrap('--skip-hooks', answer='yes\n')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(hook.read_bytes(), original)

    def guided(self, answers, passphrases=()):
        # Mock package inventory and authentication only; use real age and chezmoi.
        script = """
import sys
from pathlib import Path
from unittest.mock import patch
sys.path.insert(0, str(Path(sys.argv[1]) / 'lib'))
from dotawesome import guided
answers = iter(__import__('json').loads(sys.argv[2]))
def ask(prompt):
    print(prompt, flush=True)
    return next(answers)
with patch.object(guided.Packages, 'installed', return_value=set(guided.SHARED_PACKAGES)), patch.object(guided, 'authenticate', return_value=[]):
    sys.exit(guided.main(Path(sys.argv[1]), ask=ask))
"""
        import json
        command = [sys.executable, '-B', '-c', script, str(self.repo), json.dumps(answers)]
        return terminal(command, self.env, list(passphrases))

    def test_guided_shared_recovery_deployment_and_repeat_skip(self):
        code, output = self.guided(['1', 'desktop', 'shared', 'yes', 'yes'], ['disposable test passphrase'])
        self.assertEqual(code, 0, output.decode())
        self.assertTrue((self.home / '.gitconfig').exists())
        self.assertEqual(subprocess.check_output(['chezmoi', 'status'], env=self.env), b'')
        snapshots = list((self.home / '.local/state/dotawesome/adoption').iterdir())
        code, output = self.guided(['1', '', 'yes'])
        self.assertEqual(code, 0, output.decode())
        self.assertIn('Shared configs: current', output.decode())
        self.assertEqual(list((self.home / '.local/state/dotawesome/adoption').iterdir()), snapshots)

    def test_guided_first_enrollment_and_shared_apply_use_one_entrypoint(self):
        (self.repo / 'encryption/recipient.txt').unlink()
        (self.repo / 'encryption/identity.age').unlink()
        code, output = self.guided(['1', 'g14', 'shared', 'yes', 'yes', 'yes'],
                                  ['guided disposable passphrase'] * 3)
        self.assertEqual(code, 0, output.decode())
        self.assertTrue((self.home / '.gitconfig').exists())
        key = self.home / '.config/chezmoi/key.txt'
        self.assertEqual(key.stat().st_mode & 0o777, 0o600)
        self.assertTrue((self.repo / 'encryption/identity.age').exists())

    def test_guided_partial_enrollment_is_preserved_and_blocks_shared_apply(self):
        (self.repo / 'encryption/identity.age').unlink()
        original = (self.repo / 'encryption/recipient.txt').read_bytes()
        code, output = self.guided(['1', 'desktop', 'shared', 'yes'])
        self.assertEqual(code, 1, output.decode())
        self.assertIn('partial age enrollment', output.decode())
        self.assertEqual((self.repo / 'encryption/recipient.txt').read_bytes(), original)
        self.assertFalse((self.home / '.gitconfig').exists())
        self.assertFalse((self.home / '.config/chezmoi/key.txt').exists())

    def test_guided_shared_drift_can_be_kept_or_adopted_with_a_backup(self):
        code, output = self.guided(['1', 'desktop', 'shared', 'yes', 'yes'], ['disposable test passphrase'])
        self.assertEqual(code, 0, output.decode())
        target = self.home / '.gitconfig'
        target.write_text('# intentional local edit\n')
        code, output = self.guided(['1', '', 'yes', 'no'])
        self.assertEqual(code, 0, output.decode())
        self.assertEqual(target.read_text(), '# intentional local edit\n')
        self.assertIn('differences to review', output.decode())
        code, output = self.guided(['1', '', 'yes', 'yes'])
        self.assertEqual(code, 0, output.decode())
        snapshots = list((self.home / '.local/state/dotawesome/adoption').glob('*/.gitconfig'))
        self.assertTrue(any(p.read_text() == '# intentional local edit\n' for p in snapshots))
        self.assertTrue(all(p.stat().st_mode & 0o777 == 0o600 for p in snapshots))
