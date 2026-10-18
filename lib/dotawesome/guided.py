"""Interactive orchestration over the existing shared and component engines."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import tomllib

from . import bootstrap, cli, enrollment, niri
from .components import COMPONENTS
from .manager import Error, Manager, encode
from .packages import Packages
from .shared import validate_identity

SHARED_PACKAGES = ['git', 'chezmoi', 'age', 'fish', 'alacritty', 'github-cli', 'aws-cli-v2', 'gitleaks']
CHOICES = ('shared', 'nvim', 'tmux', 'niri')


def authenticate(ask):
    """Check quietly; only launch a machine-local login after an explicit choice."""
    remaining = []
    for name, check, login in (
        ('GitHub', ['gh', 'auth', 'status'], ['gh', 'auth', 'login']),
        ('AWS', ['aws', 'sts', 'get-caller-identity', '--profile', 'cof-tf'], ['aws', 'login', '--profile', 'cof-aws']),
    ):
        if not shutil.which(check[0]):
            remaining.append(f'{name}: install {check[0]} to sign in')
            continue
        try:
            ready = subprocess.run(check, capture_output=True, timeout=15).returncode == 0
        except (OSError, subprocess.TimeoutExpired):
            ready = False
        if ready:
            print(f'{name}: authenticated')
            continue
        print(f'{name}: login could not be verified (credentials or network may be unavailable).')
        if ask(f'Sign in to {name} now? [y/N]: ').strip().lower() in ('y', 'yes'):
            if subprocess.run(login).returncode == 0:
                try:
                    if subprocess.run(check, capture_output=True, timeout=15).returncode == 0:
                        print(f'{name}: authenticated')
                        continue
                except (OSError, subprocess.TimeoutExpired):
                    pass
        remaining.append(f'{name}: login still pending; rerun dotawesome to retry')
    return remaining


class Workflow:
    def __init__(self, manager, ask):
        self.manager = manager
        self.ask = ask
        self.packages = Packages(manager)
        self.preferences = manager.state / 'preferences.json'
        self.config = manager.config / 'chezmoi/chezmoi.toml'
        self.saved = json.loads(self.preferences.read_text()) if self.preferences.exists() else {}
        self.local = tomllib.loads(self.config.read_text()) if self.config.exists() else {}
        self.machine = self.local.get('data', {}).get('machine') or self.saved.get('machine')
        if self.machine is None:
            previous_profile = manager.records().get('niri', {}).get('profile_source')
            if previous_profile in ('g14', 'desktop'):
                self.machine = previous_profile
        if self.machine not in (None, 'g14', 'desktop'):
            raise Error(f'invalid saved machine selection; inspect {self.config} and {self.preferences}')
        self.remaining = []
        self.failed = False

    def confirm(self, prompt):
        return self.ask(prompt + ' [y/N]: ').strip().lower() in ('y', 'yes')

    def choose(self, prompt, options, default=None):
        while True:
            value = self.ask(prompt).strip().lower() or default
            if value in options:
                return value
            print('Choose ' + ', '.join(options) + '.')

    def profile(self, name):
        if name != 'niri':
            return None
        source = self.manager.records().get(name, {}).get('profile_source')
        return source or self.machine or 'generic'

    def details(self, name):
        component = COMPONENTS[name]
        profile = component.profile(self.manager.repo, self.profile(name))
        files = component.render(self.manager, profile)
        record = self.manager.records().get(name)
        if not record:
            return 'not managed', files, profile
        if any(self.manager.inspect(Path(p)) != e['installed'] for p, e in record['files'].items()):
            return 'local edits', files, profile
        expected = {str(p): encode(data, mode) for p, (data, mode) in files.items()}
        state = 'current' if expected == {p: e['installed'] for p, e in record['files'].items()} else 'updates available'
        return state, files, profile

    def enrollment(self):
        public = self.manager.repo / 'encryption/recipient.txt'
        recovery = public.with_name('identity.age')
        key = self.manager.config / 'chezmoi/key.txt'
        if public.is_file() and recovery.is_file():
            return 'identity present' if os.path.lexists(key) else 'recovery needed'
        if os.path.lexists(public) or os.path.lexists(recovery) or os.path.lexists(key):
            return 'partial enrollment; inspect existing files'
        return 'not enrolled'

    def shared_status(self):
        if not self.config.exists():
            return 'not initialized'
        if self.enrollment() != 'identity present':
            return self.enrollment()
        if not all(shutil.which(tool) for tool in ('chezmoi', 'age-keygen')):
            return 'inspection needs chezmoi and age'
        if Path(self.local.get('sourceDir', '')).expanduser().resolve() != self.manager.repo.resolve():
            return 'different checkout; inspect local chezmoi configuration'
        public = self.manager.repo / 'encryption/recipient.txt'
        try:
            validate_identity(self.manager.config / 'chezmoi/key.txt', public.read_text().strip())
        except ValueError as exc:
            return f'needs attention ({exc})'
        command = ['chezmoi', '--source', str(self.manager.repo), '--destination', str(self.manager.home),
                   '--config', str(self.config), '--no-pager', '--no-tty', 'status']
        result = subprocess.run(command, stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=15)
        if result.returncode:
            return 'inspection failed; choose setup for details'
        return 'differences to review' if result.stdout.strip() else 'current'

    def overview(self):
        print(f'Machine: {self.machine or "not selected"}')
        print(f'Encryption: {self.enrollment()}')
        print(f'Shared configs: {self.shared_status()}')
        for name in COMPONENTS:
            try:
                state, _, _ = self.details(name)
            except (Error, OSError, ValueError, KeyError, TypeError) as exc:
                state = f'needs attention ({exc})'
            print(f'{name}: {state}')
        tools = [name for name in ('pacman', 'chezmoi', 'age', 'gitleaks') if not shutil.which(name)]
        print('Tools: ' + ('missing ' + ', '.join(tools) if tools else 'available'))
        if (self.manager.state / 'transaction.json').exists():
            print('Interrupted deployment: recovery needed before setup.')

    def inspect(self):
        name = self.choose('Inspect [shared/nvim/tmux/niri]: ', CHOICES)
        if name == 'shared':
            if not self.config.exists() or not shutil.which('chezmoi'):
                print('Shared configuration is not initialized; choose setup to prepare it.')
                return 0
            print('Shared differences may contain decrypted secrets; use a private terminal.')
            command = ['chezmoi', '--source', str(self.manager.repo), '--destination', str(self.manager.home),
                       '--config', str(self.config), '--no-pager', 'diff']
            return subprocess.run(command).returncode
        _, files, _ = self.details(name)
        cli.show_diff(self.manager, name, files)
        return 0

    def selected(self):
        default = self.saved.get('components', list(CHOICES))
        if not isinstance(default, list) or not default or any(n not in CHOICES for n in default):
            default = list(CHOICES)
        print('Select components to set up/update. Unselected components are left as they are.')
        while True:
            reply = self.ask(f'Components [{" ".join(default)}] (or all): ').strip().lower()
            names = list(CHOICES) if reply == 'all' else list(dict.fromkeys(reply.replace(',', ' ').split())) if reply else default
            if names and all(n in CHOICES for n in names):
                return names
            print('Choose from: ' + ', '.join(CHOICES))

    def outside_niri(self, explain=True):
        try:
            niri.outside_session(self.manager)
            return True
        except Error as exc:
            if explain:
                self.remaining.append(f'Niri deferred: {exc}; log out and rerun dotawesome from KDE or a TTY')
            return False

    def resolve_edits(self, name, files):
        while True:
            choice = self.choose(f'{name} has local edits: [k]eep and skip, [v]iew diff, [r]estore deployed files after backing up edits: ', ('k', 'v', 'r'), 'k')
            if choice == 'k':
                self.remaining.append(f'{name}: local edits kept; update deferred')
                return False
            if choice == 'v':
                cli.show_diff(self.manager, name, files)
                continue
            with self.manager.lock():
                if name == 'niri':
                    niri.outside_session(self.manager)
                records = self.manager.records()
                changes = {}
                for p, entry in records[name]['files'].items():
                    before = self.manager.inspect(Path(p))
                    if before != entry['installed']:
                        changes[p] = {'before': before, 'after': entry['installed']}
                directory = self.manager.state / 'local-edits'
                directory.mkdir(parents=True, exist_ok=True, mode=0o700)
                saved = Path(tempfile.mkdtemp(prefix=name + '-', dir=directory)) / 'files.json'
                self.manager.save_json(saved, {p: e['before'] for p, e in changes.items()})
                self.manager.transaction(records, records, changes, reload_units=name == 'niri')
                print(f'Local edits backed up: {saved}')
            return True

    def setup(self):
        if self.machine is None:
            hardware = Path('/sys/class/dmi/id/product_name')
            if hardware.is_file():
                print(f'Detected hardware: {hardware.read_text().strip()}')
            self.machine = self.choose('Machine [g14/desktop]: ', ('g14', 'desktop'))
        names = self.selected()
        deferred = set()
        reports = {}
        for name in names:
            if name == 'shared':
                continue
            try:
                state, _, profile = self.details(name)
                reports[name] = COMPONENTS[name].check(self.manager, profile)
                if name == 'niri':
                    current = state == 'current' and not reports[name]['errors']
                    if not self.outside_niri(explain=not current):
                        deferred.add(name)
            except (Error, OSError, ValueError, KeyError, TypeError) as exc:
                self.remaining.append(f'{name}: deferred; {exc}')
                deferred.add(name)
        groups = {name: SHARED_PACKAGES if name == 'shared' else reports[name]['packages']
                  for name in names if name not in deferred}
        available = self.packages.installed() if shutil.which('pacman') else set()
        missing = sorted({p for required in groups.values() for p in required} - available)
        upgrade = any('requires >=' in error for name, report in reports.items() if name not in deferred for error in report['errors'])
        print(f'Setup/update {self.machine}: ' + ', '.join(names))
        print('Missing packages: ' + (', '.join(missing) if missing else 'none'))
        for message in self.remaining:
            print(message)
        for name, report in reports.items():
            print(name + ':')
            cli.report_checks(report)
        if not self.confirm('Proceed with this setup?'):
            print('Setup cancelled; no changes made.')
            return 0
        if (self.manager.state / 'transaction.json').exists():
            if not self.confirm('Recover the interrupted deployment before continuing?'):
                print('Setup deferred until recovery.')
                return 0
            cli.recover_before_mutation(self.manager, 'setup')
        preferences = {'machine': self.machine, 'components': names}
        if preferences != self.saved:
            self.manager.save_json(self.preferences, preferences)
        if missing or upgrade:
            print('Package installation performs a full system upgrade:')
            print('  sudo pacman -Syu --needed ' + ' '.join(dict.fromkeys(p for names in groups.values() for p in names)))
            if shutil.which('pacman') and self.confirm('Install/update the required packages?'):
                self.packages.install_many(groups)
                available = self.packages.installed()
        for name in names:
            if name in deferred:
                continue
            if set(groups[name]) - available:
                self.remaining.append(f'{name}: deferred; required packages are missing')
                continue
            try:
                if name == 'shared':
                    self.setup_shared()
                    continue
                state, files, profile = self.details(name)
                report = COMPONENTS[name].check(self.manager, profile)
                if report['errors']:
                    cli.report_checks(report)
                    self.remaining.append(f'{name}: deferred; resolve the dependency/configuration errors above')
                    continue
                if state == 'current':
                    print(f'{name}: current; no changes needed')
                    continue
                if state == 'local edits' and not self.resolve_edits(name, files):
                    continue
                self.manager.preview(name, files)
                cli.show_diff(self.manager, name, files)
                if not self.confirm(f'Validate, back up and apply {name}?'):
                    self.remaining.append(f'{name}: deferred by choice')
                    continue
                arguments = ['apply', name]
                if name == 'niri':
                    arguments += ['--profile', self.profile(name)]
                if cli.main(self.manager.repo, arguments, manager=self.manager):
                    self.failed = True
                    self.remaining.append(f'{name}: failed; inspect the error above and rerun')
            except (Error, OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as exc:
                self.failed = True
                self.remaining.append(f'{name}: deferred; {exc}')
        self.remaining.extend(authenticate(self.ask))
        self.summary()
        return int(self.failed)

    def setup_shared(self):
        state = self.enrollment()
        if state.startswith('partial'):
            raise Error('partial age enrollment; existing files preserved; inspect them before retrying')
        if state == 'not enrolled':
            print('This repository has no age enrollment. Create one dedicated key and verify its encrypted recovery copy.')
            print('Store the recovery passphrase separately in 1Password or an offline recovery record.')
            if not self.confirm("Create the repository's first age enrollment now?"):
                self.remaining.append('Shared configuration: enrollment deferred')
                return
            result = enrollment.main(self.manager.repo)
            if result == 130:
                raise KeyboardInterrupt()
            if result:
                raise Error('age enrollment did not finish; inspect existing enrollment files before retrying')
            self.remaining.append('Publish enrollment: review, commit and push encryption/recipient.txt and encryption/identity.age before setting up another machine')
        try:
            result = bootstrap.initialize(self.manager.repo, machine=self.machine, ask=self.ask, skip_current=True)
        except ValueError as exc:
            if not any(text in str(exc) for text in ('existing pre-commit hook', 'core.hooksPath')):
                raise
            print(str(exc))
            print('Chain hooks/pre-commit from your existing hook to enable secret scanning.')
            if not self.confirm('Have you chained the scanner and want to keep your existing hook setup?'):
                self.remaining.append('Shared configuration: hook integration deferred')
                return
            result = bootstrap.initialize(self.manager.repo, machine=self.machine, ask=self.ask,
                                          skip_current=True, skip_hooks=True)
        if result not in ('applied', 'current'):
            self.remaining.append('Shared configuration: deferred; rerun to finish')

    def restore(self):
        names = list(self.manager.records())
        if not names:
            print('No component deployments to restore.')
            return 0
        print('Restore a component\'s original configuration; application data is retained.')
        print('Shared-file adoption backups: ' + str(self.manager.state / 'adoption'))
        name = self.choose('Component [' + '/'.join(names) + ']: ', names)
        if name == 'niri' and not self.outside_niri():
            self.summary()
            return 0
        record = self.manager.records()[name]
        modified = any(self.manager.inspect(Path(p)) != e['installed'] for p, e in record['files'].items())
        if modified and not self.resolve_edits(name, {}):
            self.summary()
            return 0
        if cli.main(self.manager.repo, ['plan', name, '--remove'], manager=self.manager):
            return 1
        if not self.confirm(f'Restore the original {name} configuration?'):
            return 0
        arguments = ['remove', name]
        candidates = self.packages.plan(name)
        if candidates:
            print('Eligible recorded package additions: ' + ', '.join(candidates))
            if self.confirm('Also remove these packages with pacman?'):
                arguments += ['--remove-packages']
        return cli.main(self.manager.repo, arguments, manager=self.manager)

    def summary(self):
        if self.remaining:
            print('\nRemaining steps:')
            for item in self.remaining:
                print('  ' + item)
        else:
            print('\nSelected configuration is ready.')


def main(repo, manager=None, ask=None, terminal=None):
    if not (sys.stdin.isatty() if terminal is None else terminal):
        print('Guided setup requires a terminal. Use dotawesome status, plan, check, apply or --help for explicit commands.', file=sys.stderr)
        return 1
    try:
        workflow = Workflow(manager or Manager(repo), ask or input)
        workflow.overview()
        print('\n1. Set up / update this machine\n2. Inspect status and differences\n3. Restore previous component configuration\n0. Exit')
        choice = workflow.choose('Action [1]: ', ('0', '1', '2', '3'), '1')
        if choice == '0':
            return 0
        return {'1': workflow.setup, '2': workflow.inspect, '3': workflow.restore}[choice]()
    except (Error, OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as exc:
        print(f'dotawesome: {exc}', file=sys.stderr)
        return 1
    except (EOFError, KeyboardInterrupt):
        print('\nSetup stopped. Rerun dotawesome to inspect status and continue.')
        return 130
