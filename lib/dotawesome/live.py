"""Conservative appearance-only admission checks for live Niri deployments."""
import base64
import json
from pathlib import Path
import re
import tomllib

from .manager import Error, encode


# Everything not explicitly listed remains part of the behaviour comparison.
SHELL_STYLE = {
    'theme.source', 'theme.builtin', 'theme.custom_palette',
    'shell.corner_radius_scale', 'shell.font_family', 'shell.popup_shadows',
    'shell.settings_window_translucent', 'shell.animation.enabled', 'shell.animation.speed',
    'shell.shadow.alpha', 'shell.shadow.direction',
    'shell.panel.transparency_mode', 'shell.panel.borders', 'shell.panel.shadow',
    'shell.panel.control_center_placement', 'shell.panel.control_center_position',
    'shell.panel.launcher_placement', 'shell.panel.launcher_position',
    'shell.panel.clipboard_placement', 'shell.panel.clipboard_position',
    'shell.panel.floating_offset', 'notification.background_opacity',
}
BAR_STYLE = {
    'thickness', 'margin_ends', 'margin_edge', 'radius', 'radius_top_left',
    'radius_top_right', 'radius_bottom_left', 'radius_bottom_right',
    'background_opacity', 'compositor_blur', 'border', 'border_width', 'shadow',
    'font_family', 'font_scale', 'font_weight', 'padding', 'widget_spacing',
}
COMPILED_PROFILE_FIELDS = {
    'name', 'terminal', 'render_device', 'secret_portal', 'vivaldi_password_store',
    'signal_password_store', 'wallpaper', 'laptop', 'external',
}


def profile_behaviour(data):
    # work.py consumes apps. These other fields are compiled into the KDL,
    # shell config, portals and flags, which are independently checked below.
    return {k: v for k, v in json.loads(data).items() if k not in COMPILED_PROFILE_FIELDS}


def shell_behaviour(data):
    def visit(value, prefix=''):
        result = {}
        for key, child in value.items():
            path = prefix + key
            if path in SHELL_STYLE or (path.startswith('bar.default.') and path[12:] in BAR_STYLE):
                continue
            if isinstance(child, dict):
                result.update(visit(child, path + '.'))
            else:
                result[path] = child
        return result
    return visit(tomllib.loads(data.decode()))


def niri_behaviour(data):
    # Keep quoted arguments intact: regex-removing raw blocks could hide a change
    # to a spawn command containing text that looks like a visual KDL node.
    tokens = re.findall(r'"(?:\\.|[^"\\])*"|//[^\n]*|/\*.*?\*/|[{};\n]|[^\s{};"]+', data.decode(), re.S)
    tokens = iter(t for t in tokens if not t.startswith(('//', '/*')))

    def nodes(nested=False):
        result, header = [], []
        for token in tokens:
            if token == '}':
                if not nested:
                    raise Error('live update: unsupported Niri syntax')
                if header:
                    result.append((tuple(header), None))
                return result
            if token == '{':
                if not header:
                    raise Error('live update: unsupported Niri syntax')
                result.append((tuple(header), nodes(True)))
                header = []
            elif token in (';', '\n'):
                if header:
                    result.append((tuple(header), None))
                    header = []
            else:
                header.append(token)
        if nested:
            raise Error('live update: unsupported Niri syntax')
        if header:
            result.append((tuple(header), None))
        return result

    result = []
    for header, children in nodes():
        name = header[0]
        if name == 'animations':
            continue
        allowed = {'layout': {'gaps', 'focus-ring', 'border', 'shadow'},
                   'window-rule': {'geometry-corner-radius', 'clip-to-geometry'}}.get(name, set())
        if name == 'layer-rule' and children and any(
                h == ('match', 'namespace=', '"^noctalia-"') for h, _ in children):
            allowed = {'background-effect'}
        if children is not None:
            children = [(h, c) for h, c in children if h[0] not in allowed]
            if not children and name == 'window-rule':
                continue
            if name == 'layer-rule' and allowed and all(h[0] == 'match' for h, _ in children):
                continue
        result.append((header, children))
    return result


def validate_live_update(manager, files):
    if (manager.state / 'transaction.json').exists():
        raise Error('live update refused: interrupted deployment requires recovery outside Niri')
    record = manager.records().get('niri')
    if not record:
        raise Error('live update requires an existing deployment; install outside Niri first')
    shell_root = manager.config / 'dotawesome/niri/noctalia/noctalia'
    shell = tomllib.loads(files[shell_root / 'config.toml'][0].decode())
    if shell['theme']['source'] not in ('builtin', 'custom'):
        raise Error('live update supports only fixed builtin or custom palettes')
    installed = record['files']
    for path in sorted(set(files) | {Path(p) for p in installed}):
        old = installed.get(str(path))
        current = manager.inspect(path)
        if old and current != old['installed']:
            raise Error(f'modified managed file: {path}; preserve and reconcile the edit before live update')
        candidate = encode(*files[path]) if path in files else None
        if current == candidate:
            continue
        if path.parent == shell_root / 'palettes' and path.suffix == '.json' and path in files:
            if not old and current is not None:
                raise Error(f'live update refuses to overwrite an unmanaged palette: {path}')
            json.loads(files[path][0])
            continue
        safe = False
        if current and candidate and 'data' in current and current['mode'] == candidate['mode']:
            before = base64.b64decode(current['data'])
            after = files[path][0]
            if path == manager.config / 'niri/config.kdl':
                safe = niri_behaviour(before) == niri_behaviour(after)
            elif path == shell_root / 'config.toml':
                safe = shell_behaviour(before) == shell_behaviour(after)
            elif path == manager.config / 'dotawesome/niri/profile.json':
                safe = profile_behaviour(before) == profile_behaviour(after)
        if not safe:
            raise Error(f'live update changes non-appearance configuration: {path}; apply outside Niri')
