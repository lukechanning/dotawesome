package app

import (
	"dotawesome/internal/deploy"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

type kdlNode struct {
	Header   []string
	Children []kdlNode
}

func niriBehaviour(data []byte) ([]kdlNode, error) {
	re := regexp.MustCompile(`"(?:\\.|[^"\\])*"|//[^\n]*|/\*[\s\S]*?\*/|[{};\n]|[^\s{};"]+`)
	tokens := re.FindAllString(string(data), -1)
	i := 0
	var parse func(bool) ([]kdlNode, error)
	parse = func(nested bool) ([]kdlNode, error) {
		nodes := []kdlNode{}
		header := []string{}
		flush := func() {
			if len(header) > 0 {
				nodes = append(nodes, kdlNode{Header: header})
				header = []string{}
			}
		}
		for i < len(tokens) {
			t := tokens[i]
			i++
			switch {
			case strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*"):
				continue
			case t == "}":
				if !nested {
					return nil, errors.New("live update: unsupported Niri syntax")
				}
				flush()
				return nodes, nil
			case t == "{":
				if len(header) == 0 {
					return nil, errors.New("live update: unsupported Niri syntax")
				}
				children, e := parse(true)
				if e != nil {
					return nil, e
				}
				nodes = append(nodes, kdlNode{header, children})
				header = []string{}
			case t == ";" || t == "\n":
				flush()
			default:
				header = append(header, t)
			}
		}
		if nested {
			return nil, errors.New("live update: unsupported Niri syntax")
		}
		flush()
		return nodes, nil
	}
	nodes, e := parse(false)
	if e != nil {
		return nil, e
	}
	result := []kdlNode{}
	for _, n := range nodes {
		name := n.Header[0]
		if name == "animations" {
			continue
		}
		allowed := map[string]bool{}
		switch name {
		case "layout":
			for _, s := range []string{"gaps", "focus-ring", "border", "shadow"} {
				allowed[s] = true
			}
		case "window-rule":
			allowed["geometry-corner-radius"] = true
			allowed["clip-to-geometry"] = true
		case "layer-rule":
			for _, child := range n.Children {
				if reflect.DeepEqual(child.Header, []string{"match", "namespace=", "\"^noctalia-\""}) {
					allowed["background-effect"] = true
				}
			}
		}
		if n.Children != nil {
			children := []kdlNode{}
			for _, c := range n.Children {
				if !allowed[c.Header[0]] {
					children = append(children, c)
				}
			}
			n.Children = children
			if len(children) == 0 && name == "window-rule" {
				continue
			}
			if name == "layer-rule" && allowed["background-effect"] {
				onlyMatch := true
				for _, c := range children {
					if c.Header[0] != "match" {
						onlyMatch = false
					}
				}
				if onlyMatch {
					continue
				}
			}
		}
		result = append(result, n)
	}
	return result, nil
}

var shellStyle = strings.Fields("theme.source theme.builtin theme.custom_palette shell.corner_radius_scale shell.font_family shell.popup_shadows shell.settings_window_translucent shell.animation.enabled shell.animation.speed shell.shadow.alpha shell.shadow.direction shell.panel.transparency_mode shell.panel.borders shell.panel.shadow shell.panel.control_center_placement shell.panel.control_center_position shell.panel.launcher_placement shell.panel.launcher_position shell.panel.clipboard_placement shell.panel.clipboard_position shell.panel.floating_offset notification.background_opacity")
var barStyle = strings.Fields("thickness margin_ends margin_edge radius radius_top_left radius_top_right radius_bottom_left radius_bottom_right background_opacity compositor_blur border border_width shadow font_family font_scale font_weight padding widget_spacing")

func shellBehaviour(b []byte) (map[string]any, error) {
	var data map[string]any
	if e := toml.Unmarshal(b, &data); e != nil {
		return nil, e
	}
	allowed := set(shellStyle)
	for _, s := range barStyle {
		allowed["bar.default."+s] = true
	}
	result := map[string]any{}
	var walk func(map[string]any, string)
	walk = func(m map[string]any, prefix string) {
		for k, v := range m {
			path := prefix + k
			if allowed[path] {
				continue
			}
			if child, ok := v.(map[string]any); ok {
				walk(child, path+".")
			} else {
				result[path] = v
			}
		}
	}
	walk(data, "")
	return result, nil
}
func profileBehaviour(b []byte) (map[string]any, error) {
	var data map[string]any
	if e := json.Unmarshal(b, &data); e != nil {
		return nil, e
	}
	for _, k := range strings.Fields("name terminal preferred_output render_device secret_portal vivaldi_password_store signal_password_store wallpaper laptop external") {
		delete(data, k)
	}
	return data, nil
}
func (s *Service) ValidateLive(r *Inspection) error {
	m := s.Manager
	if deploy.Exists(filepath.Join(m.State, "transaction.json")) {
		return errors.New("live update refused: interrupted deployment requires recovery outside Niri")
	}
	records, e := m.Records()
	if e != nil {
		return e
	}
	record, ok := records["niri"]
	if !ok {
		return errors.New("live update requires an existing deployment; install outside Niri first")
	}
	root := filepath.Join(m.Config, "dotawesome/niri/noctalia/noctalia")
	if value := r.Plan.Files[filepath.Join(root, "config.toml")]; value != nil {
		var shell struct{ Theme struct{ Source string } }
		if e = toml.Unmarshal(value.Bytes(), &shell); e != nil {
			return e
		}
		if shell.Theme.Source != "builtin" && shell.Theme.Source != "custom" {
			return errors.New("live update supports only fixed builtin or custom palettes")
		}
	}
	for p, c := range r.Plan.Changes {
		old, owned := record.Files[p]
		if owned && !reflect.DeepEqual(old.Installed, c.Before) {
			return fmt.Errorf("modified managed file: %s", p)
		}
		safe := false
		if filepath.Dir(p) == filepath.Join(root, "palettes") && filepath.Ext(p) == ".json" && c.After != nil {
			if !owned && c.Before != nil {
				return fmt.Errorf("live update refuses unmanaged palette: %s", p)
			}
			if !json.Valid(c.After.Bytes()) {
				return errors.New("invalid palette JSON")
			}
			continue
		}
		if c.Before != nil && c.After != nil && c.Before.Mode != nil && c.After.Mode != nil && *c.Before.Mode == *c.After.Mode {
			before, after := c.Before.Bytes(), c.After.Bytes()
			switch p {
			case filepath.Join(m.Config, "niri/config.kdl"):
				a, e := niriBehaviour(before)
				if e != nil {
					return e
				}
				b, e := niriBehaviour(after)
				if e != nil {
					return e
				}
				safe = reflect.DeepEqual(a, b)
			case filepath.Join(root, "config.toml"):
				a, e := shellBehaviour(before)
				if e != nil {
					return e
				}
				b, e := shellBehaviour(after)
				if e != nil {
					return e
				}
				safe = reflect.DeepEqual(a, b)
			case filepath.Join(m.Config, "dotawesome/niri/profile.json"):
				a, e := profileBehaviour(before)
				if e != nil {
					return e
				}
				b, e := profileBehaviour(after)
				if e != nil {
					return e
				}
				safe = reflect.DeepEqual(a, b)
			}
		}
		if !safe {
			return fmt.Errorf("live update changes non-appearance configuration: %s; apply outside Niri", p)
		}
	}
	return nil
}
