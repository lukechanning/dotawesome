"""Opt-in Arch package cleanup with provenance separate from config backups."""
import json
import os
import re
import subprocess

from .manager import Error


class Packages:
    def __init__(self, manager):
        self.manager = manager
        self.path = manager.state / 'packages.json'

    def records(self):
        return json.loads(self.path.read_text()) if self.path.exists() else {}

    def names(self, output):
        if output.strip() == 'there is nothing to do':
            return set()
        names = output.splitlines()
        if any(not re.fullmatch(r'[a-zA-Z0-9@_+][a-zA-Z0-9@_.+:-]*', name) for name in names):
            raise Error('unexpected pacman package-list output; package operation refused')
        return set(names)

    def installed(self):
        return self.names(self.manager.runner(['pacman', '-Qq']))

    def require(self, component, names):
        with self.manager.lock():
            records = self.records()
            entry = records.setdefault(component, {'added': [], 'roots': []})
            entry['required'] = list(names)
            self.manager.save_json(self.path, records)

    def release(self, component):
        # Keep installation provenance so cleanup remains available after deactivation.
        if component not in self.records():
            return
        with self.manager.lock():
            records = self.records()
            records[component]['required'] = []
            self.manager.save_json(self.path, records)

    def install(self, component, required):
        self.install_many({component: required})

    def install_many(self, requirements):
        """One explicit upgrade; retain before/after provenance for each owner."""
        if not requirements:
            return
        required = list(dict.fromkeys(p for names in requirements.values() for p in names))
        with self.manager.lock():
            records = self.records()
            if any(entry.get('pending') or entry.get('removing') for entry in records.values()):
                raise Error('interrupted package installation: inspect packages.json and pacman logs before retrying')
            before = self.installed()
            entries = {}
            for component, names in requirements.items():
                entry = records.setdefault(component, {'added': [], 'roots': []})
                entry.update(required=list(names), pending=True)
                entries[component] = entry
            self.manager.save_json(self.path, records)
            try:
                result = subprocess.run(['sudo', 'pacman', '-Syu', '--needed', *required])
            finally:
                # Capture additions even when pacman fails or is interrupted.
                after = self.installed()
                for component, entry in entries.items():
                    entry['added'] = sorted(set(entry['added']) | (after - before))
                    entry['roots'] = sorted(set(entry['roots']) | (set(requirements[component]) & (after - before)))
                    entry['pending'] = False
                self.manager.save_json(self.path, records)
            if result.returncode:
                raise Error('package update failed; additions recorded for optional cleanup; no configuration deployed')

    def plan(self, component):
        records = self.records()
        entry = records.get(component, {})
        if entry.get('pending') or entry.get('removing'):
            raise Error('interrupted package operation: cleanup provenance needs manual review of packages.json and pacman logs')
        if not entry.get('roots'):
            return []
        installed = self.installed()
        protected = set()
        for owner, other in records.items():
            if owner != component:
                protected.update(other.get('required', []))
        roots = (set(entry['roots']) & installed) - protected
        if not roots:
            return []
        # Ask pacman for the currently unneeded dependency closure. It may contain
        # preexisting dependencies: intersect with our before/after additions.
        closure = self.names(self.manager.runner(['pacman', '-Rsu', '--print-format', '%n', *sorted(roots)], env=dict(os.environ, LC_ALL='C')))
        candidates = (closure & set(entry['added']) & installed) - protected
        return sorted(candidates)

    def remove(self, component):
        with self.manager.lock():
            if component in self.manager.records():
                raise Error('remove the component before removing its packages')
            candidates = self.plan(component)
            if not candidates:
                return []
            records = self.records()
            entry = records[component]
            entry['removing'] = True
            self.manager.save_json(self.path, records)
            # No recursion, cascade, nodeps, or nosave: only named additions can
            # disappear. --unneeded preserves packages now required elsewhere.
            try:
                result = subprocess.run(['sudo', 'pacman', '-Ru', *candidates])
            finally:
                remaining = self.installed()
                removed = set(candidates) - remaining
                entry['added'] = sorted(set(entry['added']) - removed)
                entry['roots'] = sorted(set(entry['roots']) - removed)
                entry['removing'] = False
                self.manager.save_json(self.path, records)
            if result.returncode:
                raise Error('package removal failed or cancelled; configuration is removed; retry --remove-packages later')
            return sorted(removed)
