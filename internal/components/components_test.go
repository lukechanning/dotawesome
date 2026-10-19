package components

import (
	"context"
	"dotawesome/internal/deploy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
