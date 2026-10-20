package app

import (
	"context"
	"dotawesome/internal/deploy"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNiriValidationRejectsInvalidJumpMenu(t *testing.T) {
	s := service(t)
	r, err := s.Inspect(context.Background(), "niri", "generic", false)
	if err != nil {
		t.Fatal(err)
	}
	invalid := errors.New("invalid jump menu")
	s.Manager.Run = func(_ context.Context, args ...string) (string, error) {
		if args[0] == "wlr-which-key" {
			if len(args) != 4 || args[2] != "--initial-keys" || args[3] != "Escape" {
				t.Fatalf("unsafe menu validation: %v", args)
			}
			if _, err := os.ReadFile(args[1]); err != nil {
				t.Fatal(err)
			}
			return "", invalid
		}
		return "", nil
	}
	if err := s.Validate(context.Background(), r); !errors.Is(err, invalid) {
		t.Fatalf("menu validation bypassed: %v", err)
	}
}

func service(t *testing.T) *Service {
	r, _ := filepath.Abs("../..")
	m := deploy.New(r, t.TempDir())
	m.Config = filepath.Join(m.Home, ".config")
	m.State = filepath.Join(m.Home, ".local/state/dotawesome")
	m.Run = func(context.Context, ...string) (string, error) { return "", nil }
	return &Service{Manager: m, Helper: []byte("helper"), Lookup: func(string) bool { return true }}
}
func TestIsolatedSetupNoopDriftRestore(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	r, e := s.Inspect(ctx, "tmux", "", false)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Execute(ctx, r, false); e != nil {
		t.Fatal(e)
	}
	r, e = s.Inspect(ctx, "tmux", "", false)
	if e != nil || len(r.Plan.Changes) != 0 {
		t.Fatal(r, e)
	}
	p := filepath.Join(s.Manager.Home, ".tmux.conf")
	os.WriteFile(p, []byte("edit"), 0644)
	if _, e = s.Inspect(ctx, "tmux", "", false); e == nil {
		t.Fatal("drift accepted")
	}
	s.Manager.Write(p, r.Plan.Files[p])
	r, e = s.Inspect(ctx, "tmux", "", true)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Execute(ctx, r, false); e != nil {
		t.Fatal(e)
	}
	if deploy.Exists(p) {
		t.Fatal("restore retained created file")
	}
}
func TestPreflightBlocksMutation(t *testing.T) {
	s := service(t)
	s.Lookup = func(string) bool { return false }
	r, e := s.Inspect(context.Background(), "tmux", "", false)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Errors) == 0 {
		t.Fatal("missing dependency accepted")
	}
	if e = s.Execute(context.Background(), r, false); e == nil || !strings.Contains(e.Error(), "preflight") {
		t.Fatal(e)
	}
}
func TestPackageNamesAndProtection(t *testing.T) {
	if _, e := packageNames("bad package\n"); e == nil {
		t.Fatal("accepted malformed pacman output")
	}
	s := service(t)
	s.Manager.Run = func(ctx context.Context, a ...string) (string, error) {
		if a[1] == "-Qq" {
			return "tmux\npreexisting\nshared", nil
		}
		return "tmux\npreexisting\nshared", nil
	}
	records := map[string]PackageRecord{"tmux": {Added: []string{"tmux", "shared"}, Roots: []string{"tmux", "shared"}}, "nvim": {Required: []string{"shared"}}}
	os.MkdirAll(s.Manager.State, 0700)
	s.Manager.SaveJSON(filepath.Join(s.Manager.State, "packages.json"), records)
	got, e := s.PackagePlan(context.Background(), "tmux")
	if e != nil || strings.Join(got, ",") != "tmux" {
		t.Fatal(got, e)
	}
}
func TestLiveRejectsBehaviorAndAllowsStyle(t *testing.T) {
	s := service(t)
	m := s.Manager
	path := filepath.Join(m.Config, "niri/config.kdl")
	before := deploy.File([]byte("layout {\n gaps 8\n}\nbinds { Mod+Return { spawn \"alacritty\"; } }\n"), 0644)
	m.Write(path, before)
	records := deploy.Records{"niri": {Files: map[string]deploy.Entry{path: {Original: nil, Installed: before}}}}
	os.MkdirAll(m.State, 0700)
	m.SaveJSON(filepath.Join(m.State, "installed.json"), records)
	for _, tc := range []struct {
		data    string
		allowed bool
	}{{"layout {\n gaps 10\n}\nbinds { Mod+Return { spawn \"alacritty\"; } }\n", true}, {"layout {\n gaps 8\n}\nbinds { Mod+Return { spawn \"danger\"; } }\n", false}} {
		plan, e := m.Plan("niri", deploy.Files{path: deploy.File([]byte(tc.data), 0644)})
		if e != nil {
			t.Fatal(e)
		}
		r := &Inspection{Plan: plan}
		if e = s.ValidateLive(r); (e == nil) != tc.allowed {
			t.Fatal(tc, e)
		}
	}
}
func TestFailedPackageInstallReconcilesAttribution(t *testing.T) {
	s := service(t)
	queries := 0
	s.Manager.Run = func(ctx context.Context, a ...string) (string, error) {
		queries++
		if queries == 1 {
			return "preexisting", nil
		}
		if ctx.Err() != nil {
			t.Fatal("reconciliation inherited cancelled context")
		}
		return "preexisting\ntmux\ndep", nil
	}
	s.Terminal = func(context.Context, ...string) error { return context.Canceled }
	if e := s.InstallPackages(context.Background(), map[string][]string{"tmux": {"tmux", "preexisting"}}); e == nil {
		t.Fatal("failed command accepted")
	}
	r, e := s.packageRecords()
	if e != nil {
		t.Fatal(e)
	}
	entry := r["tmux"]
	if entry.Pending || strings.Join(entry.Added, ",") != "dep,tmux" || strings.Join(entry.Roots, ",") != "tmux" {
		t.Fatal(entry)
	}
}
func TestFailedPackageReconciliationRetainsPending(t *testing.T) {
	s := service(t)
	queries := 0
	s.Manager.Run = func(ctx context.Context, a ...string) (string, error) {
		queries++
		if queries == 1 {
			return "old", nil
		}
		return "", os.ErrPermission
	}
	s.Terminal = func(context.Context, ...string) error { return context.Canceled }
	if e := s.InstallPackages(context.Background(), map[string][]string{"tmux": {"tmux"}}); e == nil {
		t.Fatal("reconciliation error ignored")
	}
	r, e := s.packageRecords()
	if e != nil || !r["tmux"].Pending {
		t.Fatal(r, e)
	}
	if _, e = s.PackagePlan(context.Background(), "tmux"); e == nil {
		t.Fatal("cleaned up uncertain provenance")
	}
}
func TestPackageRecordKeepsLegacyArrayShape(t *testing.T) {
	b, _ := json.Marshal(PackageRecord{})
	if strings.Contains(string(b), "null") {
		t.Fatal(string(b))
	}
}
func TestNeovimInspectionDoesNotInitializeLiveLogger(t *testing.T) {
	s := service(t)
	isolated := false
	s.EnvironmentRun = func(ctx context.Context, env []string, args ...string) (string, error) {
		if args[0] == "nvim" && args[1] == "--version" {
			isolated = strings.Join(env, ",") == "NVIM_LOG_FILE="+os.DevNull
			return "NVIM v0.12.0", nil
		}
		return "", nil
	}
	if _, e := s.Inspect(context.Background(), "nvim", "", false); e != nil {
		t.Fatal(e)
	}
	if !isolated {
		t.Fatal("version check did not isolate Neovim's logger")
	}
}
func TestCustomProfileSourceIsAbsolute(t *testing.T) {
	s := service(t)
	source := filepath.Join(t.TempDir(), "custom.toml")
	os.WriteFile(source, []byte("name='custom'\nterminal=['alacritty']\n"), 0600)
	cwd, _ := os.Getwd()
	relative, _ := filepath.Rel(cwd, source)
	r, e := s.Inspect(context.Background(), "niri", relative, false)
	if e != nil {
		t.Fatal(e)
	}
	if r.Profile != source {
		t.Fatal("relative profile would break from another working directory:", r.Profile)
	}
}
