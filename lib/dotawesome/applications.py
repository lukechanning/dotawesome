"""File-based application components; validation uses disposable environments."""
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

from .manager import Error, run

PACKAGES = {'nvim': ['neovim', 'git', 'ripgrep'], 'tmux': ['tmux']}


def render(name, manager):
    source = manager.repo / 'components' / name
    if name == 'tmux':
        return {manager.home / '.tmux.conf': ((source / 'tmux.conf').read_bytes(), 0o644)}
    return {manager.config / 'nvim' / path.relative_to(source / 'config'): (path.read_bytes(), 0o644)
            for path in sorted((source / 'config').rglob('*')) if path.is_file()}


def check(name, manager):
    commands = ['nvim', 'git', 'rg'] if name == 'nvim' else ['tmux']
    report = {'errors': [], 'warnings': [], 'packages': PACKAGES[name]}
    for command in commands:
        if not shutil.which(command):
            report['errors'].append(f'missing command: {command}')
    if name == 'nvim' and (manager.config / 'nvim/init.vim').exists():
        report['errors'].append('existing nvim/init.vim conflicts with init.lua; move it aside before applying (it is not deleted automatically)')
    if name == 'nvim' and shutil.which('nvim'):
        # Even --version initializes Neovim's logger; keep plans read-only.
        version = run(['nvim', '--version'], env=dict(os.environ, NVIM_LOG_FILE=os.devnull))
        match = re.search(r'NVIM v(\d+)\.(\d+)', version)
        if not match or tuple(map(int, match.groups())) < (0, 12):
            report['errors'].append('Neovim >= 0.12 required')
        report['warnings'].append('Plugins download on first editor launch; language tools and parsers are installed explicitly in Neovim.')
    return report


def validate(name, manager, files):
    with tempfile.TemporaryDirectory(prefix=f'dotawesome-{name}-') as tmp:
        work = Path(tmp)
        if name == 'tmux':
            config = work / 'tmux.conf'
            config.write_bytes(files[manager.home / '.tmux.conf'][0])
            socket = str(work / 'socket')
            # Empty server: source options without a session or inherited TMUX.
            command = ['tmux', '-S', socket, '-f', '/dev/null', 'start-server', ';',
                       'set-option', '-g', 'exit-empty', 'off', ';', 'source-file', str(config)]
            env = dict(os.environ, TMUX='', HOME=str(work))
            try:
                run(command, env=env)
            finally:
                subprocess.run(['tmux', '-S', socket, 'kill-server'], env=env, capture_output=True)
        else:
            root = manager.config / 'nvim'
            for path, (data, _) in files.items():
                target = work / 'config/nvim' / path.relative_to(root)
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(data)
            env = dict(os.environ, HOME=str(work), XDG_CONFIG_HOME=str(work / 'config'),
                       XDG_DATA_HOME=str(work / 'data'), XDG_STATE_HOME=str(work / 'state'),
                       XDG_CACHE_HOME=str(work / 'cache'), NVIM_LOG_FILE=str(work / 'nvim.log'), NVIM_APPNAME='nvim', DOTAWESOME_NVIM_VALIDATE='1')
            output = run(['nvim', '--headless', '-i', 'NONE', '+lua for _, p in ipairs(vim.fn.globpath(vim.fn.stdpath("config"), "**/*.lua", false, true)) do assert(loadfile(p)) end', '+lua if vim.v.errmsg ~= "" then error(vim.v.errmsg) end', '+qa!'], env=env)
            if re.search(r'Error|E\d{3}:', output):
                raise Error(f'Neovim validation failed: {output}')
    return True
