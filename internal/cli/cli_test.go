package cli

import (
	"bytes"
	"context"
	"dotawesome/internal/app"
	"dotawesome/internal/deploy"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandsAndInvalidFlags(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	m := deploy.New(repo, t.TempDir())
	s := app.New(m, []byte("helper"))
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{{[]string{"list"}, 0, "niri"}, {[]string{"plan", "tmux"}, 0, "file(s) changed"}, {[]string{"plan", "tmux", "--live"}, 1, "only valid for niri"}, {[]string{"check", "tmux", "--unknown"}, 2, "flag provided"}, {[]string{"apply", "tmux"}, 1, "declined"}, {[]string{"plan", "niri"}, 1, "requires --profile"}} {
		var out bytes.Buffer
		code := Run(context.Background(), s, tc.args, strings.NewReader("no\n"), &out, &out)
		if code != tc.code || !strings.Contains(out.String(), tc.want) {
			t.Fatal(tc.args, code, out.String())
		}
	}
}
func TestSetupHandoffReturnsTypedDecline(t *testing.T) {
	repo := t.TempDir()
	m := deploy.New(repo, t.TempDir())
	m.Config = filepath.Join(m.Home, ".config")
	m.State = filepath.Join(m.Home, "state")
	s := app.New(m, nil)
	s.Manager.Run = func(ctx context.Context, args ...string) (string, error) {
		if args[len(args)-1] == "--untracked-files=normal" {
			return "", nil
		}
		return "main", nil
	}
	called := false
	s.Terminal = func(context.Context, ...string) error { called = true; return nil }
	receipt := filepath.Join(t.TempDir(), "result.json")
	var out bytes.Buffer
	code := Run(context.Background(), s, []string{"setup-step", "--action", "pull", "--receipt", receipt}, strings.NewReader("no\n"), &out, &out)
	var result app.SetupResult
	e := deploy.ReadStateJSON(receipt, &result)
	if code != 0 || e != nil || result.Outcome != "skipped" || called {
		t.Fatal(code, result, e, out.String())
	}
	st, _ := os.Stat(receipt)
	if st.Mode().Perm() != 0600 {
		t.Fatal("public setup receipt")
	}
}

func TestActionHelpAndSyncthingRoutes(t *testing.T) {
	s := app.New(deploy.New(t.TempDir(), t.TempDir()), nil)
	s.Lookup = func(string) bool { return false }
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"help"}, 0, "bin/dotawesome syncthing setup"},
		{[]string{"syncthing", "status"}, 0, "missing tools"},
		{[]string{"syncthing", "setup"}, 1, "missing tools"},
		{[]string{"syncthing", "status", "extra"}, 2, "Usage"},
		{[]string{"app", "setup", "unknown"}, 2, ""},
	} {
		var out bytes.Buffer
		code := Run(context.Background(), s, tc.args, strings.NewReader("no\n"), &out, &out)
		if code != tc.code || !strings.Contains(out.String(), tc.want) {
			t.Fatal(tc, code, out.String())
		}
	}
}

func TestHandoffReceiptRetainsTailscaleFailure(t *testing.T) {
	receipt := filepath.Join(t.TempDir(), "result.json")
	t.Setenv("DOTAWESOME_HANDOFF_RECEIPT", receipt)
	s := app.New(deploy.New(t.TempDir(), t.TempDir()), nil)
	s.Lookup = func(string) bool { return true }
	s.Manager.Run = func(context.Context, ...string) (string, error) { return "", fmt.Errorf("not connected") }
	s.Terminal = func(context.Context, ...string) error { return fmt.Errorf("exit status 1") }
	var out bytes.Buffer
	code := Run(context.Background(), s, []string{"syncthing", "setup"}, strings.NewReader("yes\n"), &out, &out)
	var result app.SetupResult
	if e := deploy.ReadStateJSON(receipt, &result); e != nil {
		t.Fatal(e)
	}
	if code != 1 || result.Outcome != "failed" || !strings.Contains(result.Detail, "tailscaled service startup failed") || !strings.Contains(result.Detail, "journalctl -u tailscaled") {
		t.Fatal(code, result, out.String())
	}
}
