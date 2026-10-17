"""Validate generated configs with native tools whenever they are installed."""
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome.manager import Manager
from dotawesome import niri


class NativeConfigTests(unittest.TestCase):
    def validate_profiles(self, tool):
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp)
            manager = Manager(ROOT, work / 'home')
            for name in ('generic', 'g14'):
                with self.subTest(profile=name):
                    files = niri.render(manager, niri.profile(ROOT, name))
                    if tool == 'niri':
                        path = work / f'{name}.kdl'
                        path.write_bytes(files[manager.config / 'niri/config.kdl'][0])
                        command = ['niri', 'validate', '--config', str(path)]
                    else:
                        shell_root = manager.config / 'dotawesome/niri/noctalia/noctalia'
                        for source, (data, _) in files.items():
                            if source.is_relative_to(shell_root):
                                target = work / name / source.relative_to(shell_root)
                                target.parent.mkdir(parents=True, exist_ok=True)
                                target.write_bytes(data)
                        path = work / name / 'config.toml'
                        command = ['noctalia', 'config', 'validate', str(path)]
                    result = subprocess.run(command, capture_output=True, text=True, timeout=30)
                    diagnostics = result.stdout + result.stderr
                    self.assertEqual(result.returncode, 0, diagnostics)
                    self.assertIsNone(re.search(r'\b(?:warn(?:ing)?|unknown|obsolete|error)\b', diagnostics, re.I), diagnostics)

    @unittest.skipUnless(shutil.which('niri'), 'Niri validator not installed')
    def test_niri_profiles_pass_native_validation(self):
        self.validate_profiles('niri')

    @unittest.skipUnless(shutil.which('noctalia'), 'Noctalia validator not installed')
    def test_noctalia_profiles_pass_native_validation(self):
        self.validate_profiles('noctalia')
