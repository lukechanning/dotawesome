// Package components owns the explicit workstation component registry and renderers.
package components

import (
	"dotawesome/internal/deploy"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var Names = []string{"niri", "nvim", "tmux"}

type App struct {
	Name      string   `toml:"name" json:"name"`
	JumpKey   string   `toml:"jump_key" json:"jump_key,omitempty"`
	Command   []string `toml:"command" json:"command"`
	IDs       []string `toml:"app_ids" json:"app_ids"`
	Workspace string   `toml:"workspace" json:"workspace"`
	Width     string   `toml:"width" json:"width"`
}
type Output struct {
	Identity string   `toml:"identity" json:"identity"`
	Mode     string   `toml:"mode" json:"mode"`
	Scale    float64  `toml:"scale" json:"scale"`
	Verified []string `toml:"verified_connections" json:"verified_connections,omitempty"`
	Modes    []string `toml:"advertised_modes" json:"advertised_modes,omitempty"`
}
type Wallpaper struct {
	Default   string   `toml:"default" json:"default,omitempty"`
	Favorites []string `toml:"favorites" json:"favorites,omitempty"`
}
type Profile struct {
	Name         string    `toml:"name" json:"name"`
	Terminal     []string  `toml:"terminal" json:"terminal"`
	Preferred    string    `toml:"preferred_output" json:"preferred_output,omitempty"`
	RenderDevice string    `toml:"render_device" json:"render_device,omitempty"`
	SecretPortal string    `toml:"secret_portal" json:"secret_portal,omitempty"`
	Vivaldi      string    `toml:"vivaldi_password_store" json:"vivaldi_password_store,omitempty"`
	Signal       string    `toml:"signal_password_store" json:"signal_password_store,omitempty"`
	Wallpaper    Wallpaper `toml:"wallpaper" json:"wallpaper,omitempty"`
	Laptop       *Output   `toml:"laptop" json:"laptop,omitempty"`
	External     *Output   `toml:"external" json:"external,omitempty"`
	Apps         []App     `toml:"apps" json:"apps"`
}

func quote(s string) string { b, _ := json.Marshal(s); return string(b) }
func validText(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < 32 {
			return false
		}
	}
	return true
}
func argv(v []string) (string, error) {
	a := []string{}
	for _, s := range v {
		if !validText(s) {
			return "", errors.New("commands must be argv arrays of nonempty strings")
		}
		a = append(a, quote(s))
	}
	return strings.Join(a, " "), nil
}
func (p Profile) Validate() error {
	if len(p.Terminal) == 0 {
		return errors.New("terminal command is required")
	}
	if _, e := argv(p.Terminal); e != nil {
		return e
	}
	if p.RenderDevice != "" && !regexp.MustCompile(`^/dev/dri/(renderD[0-9]+|by-path/pci-[\w:.-]+-render)$`).MatchString(p.RenderDevice) {
		return errors.New("render_device must be a DRM render node path")
	}
	for _, s := range []string{p.Vivaldi, p.Signal} {
		if s != "" && s != "kwallet6" {
			return errors.New("password_store must be kwallet6")
		}
	}
	if p.SecretPortal != "" && p.SecretPortal != "kwallet" && p.SecretPortal != "gnome-keyring" {
		return errors.New("secret_portal must be kwallet or gnome-keyring")
	}
	jumpKeys := map[string]bool{}
	for _, a := range p.Apps {
		if a.JumpKey != "" {
			if (a.JumpKey != "b" && a.JumpKey != "t") || jumpKeys[a.JumpKey] {
				return errors.New("jump_key must be b or t, with no duplicates")
			}
			jumpKeys[a.JumpKey] = true
		}
		if _, e := argv(a.Command); e != nil {
			return e
		}
		if (a.Workspace != "One" && a.Workspace != "Two") || !strings.Contains("|33%|50%|67%|100%|", "|"+a.Width+"|") || len(a.IDs) == 0 || a.Name == "" {
			return errors.New("invalid work app workspace, width, name or exact app_ids")
		}
		for _, s := range a.IDs {
			if !validText(s) {
				return errors.New("invalid app ID")
			}
		}
	}
	for i, o := range []*Output{p.Laptop, p.External} {
		if o == nil {
			continue
		}
		if !validText(o.Identity) || !regexp.MustCompile(`^\d+x\d+(@\d+(\.\d+)?)?$`).MatchString(o.Mode) || o.Scale < .5 || o.Scale > 4 || math.IsNaN(o.Scale) || math.IsInf(o.Scale, 0) {
			return errors.New("invalid display identity, mode or scale")
		}
		if i == 1 {
			v := append([]string{}, o.Verified...)
			sort.Strings(v)
			if len(strings.Fields(o.Identity)) < 3 || strings.Join(v, ",") != "direct-hdmi,dock-hdmi" || !contains(o.Modes, o.Mode) || !strings.HasPrefix(o.Mode, "1920x1080") {
				return errors.New("external requires manufacturer/model/serial identity, verified dock/direct HDMI and advertised native 1920x1080 mode")
			}
		}
	}
	return nil
}
func contains(v []string, s string) bool {
	for _, x := range v {
		if x == s {
			return true
		}
	}
	return false
}
func LoadProfile(repo, name string) (Profile, error) {
	var p Profile
	if name == "" {
		return p, errors.New("niri requires --profile generic, g14, desktop, or a TOML path")
	}
	path := name
	if contains([]string{"generic", "g14", "desktop"}, name) {
		path = filepath.Join(repo, "components/niri/profiles", name+".toml")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return p, e
	}
	if e = toml.Unmarshal(b, &p); e != nil {
		return p, e
	}
	return p, p.Validate()
}
func Render(m *deploy.Manager, name, profile string, helper []byte) (deploy.Files, error) {
	files := deploy.Files{}
	add := func(target string, b []byte, mode uint32) { files[target] = deploy.File(b, mode) }
	source := filepath.Join(m.Repo, "components", name)
	read := func(path string) ([]byte, error) { return os.ReadFile(filepath.Join(source, path)) }
	if name != "niri" {
		if profile != "" {
			return nil, fmt.Errorf("%s does not use --profile", name)
		}
		if name == "tmux" {
			b, e := read("tmux.conf")
			if e != nil {
				return nil, e
			}
			add(filepath.Join(m.Home, ".tmux.conf"), b, 0644)
		} else if name == "nvim" {
			root := filepath.Join(source, "config")
			e := filepath.WalkDir(root, func(path string, d os.DirEntry, e error) error {
				if e != nil {
					return e
				}
				if d.IsDir() {
					return nil
				}
				if d.Type()&os.ModeSymlink != 0 {
					return fmt.Errorf("unexpected source symlink: %s", path)
				}
				rel, _ := filepath.Rel(root, path)
				b, e := os.ReadFile(path)
				if e == nil {
					add(filepath.Join(m.Config, "nvim", rel), b, 0644)
				}
				return e
			})
			if e != nil {
				return nil, e
			}
		} else {
			return nil, fmt.Errorf("unknown component: %s", name)
		}
		return files, nil
	}
	p, e := LoadProfile(m.Repo, profile)
	if e != nil {
		return nil, e
	}
	base, e := read("base.kdl")
	if e != nil {
		return nil, e
	}
	config := string(base)
	if p.RenderDevice != "" {
		config += "\ndebug {\n    render-drm-device " + quote(p.RenderDevice) + "\n}\n"
	}
	for i, o := range []*Output{p.Laptop, p.External} {
		if o == nil {
			continue
		}
		x := 0
		focus := ""
		if i == 0 && p.External != nil {
			x = int(math.RoundToEven(1920 / p.External.Scale))
		}
		if i == 1 || p.External == nil {
			focus = "    focus-at-startup\n"
		}
		config += fmt.Sprintf("\noutput %s {\n%s    mode %s\n    scale %s\n    position x=%d y=0\n}\n", quote(o.Identity), focus, quote(o.Mode), strconv.FormatFloat(o.Scale, 'f', -1, 64), x)
	}
	prefer := p.Preferred
	if p.Laptop != nil && prefer == "" {
		prefer = p.Laptop.Identity
	}
	if p.External != nil {
		prefer = p.External.Identity
	}
	for _, w := range []string{"One", "Two"} {
		rule := ""
		if prefer != "" {
			rule = " open-on-output " + quote(prefer) + "; "
		}
		config += "workspace " + quote(w) + " {" + rule + "}\n"
	}
	terminal, _ := argv(p.Terminal)
	helperPath := filepath.Join(m.Config, "dotawesome/niri/work")
	menuPath := filepath.Join(m.Config, "dotawesome/niri/jump.yaml")
	menu, e := read("jump.yaml")
	if e != nil {
		return nil, e
	}
	menu, e = jumpMenu(menu, helperPath, p.Apps)
	if e != nil {
		return nil, e
	}
	add(menuPath, menu, 0644)
	bindings := "    Mod+Return { spawn " + terminal + "; }\n    Mod+Shift+Return { spawn " + quote(helperPath) + "; }\n"
	bindings += "    Mod+G repeat=false hotkey-overlay-title=\"Jump to app\" { spawn \"wlr-which-key\" " + quote(menuPath) + "; }\n"
	binds := [][2]string{{"Mod+Space", "panel-toggle launcher"}, {"Mod+V", "panel-toggle clipboard"}, {"Mod+S", "panel-toggle control-center"}, {"Mod+Alt+L", "lock"}, {"Mod+N", "panel-toggle control-center notifications"}, {"Mod+B", "panel-toggle control-center network"}, {"Mod+Shift+B", "panel-toggle control-center bluetooth"}, {"XF86AudioRaiseVolume", "volume-up"}, {"XF86AudioLowerVolume", "volume-down"}, {"XF86AudioMute", "volume-mute"}, {"XF86AudioMicMute", "mic-mute"}, {"XF86AudioPlay", "media toggle"}, {"XF86AudioNext", "media next"}, {"XF86AudioPrev", "media previous"}, {"XF86MonBrightnessUp", "brightness-up"}, {"XF86MonBrightnessDown", "brightness-down"}}
	for _, b := range binds {
		locked := ""
		if strings.HasPrefix(b[0], "XF86") {
			locked = " allow-when-locked=true"
		}
		args, _ := argv(strings.Fields(b[1]))
		bindings += "    " + b[0] + locked + " { spawn \"noctalia\" \"msg\" " + args + "; }\n"
	}
	config = strings.ReplaceAll(config, "// @DOTAWESOME_COMMAND_BINDS@", strings.TrimSuffix(bindings, "\n"))
	add(filepath.Join(m.Config, "niri/config.kdl"), []byte(config), 0644)
	if len(helper) == 0 {
		return nil, errors.New("compiled work helper missing; rebuild bin/dotawesome")
	}
	add(helperPath, helper, 0755)
	b, _ := json.MarshalIndent(p, "", "  ")
	add(filepath.Join(m.Config, "dotawesome/niri/profile.json"), append(b, '\n'), 0644)
	shell, e := read("noctalia.toml")
	if e != nil {
		return nil, e
	}
	if p.Name == "desktop" {
		// Desktop metrics include temperature and omit laptop battery status.
		shell = []byte(strings.Replace(string(shell), `members = ["cpu", "ram"]`, `members = ["cpu", "cpu-temp", "ram"]`, 1))
		shell = []byte(strings.Replace(string(shell), `"volume", "battery", "caffeine"`, `"volume", "caffeine"`, 1))
		desktop, err := read("noctalia-desktop.toml")
		if err != nil {
			return nil, err
		}
		shell = append(append(shell, '\n'), desktop...)
	}
	expand := func(s string) (string, error) {
		if !validText(s) {
			return "", errors.New("invalid wallpaper path")
		}
		if strings.HasPrefix(s, "~/") {
			s = filepath.Join(m.Home, s[2:])
		}
		if !filepath.IsAbs(s) {
			return "", errors.New("wallpaper path must be absolute or start with ~/")
		}
		return s, nil
	}
	if p.Wallpaper.Default != "" {
		s, e := expand(p.Wallpaper.Default)
		if e != nil {
			return nil, e
		}
		shell = append(shell, []byte("\n[wallpaper.default]\npath = "+quote(s)+"\n")...)
	}
	for _, v := range p.Wallpaper.Favorites {
		s, e := expand(v)
		if e != nil {
			return nil, e
		}
		shell = append(shell, []byte("\n[[wallpaper.favorite]]\npath = "+quote(s)+"\n")...)
	}
	var check map[string]any
	if e = toml.Unmarshal(shell, &check); e != nil {
		return nil, e
	}
	add(filepath.Join(m.Config, "dotawesome/niri/noctalia/noctalia/config.toml"), shell, 0644)
	for _, palette := range []string{"Architectural-Glass", "Graphite-Jade"} {
		b, e := read("palettes/" + palette + ".json")
		if e != nil {
			return nil, e
		}
		add(filepath.Join(m.Config, "dotawesome/niri/noctalia/noctalia/palettes", palette+".json"), b, 0644)
	}
	unit, e := read("noctalia.service")
	if e != nil {
		return nil, e
	}
	unitPath := func(p string) string { return strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "%", "%%").Replace(p) }
	unit = []byte(strings.NewReplacer("@CONFIG@", unitPath(m.Config), "@STATE@", unitPath(m.State)).Replace(string(unit)))
	add(filepath.Join(m.Config, "systemd/user/dotawesome-niri-noctalia.service"), unit, 0644)
	add(filepath.Join(m.Config, "systemd/user/niri.service.d/50-dotawesome.conf"), []byte("[Unit]\nWants=dotawesome-niri-noctalia.service\n"), 0644)
	portals := "[preferred]\ndefault=gnome;gtk;\norg.freedesktop.impl.portal.FileChooser=gtk\n"
	if p.SecretPortal != "" {
		portals += "org.freedesktop.impl.portal.Secret=" + p.SecretPortal + "\n"
	}
	add(filepath.Join(m.Config, "xdg-desktop-portal/niri-portals.conf"), []byte(portals), 0644)
	for dest, value := range map[string]string{"vivaldi-stable.conf": p.Vivaldi, "signal-desktop-flags.conf": p.Signal} {
		if value != "" {
			add(filepath.Join(m.Config, dest), []byte("--password-store=kwallet6\n"), 0644)
		}
	}
	return files, nil
}
