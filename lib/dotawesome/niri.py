"""Niri rendering and session policy; shared deployment stays in manager.py."""
import json
import os
from pathlib import Path
import re
import shutil
import tempfile
import tomllib

from .manager import Error


def definition(repo):
    return json.loads((repo / 'components/niri/definition.json').read_text())


def profile(repo, name):
    path = Path(name)
    if name in ('generic', 'g14'):
        path = repo / 'components/niri/profiles' / f'{name}.toml'
    try:
        return tomllib.loads(path.read_text())
    except (OSError, ValueError) as exc:
        raise Error(f'cannot load profile {name}: {exc}') from exc


def argv(value):
    if not isinstance(value, list) or not all(isinstance(s, str) and s and not any(ord(c) < 32 for c in s) for s in value):
        raise Error('commands must be argv arrays of nonempty strings')
    return ' '.join(json.dumps(s, ensure_ascii=False) for s in value)


def validate_profile(p):
    if p.get('render_device') is not None and (not isinstance(p['render_device'], str) or
            not re.fullmatch(r'/dev/dri/(?:renderD\d+|by-path/pci-[\w:.-]+-render)', p['render_device'])):
        raise Error('render_device must be a DRM render node path')
    if p.get('vivaldi_password_store') not in (None, 'kwallet6'):
        raise Error('vivaldi_password_store must be kwallet6')
    if p.get('secret_portal') not in (None, 'kwallet', 'gnome-keyring'):
        raise Error('secret_portal must be kwallet or gnome-keyring')
    if not argv(p.get('terminal')):
        raise Error('terminal command is required')
    for app in p.get('apps', []):
        argv(app.get('command'))
        if app.get('workspace') not in ('Web', 'Code', 'Media') or app.get('width') not in ('33%', '50%', '67%', '100%'):
            raise Error('invalid work app workspace or width')
        if not app.get('app_ids') or not all(isinstance(s, str) and s for s in app['app_ids']):
            raise Error('each app needs exact app_ids')
        if not isinstance(app.get('name'), str):
            raise Error('each app needs a name')
    for key in ('laptop', 'external'):
        output = p.get(key)
        if not output:
            continue
        if not isinstance(output.get('identity'), str) or not output['identity'].strip():
            raise Error(f'{key}: display identity required')
        if not re.fullmatch(r'\d+x\d+(?:@\d+(?:\.\d+)?)?', output.get('mode', '')):
            raise Error(f'{key}: invalid mode')
        scale = output.get('scale')
        if isinstance(scale, bool) or not isinstance(scale, (int, float)) or not .5 <= scale <= 4:
            raise Error(f'{key}: scale must be between 0.5 and 4')
        if key == 'external':
            if re.fullmatch(r'(?:HDMI|DP|eDP|DVI)[-\w]*', output['identity']) or len(output['identity'].split()) < 3:
                raise Error('external requires manufacturer/model/serial identity, not a connector')
            if set(output.get('verified_connections', [])) != {'dock-hdmi', 'direct-hdmi'}:
                raise Error('external identity must be verified on dock-hdmi and direct-hdmi')
            if output['mode'] not in output.get('advertised_modes', []):
                raise Error('external mode must be captured in advertised_modes')
            if not output['mode'].startswith('1920x1080'):
                raise Error('work monitor must use native 1920x1080')


def wallpaper_paths(manager, p):
    wallpaper = p.get('wallpaper', {})
    paths = [wallpaper.get('default'), *wallpaper.get('favorites', [])]
    resolved = []
    for path in paths:
        if path is None:
            continue
        if not isinstance(path, str) or not path or any(ord(c) < 32 for c in path):
            raise Error('wallpaper paths must be nonempty file paths')
        expanded = manager.home / path[2:] if path.startswith('~/') else Path(path)
        if not expanded.is_absolute():
            raise Error('wallpaper paths must be absolute or start with ~/')
        resolved.append(expanded)
    return resolved


def render(manager, p):
    validate_profile(p)
    source = manager.repo / 'components/niri'
    config = (source / 'base.kdl').read_text()
    if p.get('render_device'):
        config += f'\ndebug {{\n    render-drm-device {json.dumps(p["render_device"])}\n}}\n'
    external = p.get('external')
    for key in ('laptop', 'external'):
        output = p.get(key)
        if not output:
            continue
        # Place external at zero and laptop to its right, using logical width.
        x = round(1920 / external['scale']) if key == 'laptop' and external else 0
        focus = '    focus-at-startup\n' if key == 'external' or not external else ''
        config += f'\noutput {json.dumps(output["identity"])} {{\n{focus}    mode {json.dumps(output["mode"])}\n    scale {output["scale"]}\n    position x={x} y=0\n}}\n'
    for workspace in ('Web', 'Code', 'Media'):
        prefer = external or p.get('laptop')
        rule = f' open-on-output {json.dumps(prefer["identity"])}; ' if prefer else ''
        config += f'workspace "{workspace}" {{{rule}}}\n'
    helper = manager.config / 'dotawesome/niri/work.py'
    bindings = f'    Mod+Return {{ spawn {argv(p["terminal"])}; }}\n'
    bindings += f'    Mod+Shift+Return {{ spawn "python3" {json.dumps(str(helper))}; }}\n'
    ipc = {
        'Mod+Space': ['panel-toggle', 'launcher'], 'Mod+V': ['panel-toggle', 'clipboard'],
        'Mod+S': ['panel-toggle', 'control-center'], 'Mod+Alt+L': ['lock'],
        'Mod+N': ['panel-toggle', 'control-center', 'notifications'],
        'Mod+B': ['panel-toggle', 'control-center', 'network'],
        'Mod+Shift+B': ['panel-toggle', 'control-center', 'bluetooth'],
        'XF86AudioRaiseVolume': ['volume-up'], 'XF86AudioLowerVolume': ['volume-down'],
        'XF86AudioMute': ['volume-mute'], 'XF86AudioMicMute': ['mic-mute'],
        'XF86AudioPlay': ['media', 'toggle'], 'XF86AudioNext': ['media', 'next'],
        'XF86AudioPrev': ['media', 'previous'],
        'XF86MonBrightnessUp': ['brightness-up'], 'XF86MonBrightnessDown': ['brightness-down'],
    }
    for binding, args in ipc.items():
        locked = ' allow-when-locked=true' if binding.startswith('XF86') else ''
        bindings += f'    {binding}{locked} {{ spawn "noctalia" "msg" {argv(args)}; }}\n'
    config = config.replace('// @DOTAWESOME_COMMAND_BINDS@', bindings.rstrip())
    shell_text = (source / 'noctalia.toml').read_text()
    paths = iter(wallpaper_paths(manager, p))
    if p.get('wallpaper', {}).get('default') is not None:
        shell_text += '\n[wallpaper.default]\npath = ' + json.dumps(str(next(paths)), ensure_ascii=False) + '\n'
    for path in paths:
        shell_text += '\n[[wallpaper.favorite]]\npath = ' + json.dumps(str(path), ensure_ascii=False) + '\n'
    shell = shell_text.encode()
    tomllib.loads(shell_text)
    # Unit expansion has its own quoting and % specifier rules.
    def unit_path(path):
        return str(path).replace('\\', '\\\\').replace('"', '\\"').replace('%', '%%')
    unit = (source / 'noctalia.service').read_text().replace('@CONFIG@', unit_path(manager.config)).replace('@STATE@', unit_path(manager.state))
    portals = '[preferred]\ndefault=gnome;gtk;\norg.freedesktop.impl.portal.FileChooser=gtk\n'
    if p.get('secret_portal'):
        portals += f'org.freedesktop.impl.portal.Secret={p["secret_portal"]}\n'
    files = {
        manager.config / 'niri/config.kdl': (config.encode(), 0o644),
        helper: ((source / 'work.py').read_bytes(), 0o644),
        helper.with_name('profile.json'): ((json.dumps(p, indent=2) + '\n').encode(), 0o644),
        manager.config / 'dotawesome/niri/noctalia/noctalia/config.toml': (shell, 0o644),
        manager.config / 'systemd/user/dotawesome-niri-noctalia.service': (unit.encode(), 0o644),
        manager.config / 'systemd/user/niri.service.d/50-dotawesome.conf': (b'[Unit]\nWants=dotawesome-niri-noctalia.service\n', 0o644),
        manager.config / 'xdg-desktop-portal/niri-portals.conf': (portals.encode(), 0o644),
    }
    expected = {manager.config / d for d in definition(manager.repo)['destinations']}
    if p.get('vivaldi_password_store'):
        flags = manager.config / 'vivaldi-stable.conf'
        files[flags] = (b'--password-store=kwallet6\n', 0o644)
        expected.update(manager.config / d for d in definition(manager.repo)['optional_destinations'])
    if set(files) != expected:
        raise Error('component destinations do not match definition')
    return files


def startup_conflicts(manager, system_dirs=None):
    """Honor desktop-entry precedence and desktop scope; don't change any owners."""
    import configparser
    system_dirs = system_dirs if system_dirs is not None else [Path(p) for p in os.environ.get('XDG_CONFIG_DIRS', '/etc/xdg').split(':') if p]
    entries = {}
    for directory in [manager.config, *system_dirs]:
        for path in (directory / 'autostart').glob('*.desktop'):
            entries.setdefault(path.name, path)
    conflicts = []
    pattern = r'\b(?:noctalia(?:-shell)?|mako|swayidle|swaylock|waybar|dunst|swaync|polkit-kde-authentication-agent(?:-1)?)\b'
    for path in entries.values():
        ini = configparser.ConfigParser(interpolation=None, strict=False)
        try:
            ini.read(path)
            entry = ini['Desktop Entry']
            if entry.get('Hidden', '').lower() == 'true':
                continue
            only = entry.get('OnlyShowIn', '').lower().split(';')
            exclude = entry.get('NotShowIn', '').lower().split(';')
            if (entry.get('OnlyShowIn') and 'niri' not in only) or 'niri' in exclude:
                continue
            if re.search(pattern, entry.get('Exec', '')):
                conflicts.append(f'competing Niri autostart: {path}; scope or disable it before activation')
        except (OSError, configparser.Error, KeyError):
            continue
    # Only inspect enabled unit links and Niri-owned dependencies, not dormant unit definitions.
    units = manager.config / 'systemd/user'
    for path in units.glob('**/*'):
        if not path.is_file() or not any(part.endswith(('.wants', '.requires')) for part in path.parts):
            continue
        if path.name == 'dotawesome-niri-noctalia.service':
            continue
        try:
            if re.search(r'ExecStart\s*=.*' + pattern, path.read_text()):
                conflicts.append(f'competing enabled user unit: {path}; inspect its session scope')
        except (OSError, UnicodeError):
            continue
    return conflicts


def check(manager, p):
    validate_profile(p)
    info = definition(manager.repo)
    report = {'errors': [], 'warnings': [], 'versions': {}, 'packages': info['packages']}
    if not Path('/usr/share/wayland-sessions/niri.desktop').is_file() or not Path('/usr/lib/systemd/user/niri.service').is_file():
        report['errors'].append('missing packaged Niri session entry or niri.service')
    if not Path('/etc/pam.d/login').is_file():
        report['errors'].append('missing PAM login service used by Noctalia locking')
    for cmd in info['commands']:
        if not shutil.which(cmd):
            report['errors'].append(f'missing command: {cmd}')
    for cmd, minimum in info['minimum_versions'].items():
        if shutil.which(cmd):
            try:
                version = manager.runner([cmd, '--version'])
                report['versions'][cmd] = version
                match = re.search(r'(\d+)\.(\d+)', version or '')
                if not match or tuple(map(int, match.groups())) < tuple(map(int, minimum.split('.'))):
                    report['errors'].append(f'{cmd}: requires >= {minimum}, got {version}')
            except Error as exc:
                report['errors'].append(str(exc))
    if not shutil.which('pacman'):
        report['errors'].append('supported package platform requires pacman (Arch/CachyOS)')
    else:
        for package in info['packages']:
            try:
                manager.runner(['pacman', '-Q', package])
            except Error:
                report['errors'].append(f'missing package: {package}')
    for path in dict.fromkeys(wallpaper_paths(manager, p)):
        if not path.is_file():
            report['warnings'].append(f'missing local wallpaper: {path}; dark fallback will be used')
    for app in p['apps']:
        if app['command'] and not shutil.which(app['command'][0]):
            report['warnings'].append(f"optional {app['name']}: missing {app['command'][0]}")
    report['errors'].extend(startup_conflicts(manager))
    for path in (manager.config / 'systemd/user/noctalia.service', manager.config / 'noctalia/config.toml'):
        if path.exists():
            report['warnings'].append(f'existing shared Noctalia configuration/service: {path}; isolated Niri profile used')
    overrides = manager.state / 'niri-shell/noctalia/settings.toml'
    if overrides.exists():
        report['warnings'].append(f'GUI overrides take precedence: {overrides}')
    if not p.get('external'):
        report['warnings'].append('external monitor identity pending; unidentified outputs use automatic configuration')
    return report


def validate(manager, files):
    for command in ('niri', 'noctalia', 'systemd-analyze'):
        if not shutil.which(command):
            raise Error(f'native validation pending: missing {command}; activation refused')
    with tempfile.TemporaryDirectory(prefix='dotawesome-validation-') as tmp:
        work = Path(tmp)
        config = work / 'config.kdl'; config.write_bytes(files[manager.config / 'niri/config.kdl'][0])
        shell = work / 'config.toml'; shell.write_bytes(files[manager.config / 'dotawesome/niri/noctalia/noctalia/config.toml'][0])
        unit = work / 'dotawesome-niri-noctalia.service'
        unit.write_bytes(files[manager.config / 'systemd/user/dotawesome-niri-noctalia.service'][0])
        for command in (['niri', 'validate', '--config', str(config)],
                        ['noctalia', 'config', 'validate', str(shell)],
                        ['systemd-analyze', '--user', 'verify', str(unit)]):
            output = manager.runner(command) or ''
            if re.search(r'\b(?:warn(?:ing)?|unknown|obsolete|error)\b', output, re.I):
                raise Error(f'native validation warning/error: {output}')
    return True


def outside_session(manager):
    if os.environ.get('NIRI_SOCKET') or 'niri' in os.environ.get('XDG_CURRENT_DESKTOP', '').lower().split(':'):
        raise Error('log out of Niri before changing this deployment')
    try:
        active = manager.runner(['systemctl', '--user', 'is-active', 'niri.service'])
    except Error:
        active = ''
    if active == 'active':
        raise Error('niri.service is active; log out of Niri first')
