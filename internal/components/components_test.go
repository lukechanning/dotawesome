package components

import (
	"context"
	"dotawesome/internal/deploy"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestToolbarMatchesMachineProfile(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	m := deploy.New(repo, t.TempDir())
	for _, profile := range []string{"generic", "g14", "desktop"} {
		files, err := Render(m, "niri", profile, []byte("helper"))
		if err != nil {
			t.Fatal(err)
		}
		var shell struct {
			Bar map[string]struct {
				Groups []struct {
					ID      string   `toml:"id"`
					Members []string `toml:"members"`
				} `toml:"capsule_group"`
			} `toml:"bar"`
			Widget map[string]struct {
				Stat string `toml:"stat"`
			} `toml:"widget"`
		}
		if err := toml.Unmarshal(files[filepath.Join(m.Config, "dotawesome/niri/noctalia/noctalia/config.toml")].Bytes(), &shell); err != nil {
			t.Fatal(err)
		}
		members := map[string]bool{}
		for _, group := range shell.Bar["default"].Groups {
			for _, member := range group.Members {
				members[member] = true
			}
		}
		desktop := profile == "desktop"
		if members["battery"] == desktop || members["cpu-temp"] != desktop || !members["cpu"] || !members["ram"] {
			t.Fatalf("%s: unexpected toolbar members: %v", profile, members)
		}
		if desktop && shell.Widget["cpu-temp"].Stat != "cpu_temp" {
			t.Fatal("desktop CPU temperature sensor missing")
		}
	}
}

func TestJumpMenuUsesProfileIDsAndQuotesHelperPath(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	m := deploy.New(repo, filepath.Join(t.TempDir(), "home 'quoted' $literal"))
	m.Config = filepath.Join(m.Home, ".config")
	for _, profile := range []string{"generic", "g14", "desktop"} {
		files, err := Render(m, "niri", profile, []byte("helper"))
		if err != nil {
			t.Fatal(err)
		}
		menuPath := filepath.Join(m.Config, "dotawesome/niri/jump.yaml")
		menuFile := files[menuPath]
		if menuFile == nil {
			t.Fatal("jump menu not deployed")
		}
		var menu struct{ Menu []struct{ Key, Cmd string } }
		if err := json.Unmarshal(menuFile.Bytes(), &menu); err != nil {
			t.Fatal(err)
		}
		commands := map[string]string{}
		for _, entry := range menu.Menu {
			commands[entry.Key] = entry.Cmd
		}
		if len(commands) != 4 || commands["/"] != "noctalia msg panel-open launcher '/win '" {
			t.Fatal(commands)
		}
		helper := filepath.Join(m.Config, "dotawesome/niri/work")
		if err := os.MkdirAll(filepath.Dir(helper), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0755); err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{
			"s": "focus\nslack\nSlack\ncom.slack.Slack\n",
			"b": "focus\nvivaldi-stable\nvivaldi\n",
			"t": "focus\nAlacritty\nalacritty\n",
		} {
			out, err := exec.Command("sh", "-c", commands[key]).CombinedOutput()
			if err != nil || string(out) != want {
				t.Fatalf("%s: %q, %v", key, out, err)
			}
		}
		config := string(files[filepath.Join(m.Config, "niri/config.kdl")].Bytes())
		if !strings.Contains(config, `Mod+G repeat=false hotkey-overlay-title="Jump to app" { spawn "wlr-which-key" `+quote(menuPath)) {
			t.Fatal("missing Super+G menu binding")
		}
	}
}

func TestRenderProfilesAndApplications(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	m := deploy.New(repo, t.TempDir())
	m.Config = filepath.Join(m.Home, ".config")
	for _, name := range []string{"tmux", "nvim", "niri"} {
		for _, profile := range []string{"generic", "g14", "desktop"} {
			if name != "niri" && profile != "generic" {
				continue
			}
			p := ""
			if name == "niri" {
				p = profile
			}
			f, e := Render(m, name, p, []byte("executable"))
			if e != nil {
				t.Fatal(name, p, e)
			}
			if len(f) == 0 {
				t.Fatal("empty render")
			}
			for path, v := range f {
				if !strings.HasPrefix(path, m.Home+"/") {
					t.Fatal(path)
				}
				if strings.Contains(path, "work.py") {
					t.Fatal("retired python helper")
				}
				if _, e := os.Stat(path); !os.IsNotExist(e) {
					t.Fatal("render wrote destination")
				}
				if len(v.Bytes()) == 0 {
					t.Fatal(path)
				}
			}
		}
	}
}
func TestRejectInvalidProfiles(t *testing.T) {
	for _, p := range []Profile{{}, {Terminal: []string{"alacritty\n"}}, {Terminal: []string{"alacritty"}, SecretPortal: "unexpected"}, {Terminal: []string{"alacritty"}, External: &Output{Identity: "HDMI-A-1", Mode: "1920x1080", Scale: 1}}} {
		if e := p.Validate(); e == nil {
			t.Fatal(p)
		}
	}
}
func TestRetiresLegacyHelperThroughBackedUpTransaction(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	home := t.TempDir()
	m := deploy.New(repo, home)
	m.Config = filepath.Join(home, ".config")
	m.State = filepath.Join(home, "state")
	m.Build = filepath.Join(home, "snapshots")
	m.Run = func(context.Context, ...string) (string, error) { return "", nil }
	oldHelper := filepath.Join(m.Config, "dotawesome/niri/work.py")
	config := filepath.Join(m.Config, "niri/config.kdl")
	old := deploy.File([]byte("legacy helper"), 0644)
	oldConfig := deploy.File([]byte("legacy config"), 0644)
	m.Write(oldHelper, old)
	m.Write(config, oldConfig)
	link := "original-config"
	legacy := deploy.Records{"niri": {Files: map[string]deploy.Entry{oldHelper: {Installed: old}, config: {Original: &deploy.Value{Link: &link}, Installed: oldConfig}}}}
	os.MkdirAll(m.State, 0700)
	m.SaveJSON(filepath.Join(m.State, "installed.json"), legacy)
	files, e := Render(m, "niri", "g14", []byte("helper"))
	if e != nil {
		t.Fatal(e)
	}
	plan, e := m.Plan("niri", files)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Apply(context.Background(), plan, "g14", true); e != nil {
		t.Fatal(e)
	}
	if deploy.Exists(oldHelper) {
		t.Fatal("retired helper remained")
	}
	plan, e = m.Plan("niri", nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Apply(context.Background(), plan, "", true); e != nil {
		t.Fatal(e)
	}
	got, e := os.Readlink(config)
	if e != nil || got != link {
		t.Fatal(got, e)
	}
}

func TestDesktopAndG14ReuseKDEWallet(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	m := deploy.New(repo, t.TempDir())
	for _, profile := range []string{"desktop", "g14"} {
		files, err := Render(m, "niri", profile, []byte("helper"))
		if err != nil {
			t.Fatal(err)
		}
		portal := files[filepath.Join(m.Config, "xdg-desktop-portal/niri-portals.conf")]
		if portal == nil || !strings.Contains(string(portal.Bytes()), "org.freedesktop.impl.portal.Secret=kwallet\n") {
			t.Fatalf("%s does not route secrets to KDE wallet", profile)
		}
		for _, name := range []string{"vivaldi-stable.conf", "signal-desktop-flags.conf"} {
			f := files[filepath.Join(m.Config, name)]
			if f == nil || string(f.Bytes()) != "--password-store=kwallet6\n" {
				t.Fatalf("%s: %s does not use kwallet6", profile, name)
			}
		}
	}
}
