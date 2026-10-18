"""Small keyboard menus using the standard library terminal support."""
import curses
import sys


def available():
    return sys.stdin.isatty() and sys.stdout.isatty()


def select(title, options, defaults=(), *, multiple=False, read_key, draw):
    """Select values; rendering and key input are supplied by the terminal adapter."""
    selected = set(defaults)
    cursor = next((i for i, item in enumerate(options) if item in selected), 0)
    while True:
        draw(title, options, cursor, selected, multiple)
        key = read_key()
        if key in ('ESC', 'QUIT'):
            raise KeyboardInterrupt()
        if key == 'DOWN':
            cursor = (cursor + 1) % len(options)
        elif key == 'UP':
            cursor = (cursor - 1) % len(options)
        elif key == 'SPACE' and multiple:
            item = options[cursor]
            selected.symmetric_difference_update([item])
        elif key == 'ENTER':
            if not multiple:
                return [options[cursor]]
            if selected:
                return [item for item in options if item in selected]


def menu(title, options, defaults=(), *, multiple=False):
    def run(screen):
        screen.keypad(True)
        try:
            curses.curs_set(0)
        except curses.error:
            pass

        def read_key():
            key = screen.get_wch()
            return {curses.KEY_DOWN: 'DOWN', curses.KEY_UP: 'UP',
                    'j': 'DOWN', 'k': 'UP', ' ': 'SPACE', '\n': 'ENTER',
                    '\r': 'ENTER', curses.KEY_ENTER: 'ENTER', '\x1b': 'ESC',
                    '\x03': 'QUIT'}.get(key, '')

        def draw(title, options, cursor, selected, multiple):
            screen.erase()
            height, width = screen.getmaxyx()
            lines = [title, '↑/↓ move · ' + ('Space toggle · ' if multiple else '') + 'Enter accept · Esc cancel', '']
            first = max(0, cursor - max(1, height - 5) + 1)
            rows = [(line, False) for line in lines]
            for index in range(first, len(options)):
                marker = ('[x] ' if options[index] in selected else '[ ] ') if multiple else ''
                rows.append(('  ' + marker + options[index], index == cursor))
            for y, (line, focused) in enumerate(rows[:max(0, height - 1)]):
                screen.addnstr(y, 0, line, max(0, width - 1), curses.A_REVERSE if focused else curses.A_NORMAL)
            screen.refresh()

        return select(title, options, defaults, multiple=multiple, read_key=read_key, draw=draw)

    return curses.wrapper(run)
