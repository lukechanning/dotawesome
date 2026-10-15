#!/usr/bin/env python3
"""Bounded, serialized Niri work layout. Uses exact, profile-owned app IDs."""
import fcntl
import json
import os
from pathlib import Path
import subprocess
import sys
import time


class IPC:
    def call(self, *args):
        result = subprocess.run(['niri', 'msg', *args], capture_output=True, text=True, timeout=5)
        if result.returncode:
            raise RuntimeError(result.stderr.strip() or result.stdout.strip())
        return result.stdout

    def query(self, kind):
        return json.loads(self.call('--json', kind))

    def action(self, *args):
        self.call('action', *args)

    def launch(self, command):
        # Niri owns child lifecycle, including cleanup at logout.
        self.action('spawn', '--', *command)


def arrange(ipc, apps, timeout=12):
    workspaces = {w['name']: w['id'] for w in ipc.query('workspaces') if w.get('name')}
    failures = []
    code_window = None
    for app in apps:
        if not app['command']:
            continue
        try:
            if app['workspace'] not in workspaces:
                raise RuntimeError(f"workspace {app['workspace']} is missing")
            def matching():
                return sorted((w for w in ipc.query('windows') if w.get('app_id') in app['app_ids']),
                              key=lambda w: (w.get('workspace_id') != workspaces[app['workspace']], w['id']))
            matches = matching()
            new = not matches
            if new:
                ipc.launch(app['command'])
                deadline = time.monotonic() + timeout
                while True:
                    matches = matching()
                    if matches or time.monotonic() >= deadline:
                        break
                    time.sleep(.15)
            if not matches:
                raise RuntimeError('no matching window before startup timeout; inspect niri msg --json windows')
            window = matches[0]
            ipc.action('focus-window', '--id', str(window['id']))
            ipc.action('move-window-to-workspace', '--window-id', str(window['id']), '--focus', 'false', app['workspace'])
            if new:
                ipc.action('focus-window', '--id', str(window['id']))
                ipc.action('set-column-width', app['width'])
            if app['workspace'] == 'Code' and code_window is None:
                code_window = window['id']
        except (OSError, RuntimeError, subprocess.SubprocessError) as exc:
            failures.append(f"{app['name']}: {exc}")
    ipc.action('focus-workspace', 'Code')
    if code_window is not None:
        ipc.action('focus-window', '--id', str(code_window))
    return failures


def main():
    runtime = os.environ.get('XDG_RUNTIME_DIR')
    if not runtime:
        raise RuntimeError('XDG_RUNTIME_DIR is required inside the Niri session')
    with (Path(runtime) / 'dotawesome-niri-work.lock').open('a') as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            return 0  # An invocation already owns the complete layout operation.
        profile = json.loads(Path(__file__).with_name('profile.json').read_text())
        failures = arrange(IPC(), profile['apps'])
        if failures:
            message = '\n'.join(failures)
            print(message, file=sys.stderr)
            subprocess.run(['noctalia', 'msg', 'notification-show', 'Work layout: launch failures', message], timeout=5, check=False)
            return 1
    return 0


if __name__ == '__main__':
    try:
        sys.exit(main())
    except (OSError, RuntimeError, subprocess.SubprocessError) as exc:
        print(f'work layout: {exc}', file=sys.stderr)
        sys.exit(1)
