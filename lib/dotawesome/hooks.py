"""Repository hook installation shared by machine initialization."""
from pathlib import Path
import subprocess


def install(repo):
    custom = subprocess.run(['git', '-C', str(repo), 'config', '--get', 'core.hooksPath'], capture_output=True, text=True)
    if custom.returncode == 0:
        raise ValueError('core.hooksPath already configured; chain hooks/pre-commit manually')
    hook = Path(subprocess.check_output(['git', '-C', str(repo), 'rev-parse', '--path-format=absolute', '--git-path', 'hooks/pre-commit'], text=True).strip())
    source = repo / 'hooks/pre-commit'
    if hook.exists() or hook.is_symlink():
        if not hook.is_symlink() and hook.read_bytes() == source.read_bytes():
            return
        raise ValueError('existing pre-commit hook preserved; chain hooks/pre-commit manually')
    hook.parent.mkdir(parents=True, exist_ok=True)
    with hook.open('xb') as output:
        output.write(source.read_bytes())
    hook.chmod(0o755)
    print(f'Installed {hook}')
