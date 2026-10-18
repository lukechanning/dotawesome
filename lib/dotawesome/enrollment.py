"""One-time, terminal-only enrollment of a dedicated X25519 identity."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile

from .shared import validate_identity


def main(repo):
    previous = os.umask(0o077)
    try:
        return enroll(repo)
    finally:
        os.umask(previous)


def enroll(repo):
    directory = Path(os.environ.get('XDG_CONFIG_HOME', Path.home() / '.config')) / 'chezmoi'
    identity = directory / 'key.txt'
    public = repo / 'encryption/recipient.txt'
    recovery = repo / 'encryption/identity.age'
    try:
        if any(os.path.lexists(p) for p in (identity, public, recovery)):
            raise ValueError('enrollment already exists or is partial; inspect it manually; nothing overwritten')
        if not sys.stdin.isatty():
            raise ValueError('enrollment requires a local terminal for the recovery passphrase')
        directory.mkdir(parents=True, exist_ok=True, mode=0o700)
        public.parent.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(prefix='.enroll-', dir=directory) as temporary:
            work = Path(temporary)
            key = work / 'key.txt'
            with key.open('xb') as output:
                subprocess.run(['age-keygen'], stdout=output, check=True)
            recipient = subprocess.check_output(['age-keygen', '-y', str(key)], text=True).strip()
            validate_identity(key, recipient)
            encrypted = work / 'identity.age'
            print('Choose a recovery passphrase; store it separately in 1Password or an offline recovery record.', flush=True)
            subprocess.run(['age', '--passphrase', '--output', str(encrypted), str(key)], check=True)
            verified = work / 'verified.txt'
            print('Re-enter that passphrase to verify recovery before enrollment is published.', flush=True)
            with verified.open('xb') as output:
                subprocess.run(['age', '--decrypt', str(encrypted)], stdout=output, check=True)
            validate_identity(verified, recipient)
            # Copy ciphertext to the repository filesystem for atomic link publication.
            fd, name = tempfile.mkstemp(prefix='.enroll-', dir=public.parent)
            staged = Path(name)
            try:
                with os.fdopen(fd, 'wb') as output:
                    output.write(encrypted.read_bytes())
                os.link(staged, recovery)
                with public.open('x') as output:
                    output.write(recipient + '\n')
                os.link(key, identity)
            finally:
                staged.unlink(missing_ok=True)
        print('Recovery verified. Commit only encryption/recipient.txt and encryption/identity.age.')
        return 0
    except (OSError, ValueError, subprocess.CalledProcessError) as exc:
        print(f'enroll-age: {exc}', file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        print('\nenroll-age: interrupted; inspect enrollment before retrying', file=sys.stderr)
        return 130
