from pathlib import Path
import os
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
FISH = ROOT / 'home/dot_config/fish'


class FishTests(unittest.TestCase):
    def run_fish(self, script, home, path='/usr/bin', interactive=False):
        sources = '; '.join(f'source {p}' for p in sorted(FISH.rglob('*.fish')))
        return subprocess.run(['/usr/bin/fish', '--no-config', *(['-i'] if interactive else []), '-c', sources + '; ' + script],
                              env=dict(os.environ, HOME=str(home), PATH=path), capture_output=True, text=True)

    def test_syntax_and_quiet_startup(self):
        paths = list(FISH.rglob('*.fish'))
        self.assertTrue(paths, 'personal Fish sources missing')
        for path in paths:
            result = subprocess.run(['fish', '-n', str(path)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stderr)
        with tempfile.TemporaryDirectory() as tmp:
            result = self.run_fish('true', tmp)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout + result.stderr, '')

    def test_helpers_fail_usefully_without_local_dependencies(self):
        with tempfile.TemporaryDirectory() as tmp:
            for command, message in [('slakker-dev', 'Slakker'), ('vpn on', 'nmcli'), ('@kyper', 'project')]:
                result = self.run_fish(command, tmp, path=tmp)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(message, result.stderr)

    def test_helpers_forward_arguments_when_dependencies_exist(self):
        with tempfile.TemporaryDirectory() as tmp:
            home = Path(tmp)
            launcher = home / 'Code/slakker/bin/slakker-dev'
            launcher.parent.mkdir(parents=True)
            launcher.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            launcher.chmod(0o755)
            result = self.run_fish("slakker-dev 'project with spaces'", home)
            self.assertEqual(result.stdout, 'project with spaces\n')
            self.assertEqual(result.returncode, 0, result.stderr)
            (home / 'Code/kyper').mkdir()
            result = self.run_fish('@kyper; pwd', home)
            self.assertEqual(result.stdout.strip(), str(home / 'Code/kyper'))
            nmcli = home / 'nmcli'
            nmcli.write_text('#!/bin/sh\nprintf "%s\\n" "$@"\n')
            nmcli.chmod(0o755)
            result = self.run_fish('vpn on', home, path=str(home))
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn('--ask\nconnection\nup\nid\nus-lax-surfshark', result.stdout)

    def test_interactive_mise_activation_is_conditional(self):
        with tempfile.TemporaryDirectory() as tmp:
            home = Path(tmp)
            result = self.run_fish('true', home, path=tmp, interactive=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, '')
            mise = home / 'mise'
            mise.write_text("#!/bin/sh\nprintf 'set -g DOTAWESOME_TEST_MISE yes\\n'\n")
            mise.chmod(0o755)
            result = self.run_fish('echo $DOTAWESOME_TEST_MISE', home, path=tmp, interactive=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stdout, 'yes\n')

    def test_missing_vpn_connection_has_local_setup_guidance(self):
        with tempfile.TemporaryDirectory() as tmp:
            nmcli = Path(tmp) / 'nmcli'
            nmcli.write_text('#!/bin/sh\nexit 1\n')
            nmcli.chmod(0o755)
            result = self.run_fish('vpn on', tmp, path=tmp)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Missing VPN connection', result.stderr)
