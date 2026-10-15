"""Small shared deployment engine. No component discovery or installer plugins."""
import base64
from contextlib import contextmanager
import fcntl
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import tempfile


class Error(Exception):
    pass


def run(argv, **kwargs):
    result = subprocess.run(argv, capture_output=True, text=True, timeout=60, **kwargs)
    if result.returncode:
        raise Error(f"{' '.join(argv)}: {result.stderr.strip() or result.stdout.strip()}")
    return '\n'.join(part.strip() for part in (result.stdout, result.stderr) if part.strip())


def encode(content, mode):
    return {'data': base64.b64encode(content).decode(), 'mode': mode}


class Manager:
    def __init__(self, repo, home=None, build_root=None, runner=run):
        self.repo = Path(repo)
        self.home = Path(home or Path.home()).absolute()
        self.build_root = Path(build_root or self.repo / '.build/dotawesome')
        self.config = Path(os.environ.get('XDG_CONFIG_HOME', self.home / '.config')) if home is None else self.home / '.config'
        state = Path(os.environ.get('XDG_STATE_HOME', self.home / '.local/state')) if home is None else self.home / '.local/state'
        self.state = state / 'dotawesome'
        self.runner = runner

    def records(self):
        path = self.state / 'installed.json'
        return json.loads(path.read_text()) if path.exists() else {}

    @contextmanager
    def lock(self):
        self.state.mkdir(parents=True, exist_ok=True, mode=0o700)
        with (self.state / 'lock').open('a') as lock:
            fcntl.flock(lock, fcntl.LOCK_EX)
            self.recover()
            yield

    def inspect(self, path):
        if path.is_symlink():
            return {'link': os.readlink(path)}
        if not path.exists():
            return None
        if not path.is_file():
            raise Error(f'not a regular file: {path}')
        return encode(path.read_bytes(), stat.S_IMODE(path.stat().st_mode))

    def safe_path(self, path):
        if not path.is_absolute() or '..' in path.parts:
            raise Error(f'invalid destination: {path}')
        for parent in path.parents:
            if parent.is_symlink():
                raise Error(f'symlink parent: {parent}')
            if parent.exists() and not parent.is_dir():
                raise Error(f'not a directory: {parent}')

    def preview(self, component, files):
        records = self.records()
        destinations = set(map(str, files))
        for a in destinations:
            self.safe_path(Path(a))
            for b in destinations:
                if a != b and Path(a) in Path(b).parents:
                    raise Error(f'overlapping destinations: {a}, {b}')
            for owner, record in records.items():
                for b in record['files']:
                    if owner != component and (a == b or Path(a) in Path(b).parents or Path(b) in Path(a).parents):
                        raise Error(f'{a} owned by {owner}')
            self.inspect(Path(a))
        old = records.get(component, {}).get('files', {})
        for p, entry in old.items():
            self.safe_path(Path(p))
            if self.inspect(Path(p)) != entry['installed']:
                raise Error(f'modified managed file: {p}; preserve your edit and restore the recorded file before continuing')
        return [f"{'replace / back up' if self.inspect(p) is not None else 'create'} {p}" for p in files]

    def write(self, path, value):
        """Replace the directory entry, never follow its final symlink."""
        self.safe_path(path)
        if value is None:
            path.unlink(missing_ok=True)
            return
        path.parent.mkdir(parents=True, exist_ok=True)
        fd, name = tempfile.mkstemp(prefix='.dotawesome-', dir=path.parent)
        temp = Path(name)
        try:
            if 'link' in value:
                os.close(fd); temp.unlink(); temp.symlink_to(value['link'])
            else:
                with os.fdopen(fd, 'wb') as stream:
                    stream.write(base64.b64decode(value['data'])); stream.flush(); os.fsync(stream.fileno())
                temp.chmod(value['mode'])
            temp.replace(path)
        finally:
            temp.unlink(missing_ok=True)

    def save_json(self, path, value):
        self.write(path, encode((json.dumps(value, indent=2, sort_keys=True) + '\n').encode(), 0o600))

    def recover(self):
        journal = self.state / 'transaction.json'
        if not journal.exists():
            return
        tx = json.loads(journal.read_text())
        # Don't destroy edits made after an interrupted deployment.
        for p, change in tx['changes'].items():
            if self.inspect(Path(p)) not in (change['before'], change['after']):
                raise Error(f'interrupted transaction: unexpected edit at {p}; keep transaction.json for recovery')
        for p, change in reversed(list(tx['changes'].items())):
            self.write(Path(p), change['before'])
        self.save_json(self.state / 'installed.json', tx['records'])
        self.runner(['systemctl', '--user', 'daemon-reload'])
        journal.unlink()

    def transaction(self, records, updated, changes):
        journal = self.state / 'transaction.json'
        self.save_json(journal, {'records': records, 'changes': changes})
        try:
            for p, change in changes.items():
                self.write(Path(p), change['after'])
            self.runner(['systemctl', '--user', 'daemon-reload'])
            self.save_json(self.state / 'installed.json', updated)
        except BaseException:
            # Leave the journal if rollback itself fails: next mutation recovers it.
            for p, change in reversed(list(changes.items())):
                if self.inspect(Path(p)) != change['before']:
                    self.write(Path(p), change['before'])
            self.save_json(self.state / 'installed.json', records)
            try:
                self.runner(['systemctl', '--user', 'daemon-reload'])
            except Exception:
                raise
            journal.unlink()
            raise
        journal.unlink()

    def deploy(self, component, files, snapshot, profile_source=None):
        with self.lock():
            self.preview(component, files)
            records = self.records()
            old = records.get(component, {}).get('files', {})
            installed = {}
            changes = {}
            for p, (data, mode) in files.items():
                key = str(p); before = self.inspect(p); after = encode(data, mode)
                installed[key] = {'original': old[key]['original'] if key in old else before, 'installed': after}
                changes[key] = {'before': before, 'after': after}
            for p, entry in old.items():
                if p not in installed:
                    changes[p] = {'before': entry['installed'], 'after': entry['original']}
            updated = dict(records, **{component: {'snapshot': str(snapshot), 'profile_source': profile_source, 'files': installed, 'native_validation': 'passed; session acceptance pending'}})
            self.transaction(records, updated, changes)

    def deactivate(self, component):
        with self.lock():
            records = self.records()
            if component not in records:
                return
            self.preview(component, {})
            changes = {p: {'before': e['installed'], 'after': e['original']} for p, e in records[component]['files'].items()}
            updated = dict(records); del updated[component]
            self.transaction(records, updated, changes)

    def snapshot(self, component, files):
        manifest = {str(p): encode(data, mode) for p, (data, mode) in sorted(files.items())}
        digest = hashlib.sha256(json.dumps(manifest, sort_keys=True).encode()).hexdigest()
        target = self.build_root / component / digest
        if target.exists():
            for i, value in enumerate(manifest.values()):
                if self.inspect(target / str(i)) != dict(value, mode=0o444):
                    raise Error(f'tampered snapshot: {target}')
            if json.loads((target / 'manifest.json').read_text()) != manifest:
                raise Error(f'tampered snapshot manifest: {target}')
            return target
        target.parent.mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryDirectory(dir=target.parent) as work:
            stage = Path(work) / 'snapshot'; stage.mkdir()
            for i, value in enumerate(manifest.values()):
                self.write(stage / str(i), dict(value, mode=0o444))
            (stage / 'manifest.json').write_text(json.dumps(manifest, indent=2, sort_keys=True) + '\n')
            (stage / 'manifest.json').chmod(0o444)
            stage.rename(target)
        return target
