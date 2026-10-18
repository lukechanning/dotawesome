from pathlib import Path
import os
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class SecretChecksTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.repo = Path(self.tmp.name)
        subprocess.run(['git', 'init', '-q', str(self.repo)], check=True)
        shutil.copy2(ROOT / '.gitleaks.toml', self.repo)

    def scan(self, *args):
        return subprocess.run([str(ROOT / 'bin/check-secrets'), *args], cwd=self.repo, capture_output=True, text=True)

    def test_staged_plaintext_identity_and_credentials_rejected_redacted(self):
        for name, contents in [('key.txt', 'AGE-SECRET-KEY-' + 'A' * 59),
                               ('credentials', 'aws_secret_access_key = ' + 'disposable-but-plaintext')]:
            with self.subTest(name=name):
                (self.repo / name).write_text(contents + '\n')
                subprocess.run(['git', '-C', str(self.repo), 'add', name], check=True)
                result = self.scan('--staged')
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn(contents, result.stdout + result.stderr)
                # Even if working content is clean, the index must be scanned.
                (self.repo / name).write_text('clean working tree\n')
                self.assertNotEqual(self.scan('--staged').returncode, 0)
                subprocess.run(['git', '-C', str(self.repo), 'update-index', '--force-remove', name], check=True, capture_output=True)

    def test_encrypted_source_accepted_and_tracked_working_tree_scanned(self):
        source = self.repo / 'encrypted_private_credentials.age'
        source.write_text('age-encryption.org/v1\n-> scrypt disposable encrypted data\n')
        subprocess.run(['git', '-C', str(self.repo), 'add', '.'], check=True)
        self.assertEqual(self.scan('--staged').returncode, 0)
        self.assertEqual(self.scan().returncode, 0)
        source.write_text('aws_secret_access_key = disposable-but-plaintext\n')
        self.assertNotEqual(self.scan().returncode, 0)

    def test_missing_scanner_fails_with_install_guidance(self):
        result = subprocess.run([sys.executable, str(ROOT / 'bin/check-secrets')], cwd=self.repo,
                                env=dict(os.environ, PATH=str(self.repo)), capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('gitleaks is required', result.stderr)
