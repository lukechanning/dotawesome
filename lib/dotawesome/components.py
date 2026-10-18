"""The supported components, explicitly registered without discovery."""
from dataclasses import dataclass

from . import applications, niri
from .manager import Error


@dataclass(frozen=True)
class Component:
    name: str
    description: str

    def profile(self, repo, value):
        if self.name == 'niri':
            if not value:
                raise Error('niri requires --profile generic, g14, desktop, or a TOML path')
            return niri.profile(repo, value)
        if value:
            raise Error(f'{self.name} does not use --profile')
        return None

    def render(self, manager, profile):
        return niri.render(manager, profile) if self.name == 'niri' else applications.render(self.name, manager)

    def check(self, manager, profile):
        return niri.check(manager, profile) if self.name == 'niri' else applications.check(self.name, manager)

    def validate(self, manager, files):
        return niri.validate(manager, files) if self.name == 'niri' else applications.validate(self.name, manager, files)

    @property
    def guidance(self):
        return {
            'niri': 'Select the packaged Niri session at login. Visual/session acceptance remains manual.',
            'nvim': 'Open nvim to install pinned plugins. Use :Mason and :TSInstall for optional tools and parsers.',
            'tmux': 'New sessions use this config. To reload an existing session: tmux source-file ~/.tmux.conf',
        }[self.name]


COMPONENTS = {c.name: c for c in (
    Component('niri', 'Niri + Noctalia; profiles: generic, g14, desktop'),
    Component('nvim', 'Neovim editor'),
    Component('tmux', 'tmux terminal multiplexer'),
)}
