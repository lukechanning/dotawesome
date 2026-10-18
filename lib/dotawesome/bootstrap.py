"""Initialize hooks and shared configuration from the current checkout. No package installation or component writes."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import tomllib

from . import hooks
from .shared import check_ownership, recover_identity


def command(argv, **kwargs):
    # Chezmoi applies explicit private attributes; normal directories need 0755.
    return subprocess.run(argv, check=True, umask=0o022, **kwargs)


def backup(home, targets, state):
    """Private adoption snapshot, including symlinks, before chezmoi writes."""
    state.mkdir(parents=True, exist_ok=True, mode=0o700)
    directory = Path(tempfile.mkdtemp(prefix='shared-', dir=state))
    manifest = []
    for name in targets:
        source = home / name
        if not os.path.lexists(source):
            continue
        destination = directory / name
        destination.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        if source.is_symlink():
            destination.symlink_to(os.readlink(source))
        else:
            with source.open('rb') as original, destination.open('xb') as output:
                os.chmod(destination, 0o600)
                shutil.copyfileobj(original, output)
        manifest.append(name)
    (directory / 'manifest.json').write_text(json.dumps(manifest, indent=2) + '\n')
    return directory


def initialize(repo, machine=None, skip_hooks=False, ask=None, skip_current=False):
    """Shared setup service; no component writes and no package installation."""
    previous = os.umask(0o077)
    try:
        return _initialize(repo, machine, skip_hooks, ask or input, skip_current)
    finally:
        os.umask(previous)


def _initialize(repo, machine, skip_hooks, ask, skip_current):
    for tool in ('git', 'chezmoi', 'age', 'age-keygen', 'gitleaks'):
        if not shutil.which(tool):
            raise ValueError(f'missing {tool}; install explicitly: sudo pacman -S git chezmoi age gitleaks')
    if not skip_hooks:
        hooks.install(repo)
    home = Path.home()
    config_root = Path(os.environ.get('XDG_CONFIG_HOME', home / '.config'))
    if config_root != home / '.config':
        # Sources intentionally use standard CachyOS destination paths.
        raise ValueError('shared configuration requires XDG_CONFIG_HOME=$HOME/.config')
    recipient_file = repo / 'encryption/recipient.txt'
    recovery = repo / 'encryption/identity.age'
    if not recipient_file.is_file() or not recovery.is_file():
        raise ValueError('age enrollment missing; run bin/enroll-age locally and commit the public recipient and encrypted recovery copy')
    recipient = recipient_file.read_text().strip()
    config = config_root / 'chezmoi/chezmoi.toml'
    cm = ['chezmoi', '--source', str(repo), '--destination', str(home), '--config', str(config), '--no-pager']
    requested = machine
    if config.exists():
        data = tomllib.loads(config.read_text())
        machine = data.get('data', {}).get('machine')
        if machine not in ('g14', 'desktop'):
            raise ValueError('existing chezmoi config has no valid machine; inspect it before initialization')
        if requested and requested != machine:
            raise ValueError(f'machine already initialized as {machine}; edit local chezmoi config to change it')
        if Path(data.get('sourceDir', '')).expanduser().resolve() != repo:
            raise ValueError('chezmoi sourceDir belongs to another checkout; inspect config before changing it')
    else:
        machine = requested or ask('Machine [g14/desktop]: ').strip()
        if machine not in ('g14', 'desktop'):
            raise ValueError('machine must be g14 or desktop')
        config.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        command(cm + ['init', '--promptChoice', f'Machine={machine}'])
    data = tomllib.loads(config.read_text())
    age = data.get('age', {})
    if (data.get('encryption') != 'age' or age.get('recipient') != recipient or
            Path(age.get('identity', '')).expanduser() != config.parent / 'key.txt'):
        raise ValueError('local age configuration does not match enrollment; inspect it and run chezmoi init to refresh')
    recover_identity(config.parent / 'key.txt', recovery, recipient)
    targets = subprocess.check_output(cm + ['managed', '--include', 'files,symlinks', '--path-style', 'relative'], text=True, umask=0o022).splitlines()
    directories = subprocess.check_output(cm + ['managed', '--include', 'dirs', '--path-style', 'relative'], text=True, umask=0o022).splitlines()
    check_ownership(targets, directories)
    if skip_current and not subprocess.check_output(cm + ['status'], text=True, umask=0o022).strip():
        print('Shared configuration: current; no changes needed')
        return 'current'
    print('Preview shared configuration (may display decrypted encrypted targets):', flush=True)
    command(cm + ['diff'])
    if ask('Apply shared configuration? [yes/no]: ').strip().lower() != 'yes':
        print('Shared configuration left unchanged.')
        return 'declined'
    state = Path(os.environ.get('XDG_STATE_HOME', home / '.local/state')) / 'dotawesome/adoption'
    snapshot = backup(home, targets, state)
    command(cm + ['apply', '--force'])
    print(f'Shared configuration applied. Adoption backup: {snapshot}')
    if not skip_current:
        print(f'Components remain explicit; run outside Niri:\n  bin/dotawesome plan niri --profile {machine}\n  bin/dotawesome apply niri --profile {machine}\n  bin/dotawesome plan nvim\n  bin/dotawesome plan tmux')
    return 'applied'


def main(repo):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--machine', choices=('g14', 'desktop'))
    parser.add_argument('--skip-hooks', action='store_true',
                        help='keep an existing hook setup after manually chaining hooks/pre-commit')
    args = parser.parse_args()
    try:
        initialize(repo, args.machine, args.skip_hooks)
        return 0
    except (OSError, ValueError, subprocess.CalledProcessError, EOFError) as exc:
        print(f'init: {exc}', file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print('\ninit: interrupted; rerun when ready', file=sys.stderr)
        return 130
