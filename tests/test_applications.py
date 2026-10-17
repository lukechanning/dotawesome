"""Application checks run against private homes, never the user's sessions."""
from pathlib import Path
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'lib'))
from dotawesome import applications
from dotawesome.manager import Error, Manager, encode


class ApplicationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(); self.addCleanup(self.tmp.cleanup)
        self.manager = Manager(ROOT, Path(self.tmp.name) / 'home', Path(self.tmp.name) / 'build', runner=lambda *a, **k: None)

    @unittest.skipUnless(shutil.which('nvim'), 'Neovim unavailable')
    def test_neovim_validates_without_plugins_or_network(self):
        applications.validate('nvim', self.manager, applications.render('nvim', self.manager))
        self.assertFalse(self.manager.home.exists())

    @unittest.skipUnless(shutil.which('tmux'), 'tmux unavailable')
    def test_tmux_validates_without_touching_a_running_server(self):
        applications.validate('tmux', self.manager, applications.render('tmux', self.manager))
        self.assertFalse(self.manager.home.exists())

    def test_application_rollback_and_recovery_do_not_reload_systemd(self):
        calls = []; self.manager.runner = lambda *a, **kw: calls.append(a)
        target = self.manager.home / '.tmux.conf'
        target.parent.mkdir(parents=True); target.write_text('original')
        other = self.manager.home / 'second'
        write = self.manager.write
        def fail(path, value):
            if path == other and value is not None:
                raise OSError('disk full')
            write(path, value)
        with patch.object(self.manager, 'write', side_effect=fail):
            with self.assertRaisesRegex(OSError, 'disk full'):
                self.manager.deploy('tmux', {target: (b'changed', 0o644), other: (b'x', 0o644)}, 'snapshot', reload_units=False)
        self.assertEqual(target.read_text(), 'original')
        after = encode(b'interrupted', 0o644); before = self.manager.inspect(target)
        self.manager.save_json(self.manager.state / 'transaction.json', {
            'records': {}, 'reload_units': False, 'changes': {str(target): {'before': before, 'after': after}}})
        self.manager.write(target, after)
        with self.manager.lock():
            pass
        self.assertEqual(target.read_text(), 'original')
        self.assertEqual(calls, [])

    @unittest.skipUnless(shutil.which('nvim'), 'Neovim unavailable')
    def test_missing_treesitter_parser_keeps_special_buffers_usable(self):
        script = Path(self.tmp.name) / 'missing-parser.lua'
        source = ROOT / 'components/nvim/config/lua/dotawesome/treesitter.lua'
        script.write_text("package.preload['nvim-treesitter'] = function() return { setup = function() end } end\n"
                          + f"dofile({str(source)!r})\n"
                          + "vim.cmd('enew')\nvim.cmd('setfiletype NvimTree')\n")
        env = dict(os.environ, HOME=self.tmp.name, XDG_CONFIG_HOME=self.tmp.name + '/config',
                   XDG_DATA_HOME=self.tmp.name + '/data', XDG_STATE_HOME=self.tmp.name + '/state',
                   XDG_CACHE_HOME=self.tmp.name + '/cache', NVIM_LOG_FILE=self.tmp.name + '/nvim.log')
        result = subprocess.run(['nvim', '--clean', '--headless', '-i', 'NONE', '-l', str(script)], env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    @unittest.skipUnless(all(shutil.which(c) for c in ('nvim', 'git', 'rg')), 'Neovim dependencies unavailable')
    def test_plan_does_not_create_editor_logs_or_state(self):
        work = Path(self.tmp.name) / 'cwd'; work.mkdir()
        env = dict(os.environ, HOME=str(self.manager.home), XDG_CONFIG_HOME=str(self.manager.home / 'config'),
                   XDG_DATA_HOME=str(self.manager.home / 'data'), XDG_STATE_HOME=str(self.manager.home / 'state'),
                   XDG_CACHE_HOME=str(self.manager.home / 'cache'))
        env.pop('NVIM_LOG_FILE', None)
        result = subprocess.run([str(ROOT / 'bin/dotawesome'), 'plan', 'nvim'], cwd=work, env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertFalse(self.manager.home.exists(), 'Plan created editor state')
        self.assertEqual(list(work.iterdir()), [], 'Plan created an editor log in the working directory')
