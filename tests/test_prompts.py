"""Keyboard selection behavior, independent of terminal rendering."""
import sys
from pathlib import Path
import unittest
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'lib'))
from dotawesome import prompts


class SelectionTests(unittest.TestCase):
    def select(self, keys, multiple=False, defaults=()):
        keys = iter(keys)
        return prompts.select('Components', ['niri', 'tmux', 'shared'], defaults,
                              multiple=multiple, read_key=lambda: next(keys), draw=lambda *args: None)

    def test_arrows_move_and_enter_selects(self):
        self.assertEqual(self.select(['DOWN', 'ENTER']), ['tmux'])

    def test_space_toggles_multiple_choices(self):
        self.assertEqual(self.select(['SPACE', 'DOWN', 'SPACE', 'ENTER'], True), ['niri', 'tmux'])

    def test_saved_selection_can_be_deselected(self):
        self.assertEqual(self.select(['SPACE', 'DOWN', 'SPACE', 'ENTER'], True, ['niri']), ['tmux'])

    def test_empty_multiple_selection_waits_for_choice(self):
        self.assertEqual(self.select(['ENTER', 'SPACE', 'ENTER'], True), ['niri'])

    def test_escape_cancels(self):
        with self.assertRaises(KeyboardInterrupt):
            self.select(['ESC'])
