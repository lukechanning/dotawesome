"""Small, explicit component workflows with read-only plans."""
import argparse
import base64
import difflib
import json
import shutil
from pathlib import Path
import subprocess
import sys

from . import niri
from .components import COMPONENTS
from .manager import Error, Manager, encode
from .packages import Packages


def parser():
    result = argparse.ArgumentParser(description='Manage selected workstation configuration with backups and restoration.')
    commands = result.add_subparsers(dest='command', required=True)
    commands.add_parser('list', help='list supported components')
    status = commands.add_parser('status', help='compare deployed files and source')
    status.add_argument('component', nargs='?', choices=COMPONENTS)
    for name, help_text in (('check', 'check dependencies and configuration'),
                            ('plan', 'preview changes without writing files'),
                            ('apply', 'validate, back up, and deploy configuration'),
                            ('remove', 'restore original files')):
        command = commands.add_parser(name, help=help_text)
        command.add_argument('component', choices=COMPONENTS)
        if name in ('check', 'plan', 'apply'):
            command.add_argument('--profile', help='Niri only: generic, g14, or a TOML path')
        if name in ('plan', 'apply'):
            command.add_argument('--live', action='store_true', help='Niri only: appearance-only update inside the session')
        if name == 'plan':
            command.add_argument('--remove', action='store_true', help='preview restoration and eligible package cleanup')
        if name == 'apply':
            command.add_argument('--install-packages', action='store_true', help='opt in to Arch package installation and full upgrade')
        if name == 'remove':
            command.add_argument('--remove-packages', action='store_true', help='remove eligible recorded package additions')
    return result


def report_checks(report):
    for level in ('errors', 'warnings'):
        for message in report.get(level, []):
            print(f'  {level[:-1]}: {message}')
    if not report['errors']:
        print('  dependencies: ready')


def show_diff(manager, component, files):
    record = manager.records().get(component, {})
    targets = {p: encode(data, mode) for p, (data, mode) in files.items()}
    for path, entry in record.get('files', {}).items():
        if Path(path) not in targets:
            targets[Path(path)] = entry['original']
    changed = 0
    for path, after in targets.items():
        before = manager.inspect(path)
        if before == after:
            continue
        changed += 1
        action = 'remove' if after is None else 'create' if before is None else 'replace'
        print(f'  {action}: {path}')
        def content(value):
            if value is None:
                return ''
            if 'link' in value:
                return 'symlink -> ' + value['link'] + '\n'
            return base64.b64decode(value['data']).decode(errors='replace')
        diff = ''.join(difflib.unified_diff(content(before).splitlines(True), content(after).splitlines(True),
                                           fromfile=str(path), tofile=f'{path} (planned)'))
        print(diff, end='' if diff.endswith('\n') else '\n')
    print(f'{changed} file(s) changed.')


def status(manager, packages, selected):
    records = manager.records()
    if (manager.state / 'transaction.json').exists():
        print('Interrupted deployment: recovery pending on next mutation.')
    package_records = packages.records()
    for name in ([selected] if selected else COMPONENTS):
        record = records.get(name)
        if not record:
            print(f'{name}: inactive')
        else:
            states = []
            modified = [p for p, entry in record['files'].items() if manager.inspect(Path(p)) != entry['installed']]
            if modified:
                states.append('locally-modified')
            try:
                component = COMPONENTS[name]
                profile = component.profile(manager.repo, record.get('profile_source'))
                files = component.render(manager, profile)
                generated = {str(p): encode(data, mode) for p, (data, mode) in files.items()}
                if generated != {p: e['installed'] for p, e in record['files'].items()}:
                    states.append('source-changed')
            except (Error, OSError, ValueError, KeyError, TypeError) as exc:
                states.append(f'source unavailable ({exc})')
            print(f'{name}: {", ".join(states) if states else "current"}')
            for path in modified:
                print(f'  modified: {path}')
        owned = package_records.get(name, {})
        if owned.get('roots') or owned.get('pending'):
            print('  package additions: ' + (', '.join(owned.get('roots', [])) or 'installation interrupted'))
        if name == 'niri':
            overrides = manager.state / 'niri-shell/noctalia/settings.toml'
            if overrides.exists():
                print(f'  GUI overrides: {overrides}')


def recover_before_mutation(manager, component):
    journal = manager.state / 'transaction.json'
    if journal.exists():
        # Legacy journals always included a session-level systemd reload.
        if json.loads(journal.read_text()).get('reload_units', True) and component != 'niri':
            niri.outside_session(manager)
        with manager.lock():
            pass


def removal(manager, packages, args):
    component = args.component
    preview = args.command == 'plan'
    if not preview:
        if component == 'niri':
            niri.outside_session(manager)
        recover_before_mutation(manager, component)
    manager.preview(component, {})
    if preview:
        print(f'{component}: restoration plan')
        show_diff(manager, component, {})
    cleanup = preview or args.remove_packages
    candidates = packages.plan(component) if cleanup else []
    if cleanup:
        print('Package cleanup: ' + ('sudo pacman -Ru ' + ' '.join(candidates) if candidates else 'none eligible'))
    if preview:
        return 0
    manager.deactivate(component)
    packages.release(component)
    if args.remove_packages:
        removed = packages.remove(component)
        print('Removed packages: ' + (', '.join(removed) or 'none'))
    print(f'{component}: removed; original files restored. Application data retained.')
    return 0


def main(repo):
    args = parser().parse_args()
    manager = Manager(repo)
    packages = Packages(manager)
    try:
        if args.command == 'list':
            for name, component in COMPONENTS.items():
                print(f'{name:6} {component.description}')
            return 0
        if args.command == 'status':
            status(manager, packages, args.component)
            return 0
        live = getattr(args, 'live', False)
        if live and args.component != 'niri':
            raise Error('--live is only valid for niri; application configs can be applied normally')
        if live and getattr(args, 'install_packages', False):
            raise Error('live update cannot install packages; install outside Niri')
        if args.command == 'plan' and args.remove and (args.profile or live):
            raise Error('plan --remove does not accept --profile or --live')
        if args.command == 'remove' or (args.command == 'plan' and args.remove):
            return removal(manager, packages, args)
        component = COMPONENTS[args.component]
        profile = component.profile(manager.repo, args.profile)
        if args.command == 'apply' and not live:
            if component.name == 'niri':
                niri.outside_session(manager)
            recover_before_mutation(manager, component.name)
        files = component.render(manager, profile)
        if live:
            niri.validate_live_update(manager, files)
            palette_dir = manager.config / 'dotawesome/niri/noctalia/noctalia/palettes'
            files = dict(sorted(files.items(), key=lambda item: (
                0 if item[0].parent == palette_dir else
                2 if item[0] == manager.config / 'niri/config.kdl' else 1)))
        manager.preview(component.name, files)
        report = component.check(manager, profile)
        print(f'{component.name}: {args.command}')
        report_checks(report)
        if args.command == 'plan':
            show_diff(manager, component.name, files)
            print('Packages required: ' + ', '.join(report['packages']))
            print('No changes made. Native validation runs during check/apply.')
            return 0
        if args.command == 'check':
            if not report['errors']:
                component.validate(manager, files)
                print('  configuration: valid')
            return int(bool(report['errors']))
        if args.install_packages:
            if component.name != 'niri':
                if not shutil.which('pacman'):
                    raise Error('--install-packages requires Arch/CachyOS with pacman')
            packages.install(component.name, report['packages'])
            report = component.check(manager, profile)
            report_checks(report)
        if report['errors']:
            raise Error('preflight failed; no configuration deployed')
        component.validate(manager, files)
        snapshot = manager.snapshot(component.name, files)
        if not live:
            packages.require(component.name, report['packages'])
        source = args.profile
        if source and source not in ('generic', 'g14'):
            source = str(Path(source).resolve())
        preflight = (lambda: niri.validate_live_update(manager, files)) if live else None
        manager.deploy(component.name, files, snapshot, source, preflight=preflight,
                       reload_units=component.name == 'niri',
                       validation='passed; session acceptance pending' if component.name == 'niri' else 'passed')
        print(f'{component.name}: applied' + (' live appearance update.' if live else '.'))
        print(component.guidance)
        return 0
    except (Error, OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as exc:
        print(f'dotawesome: {exc}', file=sys.stderr)
        return 1
