import argparse
import difflib
import json
import shutil
from pathlib import Path
import subprocess
import sys

from .manager import Error, Manager, encode
from . import niri
from .packages import Packages


def main(repo):
    parser = argparse.ArgumentParser(description='Manage explicitly selected dotawesome components (repository-only until activate).')
    sub = parser.add_subparsers(dest='command', required=True)
    sub.add_parser('list')
    for name in ('check', 'build', 'activate'):
        command = sub.add_parser(name)
        command.add_argument('component', choices=['niri'])
        command.add_argument('--profile', required=True, help='generic, g14, or a TOML profile path')
        if name == 'activate':
            command.add_argument('--dry-run', action='store_true')
            command.add_argument('--install-packages', action='store_true')
    status = sub.add_parser('status'); status.add_argument('component', nargs='?', choices=['niri'])
    deactivate = sub.add_parser('deactivate')
    deactivate.add_argument('component', choices=['niri'])
    deactivate.add_argument('--remove-packages', action='store_true', help='remove recorded additions that are no longer needed')
    deactivate.add_argument('--dry-run', action='store_true', help='review restoration and optional package cleanup without changes')
    args = parser.parse_args()
    manager = Manager(repo)
    packages = Packages(manager)
    try:
        if args.command == 'list':
            print('niri  Niri + Noctalia (generic, g14)'); return 0
        if args.command == 'status':
            records = manager.records()
            package_record = packages.records().get('niri', {})
            if package_record.get('roots') or package_record.get('pending'):
                print('niri package provenance: ' + (', '.join(package_record.get('roots', [])) or 'installation interrupted'))
            if (manager.state / 'transaction.json').exists():
                print('Interrupted deployment: recovery pending on next mutation.')
            for component in ([args.component] if args.component else ['niri']):
                record = records.get(component)
                if not record:
                    print(f'{component}: inactive; native and session validation pending'); continue
                print(f'{component}: {record["snapshot"]}\n  {record["native_validation"]}')
                for path, entry in record['files'].items():
                    if manager.inspect(Path(path)) != entry['installed']:
                        print(f'  MODIFIED {path}')
                if record.get('profile_source'):
                    try:
                        candidate = niri.render(manager, niri.profile(repo, record['profile_source']))
                        generated = {str(p): encode(data, mode) for p, (data, mode) in candidate.items()}
                        deployed = {p: e['installed'] for p, e in record['files'].items()}
                        if generated != deployed:
                            print('  checkout differs: build and activate to apply pending changes')
                    except (Error, OSError, ValueError) as exc:
                        print(f'  checkout comparison unavailable: {exc}')
                overrides = manager.state / 'niri-shell/noctalia/settings.toml'
                if overrides.exists(): print(f'  GUI overrides: {overrides}')
            return 0
        if args.command == 'deactivate':
            if not args.dry_run:
                niri.outside_session(manager)
            # Dry runs inspect only; a real mutation recovers the journal first.
            if args.dry_run:
                manager.preview(args.component, {})
                for path in manager.records().get(args.component, {}).get('files', {}):
                    print(f'restore original / remove managed file: {path}')
            removal = packages.plan(args.component) if args.remove_packages else []
            if args.remove_packages:
                print('Package operation: ' + ('sudo pacman -Ru ' + ' '.join(removal) if removal else 'none; no recorded additions currently eligible'))
            if args.dry_run:
                print('User unit operation: systemctl --user daemon-reload; no changes performed.')
                return 0
            manager.deactivate(args.component)
            packages.release(args.component)
            if args.remove_packages:
                removed = packages.remove(args.component)
                print('Removed packages: ' + (', '.join(removed) or 'none'))
            print('niri deactivated; original files restored. Shell state retained.')
            return 0
        if args.command == 'activate' and not args.dry_run and (manager.state / 'transaction.json').exists():
            niri.outside_session(manager)
            with manager.lock():
                pass  # Recover before preview compares managed files against the old ledger.
        p = niri.profile(repo, args.profile)
        files = niri.render(manager, p)
        changes = manager.preview(args.component, files)
        if args.command == 'build':
            snapshot = manager.snapshot(args.component, files)
            print(f'Build: {snapshot}')
            if all(shutil.which(c) for c in ('niri', 'noctalia', 'systemd-analyze')):
                niri.validate(manager, files); print('Native validation passed; session acceptance pending.')
            else:
                print('Native validation PENDING: validators unavailable; activation will require them.')
            return 0
        report = niri.check(manager, p)
        print(json.dumps(report, indent=2))
        print('\n'.join(changes))
        if args.command == 'check':
            print('Native config validation occurs during build/activation; session acceptance pending.')
            return 1 if report['errors'] else 0
        print('Package operation: ' + ('sudo pacman -Syu --needed ' + ' '.join(report['packages']) if args.install_packages else 'none (require installed dependencies)'))
        for dest, (data, _) in files.items():
            before = manager.inspect(dest)
            if before and 'data' in before:
                import base64
                old = base64.b64decode(before['data']).decode(errors='replace')
            else:
                old = ''
            print(''.join(difflib.unified_diff(old.splitlines(True), data.decode().splitlines(True), fromfile=str(dest), tofile=f'{dest} (generated)')), end='')
        print('User unit operation: systemctl --user daemon-reload; no start/enable/session switch.')
        if args.dry_run:
            return 0
        niri.outside_session(manager)
        if args.install_packages:
            packages.install(args.component, report['packages'])
            report = niri.check(manager, p)
        if report['errors']:
            raise Error('preflight failed; no configuration deployed')
        snapshot = manager.snapshot(args.component, files)
        niri.validate(manager, files)
        packages.require(args.component, report['packages'])
        manager.deploy(args.component, files, snapshot, args.profile if args.profile in ('generic', 'g14') else str(Path(args.profile).resolve()))
        print('Deployed. Log out of Plasma, select the packaged Niri session, and log in. Keep Plasma as default. Session acceptance remains pending.')
        return 0
    except (Error, OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as exc:
        print(f'dotawesome: {exc}', file=sys.stderr)
        return 1
