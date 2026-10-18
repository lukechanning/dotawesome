"""The shared-file boundary and age identity publication policy."""
import os
from pathlib import Path
import stat
import subprocess
import tempfile

# Includes entire component directories: ownership cannot silently expand into them.
COMPONENT_TARGETS = ('.tmux.conf', '.config/nvim', '.config/niri', '.config/dotawesome',
                     '.config/systemd/user/dotawesome-niri-noctalia.service',
                     '.config/systemd/user/niri.service.d', '.config/xdg-desktop-portal/niri-portals.conf',
                     '.config/vivaldi-stable.conf', '.config/signal-desktop-flags.conf')


def check_ownership(targets, directories=()):
    for name in targets:
        target = Path(name)
        if target.is_absolute() or '..' in target.parts:
            raise ValueError(f'unsafe shared target: {name}')
        for owned in map(Path, COMPONENT_TARGETS):
            if target == owned or target in owned.parents or owned in target.parents:
                raise ValueError(f'chezmoi/component ownership overlap: {name}')

    for name in directories:
        target = Path(name)
        if target.is_absolute() or '..' in target.parts:
            raise ValueError(f'unsafe shared directory: {name}')
        for owned in map(Path, COMPONENT_TARGETS):
            if target == owned or owned in target.parents:
                raise ValueError(f'chezmoi/component ownership overlap: {name}')


def validate_identity(path, recipient):
    if path.is_symlink() or not path.is_file():
        raise ValueError('identity must be a regular file')
    if stat.S_IMODE(path.stat().st_mode) != 0o600:
        raise ValueError('identity must have mode 0600')
    result = subprocess.run(['age-keygen', '-y', str(path)], capture_output=True, text=True)
    if result.returncode or result.stdout.strip() != recipient:
        raise ValueError('invalid identity or recipient mismatch; existing identity preserved')


def recover_identity(identity, recovery, recipient, runner=subprocess.run):
    """Decrypt privately, validate, then publish without replacing an existing key."""
    if os.path.lexists(identity):
        validate_identity(identity, recipient)
        return
    identity.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
    fd, name = tempfile.mkstemp(prefix='.identity-', dir=identity.parent)
    temporary = Path(name)
    try:
        with os.fdopen(fd, 'wb') as output:
            runner(['age', '--decrypt', str(recovery)], stdout=output, check=True)
            output.flush()
            os.fsync(output.fileno())
        validate_identity(temporary, recipient)
        # link is atomic and fails if another process published a key meanwhile.
        os.link(temporary, identity)
    finally:
        temporary.unlink(missing_ok=True)
