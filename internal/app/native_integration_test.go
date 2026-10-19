package app

import (
	"context"
	"dotawesome/internal/components"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeValidatorsIntegration(t *testing.T) {
	if os.Getenv("DOTAWESOME_INTEGRATION") == "" {
		t.Skip("bin/ci --integration")
	}
	s := service(t)
	s.Manager.Run = nil
	real := New(s.Manager, []byte("helper"))
	for _, name := range []string{"tmux", "nvim", "niri"} {
		profiles := []string{""}
		if name == "niri" {
			profiles = []string{"generic", "g14", "desktop"}
		}
		for _, p := range profiles {
			t.Run(name+"/"+p, func(t *testing.T) {
				files, e := components.Render(s.Manager, name, p, s.Helper)
				if e != nil {
					t.Fatal(e)
				}
				plan, e := s.Manager.Plan(name, files)
				if e != nil {
					t.Fatal(e)
				}
				if e = real.Validate(context.Background(), &Inspection{Plan: plan, Profile: p}); e != nil {
					t.Fatal(e)
				}
			})
		}
	}
	paths, _ := filepath.Glob(filepath.Join(s.Manager.Repo, "home/dot_config/fish/**/*.fish"))
	for _, p := range paths {
		if _, e := real.Manager.Run(context.Background(), "fish", "-n", p); e != nil {
			t.Fatal(e)
		}
	}
}
func TestNativeDeploymentCycleIntegration(t *testing.T) {
	if os.Getenv("DOTAWESOME_INTEGRATION") == "" {
		t.Skip("bin/ci --integration")
	}
	t.Setenv("NIRI_SOCKET", "")
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	s := service(t)
	real := New(s.Manager, []byte("helper"))
	native := real.Manager.Run
	real.Manager.Run = func(ctx context.Context, args ...string) (string, error) {
		if args[0] == "systemctl" || args[0] == "pacman" {
			return "", nil
		}
		return native(ctx, args...)
	}
	for _, name := range []string{"tmux", "nvim", "niri"} {
		t.Run(name, func(t *testing.T) {
			profile := ""
			if name == "niri" {
				profile = "g14"
			}
			files, e := components.Render(real.Manager, name, profile, real.Helper)
			if e != nil {
				t.Fatal(e)
			}
			plan, e := real.Manager.Plan(name, files)
			if e != nil {
				t.Fatal(e)
			}
			r := &Inspection{Plan: plan, Profile: profile}
			if e = real.Execute(context.Background(), r, false); e != nil {
				t.Fatal(e)
			}
			plan, e = real.Manager.Plan(name, files)
			if e != nil || len(plan.Changes) != 0 {
				t.Fatal("repeat update changed files", e)
			}
			restore, e := real.Manager.Plan(name, nil)
			if e != nil {
				t.Fatal(e)
			}
			if e = real.Execute(context.Background(), &Inspection{Plan: restore, Restore: true}, false); e != nil {
				t.Fatal(e)
			}
			for p := range files {
				if _, e = os.Lstat(p); !os.IsNotExist(e) {
					t.Fatal("created file retained after restore", p, e)
				}
			}
		})
	}
}
