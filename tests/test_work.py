import importlib.util
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('work', ROOT / 'components/niri/work.py')
work = importlib.util.module_from_spec(spec); spec.loader.exec_module(work)


class FakeIPC:
    def __init__(self, windows=(), fail=()):
        self.windows = list(windows); self.fail = fail; self.actions = []; self.launched = []
    def query(self, kind):
        if kind == 'windows': return list(self.windows)
        return [{'id': 1, 'name': 'Code'}, {'id': 2, 'name': 'Web'}, {'id': 3, 'name': 'Music'}]
    def action(self, *args): self.actions.append(args)
    def launch(self, command):
        self.launched.append(command)
        if command[0] not in self.fail:
            self.windows.append({'id': len(self.windows)+100, 'app_id': command[0], 'workspace_id': None})


class WorkTests(unittest.TestCase):
    apps = [{'name': 'Editor', 'command': ['editor'], 'app_ids': ['editor'], 'workspace': 'Code', 'width': '50%'},
            {'name': 'Browser', 'command': ['browser'], 'app_ids': ['browser'], 'workspace': 'Web', 'width': '100%'}]
    def test_empty_launches_and_sizes_new_windows(self):
        ipc = FakeIPC(); self.assertEqual(work.arrange(ipc, self.apps, timeout=0), [])
        self.assertEqual(len(ipc.launched), 2)
        self.assertEqual(sum(a[0] == 'set-column-width' for a in ipc.actions), 2)
        self.assertEqual(ipc.actions[-1], ('focus-window', '--id', '100'))
    def test_partial_reuses_and_preserves_size(self):
        ipc = FakeIPC([{'id': 5, 'app_id': 'editor', 'workspace_id': 1}])
        work.arrange(ipc, self.apps, timeout=0)
        self.assertEqual(ipc.launched, [['browser']])
        self.assertEqual(sum(a[0] == 'set-column-width' for a in ipc.actions), 1)
    def test_multiple_prefers_target_workspace_and_leaves_others(self):
        ipc = FakeIPC([{'id': 4, 'app_id': 'editor', 'workspace_id': 2}, {'id': 5, 'app_id': 'editor', 'workspace_id': 1}])
        work.arrange(ipc, self.apps[:1], timeout=0)
        self.assertIn(('focus-window', '--id', '5'), ipc.actions)
        self.assertNotIn(('focus-window', '--id', '4'), ipc.actions)
        self.assertEqual(ipc.launched, [])
    def test_failed_launch_does_not_prevent_remaining_apps(self):
        ipc = FakeIPC(fail=['editor'])
        failures = work.arrange(ipc, self.apps, timeout=0)
        self.assertEqual(len(failures), 1); self.assertIn('Editor', failures[0])
        self.assertEqual(len(ipc.launched), 2)
        self.assertEqual(ipc.actions[-1], ('focus-workspace', 'Code'))
    def test_launch_oserror_is_reported_and_continue(self):
        ipc = FakeIPC(); original = ipc.launch
        def launch(command):
            if command == ['editor']: raise FileNotFoundError('missing')
            original(command)
        ipc.launch = launch
        self.assertEqual(len(work.arrange(ipc, self.apps, timeout=0)), 1)
        self.assertEqual(ipc.launched, [['browser']])


if __name__ == '__main__': unittest.main()

class FocusRegressionTests(unittest.TestCase):
    def test_size_targets_the_moved_window(self):
        class FocusIPC(FakeIPC):
            focused = None
            def action(self, *args):
                super().action(*args)
                if args[0] == 'focus-window': self.focused = args[-1]
                if args[0] == 'move-window-to-workspace' and 'false' in args: self.focused = 'unrelated'
                if args[0] == 'set-column-width':
                    if self.focused != '100': raise AssertionError('resized unrelated column')
        work.arrange(FocusIPC(), WorkTests.apps[:1], timeout=0)

class RuntimeTests(unittest.TestCase):
    def test_serialized_invocation_does_not_launch_duplicates(self):
        import fcntl
        import os
        import tempfile
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as tmp:
            with (Path(tmp) / 'dotawesome-niri-work.lock').open('a') as lock:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                with patch.dict(os.environ, {'XDG_RUNTIME_DIR': tmp}), patch.object(work, 'IPC') as ipc:
                    self.assertEqual(work.main(), 0)
                    ipc.assert_not_called()

    def test_startup_wait_is_bounded(self):
        from unittest.mock import patch
        ipc = FakeIPC(fail=['editor'])
        with patch.object(work.time, 'monotonic', side_effect=[0, .1, 1]), patch.object(work.time, 'sleep') as sleep:
            failures = work.arrange(ipc, WorkTests.apps[:1], timeout=.5)
        self.assertEqual(len(failures), 1)
        sleep.assert_called_once_with(.15)
