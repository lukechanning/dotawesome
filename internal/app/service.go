// Package app shares typed inspections and operations between the CLI and TUI.
package app

import (
	"context"
	"dotawesome/internal/components"
	"dotawesome/internal/deploy"
	"dotawesome/internal/external"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Service struct {
	Manager        *deploy.Manager
	Helper         []byte
	Lookup         func(string) bool
	Terminal       func(context.Context, ...string) error
	EnvironmentRun func(context.Context, []string, ...string) (string, error)
}
type Inspection struct {
	Plan                       *deploy.Plan
	Profile                    string
	Restore                    bool
	Errors, Warnings, Packages []string
}
type Event struct{ Component, State, Detail string }
type Result struct{ Component, State, Detail, Backup string }

func New(m *deploy.Manager, helper []byte) *Service {
	m.Run = external.Run
	s := &Service{Manager: m, Helper: helper, Lookup: func(s string) bool { _, e := exec.LookPath(s); return e == nil }, Terminal: external.Terminal, EnvironmentRun: external.Env}
	m.BeforeRecovery = func() error { return s.OutsideSession(context.Background()) }
	return s
}
func (s *Service) Inspect(ctx context.Context, name, profile string, restore bool) (*Inspection, error) {
	if name == "niri" && profile != "" && profile != "generic" && profile != "g14" && profile != "desktop" {
		absolute, e := filepath.Abs(profile)
		if e != nil {
			return nil, e
		}
		profile = absolute
	}
	if _, e := s.packageRecords(); e != nil {
		return nil, e
	}
	var files deploy.Files
	var e error
	if !restore {
		files, e = components.Render(s.Manager, name, profile, s.Helper)
		if e != nil {
			return nil, e
		}
	}
	p, e := s.Manager.Plan(name, files)
	if e != nil {
		return nil, e
	}
	r := &Inspection{Plan: p, Profile: profile, Restore: restore}
	if restore {
		return r, nil
	}
	r.Packages = []string{"tmux"}
	commands := []string{"tmux"}
	if name == "nvim" {
		commands = []string{"nvim", "git", "rg"}
		r.Packages = []string{"neovim", "git", "ripgrep"}
		if deploy.Exists(filepath.Join(s.Manager.Config, "nvim/init.vim")) {
			r.Errors = append(r.Errors, "existing nvim/init.vim conflicts with init.lua; move it aside before applying")
		}
	}
	if name == "niri" {
		var def struct {
			Commands, Packages []string
			Minimum            map[string]string `json:"minimum_versions"`
		}
		if e = deploy.ReadJSON(filepath.Join(s.Manager.Repo, "components/niri/definition.json"), &def); e != nil {
			return nil, e
		}
		commands = def.Commands
		r.Packages = def.Packages
		for cmd, min := range def.Minimum {
			if s.Lookup(cmd) {
				out, e := s.Manager.Run(ctx, cmd, "--version")
				if e != nil || !versionAtLeast(out, min) {
					r.Errors = append(r.Errors, cmd+": requires >= "+min)
				}
			}
		}
		for _, p := range []string{"/usr/share/wayland-sessions/niri.desktop", "/usr/lib/systemd/user/niri.service", "/etc/pam.d/login"} {
			if !deploy.Exists(p) {
				r.Errors = append(r.Errors, "missing session prerequisite: "+p)
			}
		}
		if !s.Lookup("pacman") {
			r.Errors = append(r.Errors, "supported package platform requires pacman (Arch/CachyOS)")
		} else {
			for _, pkg := range r.Packages {
				if _, e = s.Manager.Run(ctx, "pacman", "-Q", pkg); e != nil {
					r.Errors = append(r.Errors, "missing package: "+pkg)
				}
			}
		}
		conflicts, e := startupConflicts(s.Manager)
		if e != nil {
			return nil, e
		}
		r.Errors = append(r.Errors, conflicts...)
		p, _ := components.LoadProfile(s.Manager.Repo, profile)
		for _, a := range p.Apps {
			if len(a.Command) > 0 && !s.Lookup(a.Command[0]) {
				r.Warnings = append(r.Warnings, "optional "+a.Name+": missing "+a.Command[0])
			}
		}
		paths := append([]string{}, p.Wallpaper.Favorites...)
		if p.Wallpaper.Default != "" {
			paths = append(paths, p.Wallpaper.Default)
		}
		for _, path := range paths {
			if strings.HasPrefix(path, "~/") {
				path = filepath.Join(s.Manager.Home, path[2:])
			}
			if !deploy.Exists(path) {
				r.Warnings = append(r.Warnings, "missing local wallpaper: "+path+"; dark fallback will be used")
			}
		}
		if deploy.Exists(filepath.Join(s.Manager.State, "niri-shell/noctalia/settings.toml")) {
			r.Warnings = append(r.Warnings, "GUI overrides take precedence: "+filepath.Join(s.Manager.State, "niri-shell/noctalia/settings.toml"))
		}
	}
	for _, cmd := range commands {
		if !s.Lookup(cmd) {
			r.Errors = append(r.Errors, "missing command: "+cmd)
		}
	}
	if name == "nvim" && s.Lookup("nvim") {
		version, e := s.runEnvironment(ctx, []string{"NVIM_LOG_FILE=" + os.DevNull}, "nvim", "--version")
		if e != nil || !versionAtLeast(version, "0.12") {
			r.Errors = append(r.Errors, "Neovim >= 0.12 required")
		}
		r.Warnings = append(r.Warnings, "Pinned plugins download on first launch. Install language tools and parsers explicitly.")
	}
	return r, nil
}
func versionAtLeast(s, min string) bool {
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	v := re.FindStringSubmatch(s)
	want := strings.Split(min, ".")
	if len(v) != 3 || len(want) != 2 {
		return false
	}
	a, _ := strconv.Atoi(v[1])
	b, _ := strconv.Atoi(v[2])
	x, _ := strconv.Atoi(want[0])
	y, _ := strconv.Atoi(want[1])
	return a > x || a == x && b >= y
}
func (s *Service) OutsideSession(ctx context.Context) error {
	if os.Getenv("NIRI_SOCKET") != "" || strings.Contains(strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP")), "niri") {
		return errors.New("log out of Niri before changing this deployment")
	}
	v, _ := s.Manager.Run(ctx, "systemctl", "--user", "is-active", "niri.service")
	if v == "active" {
		return errors.New("niri.service is active; log out of Niri first")
	}
	return nil
}
func (s *Service) Execute(ctx context.Context, r *Inspection, live bool) error {
	if len(r.Errors) > 0 {
		return fmt.Errorf("preflight failed; no configuration deployed: %s", strings.Join(r.Errors, "; "))
	}
	if r.Plan.Component == "niri" && !live {
		if e := s.OutsideSession(ctx); e != nil {
			return e
		}
	}
	if live {
		if e := s.ValidateLive(r); e != nil {
			return e
		}
		r.Plan.Live = true
	}
	if !r.Restore {
		fresh, e := components.Render(s.Manager, r.Plan.Component, r.Profile, s.Helper)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(fresh, r.Plan.Files) {
			return errors.New("source changed since review; inspect and confirm a fresh plan")
		}
		if e = s.Validate(ctx, r); e != nil {
			return e
		}
	}
	if !live && deploy.Exists(filepath.Join(s.Manager.State, "transaction.json")) {
		var tx deploy.Transaction
		if e := deploy.ReadJSON(filepath.Join(s.Manager.State, "transaction.json"), &tx); e != nil {
			return e
		}
		if tx.Reload == nil || *tx.Reload {
			if e := s.OutsideSession(ctx); e != nil {
				return e
			}
		}
	}
	guard := func() error {
		if _, e := s.packageRecords(); e != nil {
			return e
		}
		if live {
			if e := s.ValidateLive(r); e != nil {
				return e
			}
		}
		if !r.Restore {
			fresh, e := components.Render(s.Manager, r.Plan.Component, r.Profile, s.Helper)
			if e != nil {
				return e
			}
			if !reflect.DeepEqual(fresh, r.Plan.Files) {
				return errors.New("source changed since review; inspect and confirm a fresh plan")
			}
		}
		return nil
	}
	return s.Manager.Apply(ctx, r.Plan, r.Profile, r.Plan.Component == "niri", guard)
}
func (s *Service) runEnvironment(ctx context.Context, env []string, args ...string) (string, error) {
	if s.EnvironmentRun != nil {
		return s.EnvironmentRun(ctx, env, args...)
	}
	return s.Manager.Run(ctx, args...)
}
func (s *Service) Validate(ctx context.Context, r *Inspection) error {
	work, e := os.MkdirTemp("", "dotawesome-validation-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(work)
	put := func(name string, v *deploy.Value) (string, error) {
		path := filepath.Join(work, name)
		if v == nil {
			return "", fmt.Errorf("missing validation input: %s", name)
		}
		if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return "", e
		}
		return path, os.WriteFile(path, v.Bytes(), 0600)
	}
	switch r.Plan.Component {
	case "tmux":
		p, e := put("tmux.conf", r.Plan.Files[filepath.Join(s.Manager.Home, ".tmux.conf")])
		if e != nil {
			return e
		}
		socket := filepath.Join(work, "socket")
		defer s.runEnvironment(context.Background(), []string{"HOME=" + work, "TMUX="}, "tmux", "-S", socket, "kill-server")
		_, e = s.runEnvironment(ctx, []string{"HOME=" + work, "TMUX="}, "tmux", "-S", socket, "-f", "/dev/null", "start-server", ";", "set-option", "-g", "exit-empty", "off", ";", "source-file", p)
		return e
	case "nvim":
		for p, v := range r.Plan.Files {
			rel, _ := filepath.Rel(filepath.Join(s.Manager.Config, "nvim"), p)
			if _, e = put(filepath.Join("config/nvim", rel), v); e != nil {
				return e
			}
		}
		env := []string{"HOME=" + work, "XDG_CONFIG_HOME=" + filepath.Join(work, "config"), "XDG_DATA_HOME=" + filepath.Join(work, "data"), "XDG_STATE_HOME=" + filepath.Join(work, "state"), "XDG_CACHE_HOME=" + filepath.Join(work, "cache"), "NVIM_LOG_FILE=" + filepath.Join(work, "nvim.log"), "NVIM_APPNAME=nvim", "DOTAWESOME_NVIM_VALIDATE=1"}
		out, e := s.runEnvironment(ctx, env, "nvim", "--headless", "-i", "NONE", `+lua for _, p in ipairs(vim.fn.globpath(vim.fn.stdpath("config"), "**/*.lua", false, true)) do assert(loadfile(p)) end`, `+lua if vim.v.errmsg ~= "" then error(vim.v.errmsg) end`, "+qa!")
		if e != nil {
			return e
		}
		if regexp.MustCompile(`Error|E\d{3}:`).MatchString(out) {
			return fmt.Errorf("Neovim validation failed: %s", out)
		}
	case "niri":
		menu, e := put("jump.yaml", r.Plan.Files[filepath.Join(s.Manager.Config, "dotawesome/niri/jump.yaml")])
		if e != nil {
			return e
		}
		config, e := put("config.kdl", r.Plan.Files[filepath.Join(s.Manager.Config, "niri/config.kdl")])
		if e != nil {
			return e
		}
		root := filepath.Join(s.Manager.Config, "dotawesome/niri/noctalia/noctalia")
		shell, e := put("config.toml", r.Plan.Files[filepath.Join(root, "config.toml")])
		if e != nil {
			return e
		}
		for p, v := range r.Plan.Files {
			if strings.HasPrefix(p, root+"/palettes/") {
				rel, _ := filepath.Rel(root, p)
				if _, e = put(rel, v); e != nil {
					return e
				}
			}
		}
		unit, e := put("dotawesome-niri-noctalia.service", r.Plan.Files[filepath.Join(s.Manager.Config, "systemd/user/dotawesome-niri-noctalia.service")])
		if e != nil {
			return e
		}
		for _, args := range [][]string{{"wlr-which-key", menu, "--initial-keys", "Escape"}, {"niri", "validate", "--config", config}, {"noctalia", "config", "validate", shell}, {"systemd-analyze", "--user", "verify", unit}} {
			out, e := s.Manager.Run(ctx, args...)
			if e != nil {
				return e
			}
			if regexp.MustCompile(`(?i)\b(warn(ing)?|unknown|obsolete|error)\b`).MatchString(out) {
				return fmt.Errorf("native validation warning/error: %s", out)
			}
		}
	}
	return nil
}
func (s *Service) Status(ctx context.Context) ([]Result, error) {
	records, e := s.Manager.Records()
	if e != nil {
		return nil, e
	}
	packages, e := s.packageRecords()
	if e != nil {
		return nil, e
	}
	results := []Result{}
	for _, name := range components.Names {
		r, ok := records[name]
		state := "inactive"
		detail := ""
		if ok {
			state = "current"
			for p, v := range r.Files {
				current, e := deploy.Inspect(p)
				if e != nil {
					return nil, e
				}
				if !reflect.DeepEqual(current, v.Installed) {
					state = "locally-modified"
					detail += p + "\n"
				}
			}
			profile := ""
			if r.Profile != nil {
				profile = *r.Profile
			}
			f, e := components.Render(s.Manager, name, profile, s.Helper)
			if e != nil {
				detail += "source unavailable: " + e.Error()
			} else {
				installed := deploy.Files{}
				for p, v := range r.Files {
					installed[p] = v.Installed
				}
				if !reflect.DeepEqual(f, installed) {
					if state == "current" {
						state = "source-changed"
					} else {
						detail += "source-changed"
					}
				}
			}
		}
		pkg := packages[name]
		if len(pkg.Roots) > 0 {
			detail += "\nPackage additions: " + strings.Join(pkg.Roots, ", ")
		}
		if pkg.Pending || pkg.Removing {
			detail += "\nInterrupted package operation: inspect packages.json and pacman logs."
		}
		results = append(results, Result{name, state, detail, r.Snapshot})
	}
	if deploy.Exists(filepath.Join(s.Manager.State, "transaction.json")) {
		results = append(results, Result{State: "recovery pending", Detail: "Interrupted deployment: recovery pending on next mutation."})
	}
	return results, nil
}
func startupConflicts(m *deploy.Manager) ([]string, error) {
	dirs := strings.Split(os.Getenv("XDG_CONFIG_DIRS"), ":")
	if len(dirs) == 1 && dirs[0] == "" {
		dirs = []string{"/etc/xdg"}
	}
	dirs = append([]string{m.Config}, dirs...)
	seen := map[string]bool{}
	conflicts := []string{}
	re := regexp.MustCompile(`\b(noctalia(-shell)?|mako|swayidle|swaylock|waybar|dunst|swaync|polkit-kde-authentication-agent(-1)?)\b`)
	for _, dir := range dirs {
		entries, _ := filepath.Glob(filepath.Join(dir, "autostart/*.desktop"))
		for _, p := range entries {
			if seen[filepath.Base(p)] {
				continue
			}
			seen[filepath.Base(p)] = true
			b, e := os.ReadFile(p)
			if e != nil {
				return nil, e
			}
			values := map[string]string{}
			active := false
			for _, line := range strings.Split(string(b), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "[") {
					active = line == "[Desktop Entry]"
				}
				if active {
					a, b, ok := strings.Cut(line, "=")
					if ok {
						values[a] = b
					}
				}
			}
			only := strings.ToLower(values["OnlyShowIn"])
			exclude := strings.ToLower(values["NotShowIn"])
			if strings.EqualFold(values["Hidden"], "true") || only != "" && !containsDesktop(only, "niri") || containsDesktop(exclude, "niri") {
				continue
			}
			if re.MatchString(values["Exec"]) {
				conflicts = append(conflicts, "competing Niri autostart: "+p+"; scope or disable before activation")
			}
		}
	}
	root := filepath.Join(m.Config, "systemd/user")
	filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil || d.IsDir() || filepath.Base(p) == "dotawesome-niri-noctalia.service" {
			return nil
		}
		if !strings.Contains(p, ".wants/") && !strings.Contains(p, ".requires/") {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		if re.Match(b) {
			conflicts = append(conflicts, "competing enabled user unit: "+p+"; inspect session scope")
		}
		return nil
	})
	sort.Strings(conflicts)
	return conflicts, nil
}
func containsDesktop(s, name string) bool {
	for _, v := range strings.Split(s, ";") {
		if v == name {
			return true
		}
	}
	return false
}
