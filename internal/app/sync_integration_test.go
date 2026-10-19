package app

import (
	"context"
	"dotawesome/internal/deploy"
	"dotawesome/internal/external"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSyncFastForwardsAndRefusesDivergenceIntegration(t *testing.T) {
	if os.Getenv("DOTAWESOME_INTEGRATION") == "" {
		t.Skip("bin/ci --integration")
	}
	work := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid"}, args...)...)
		if b, e := c.CombinedOutput(); e != nil {
			t.Fatal(args, e, string(b))
		}
	}
	remote := filepath.Join(work, "remote.git")
	seed := filepath.Join(work, "seed")
	checkout := filepath.Join(work, "checkout")
	git("init", "--bare", "-q", remote)
	git("init", "-q", "-b", "main", seed)
	file := filepath.Join(seed, "configuration")
	commit := func(content string) {
		t.Helper()
		os.WriteFile(file, []byte(content), 0644)
		git("-C", seed, "add", "configuration")
		git("-C", seed, "commit", "-qm", content)
	}
	commit("original")
	git("-C", seed, "remote", "add", "origin", remote)
	git("-C", seed, "push", "-qu", "origin", "main")
	git("clone", "-q", "--branch", "main", remote, checkout)
	commit("updated")
	git("-C", seed, "push", "-q")
	home := filepath.Join(work, "home")
	m := deploy.New(checkout, home)
	m.Config = filepath.Join(home, ".config")
	m.State = filepath.Join(home, "state")
	s := New(m, nil)
	s.Terminal = func(ctx context.Context, args ...string) error { _, e := external.Run(ctx, args...); return e }
	r, e := s.PullSource(context.Background(), func(string) bool { return true })
	if e != nil || r.Outcome != "completed" {
		t.Fatal(r, e)
	}
	b, _ := os.ReadFile(filepath.Join(checkout, "configuration"))
	if string(b) != "updated" {
		t.Fatal(string(b))
	}
	os.WriteFile(filepath.Join(checkout, "configuration"), []byte("local edit"), 0644)
	if _, e = s.PullSource(context.Background(), func(string) bool { return true }); e == nil {
		t.Fatal("dirty checkout pulled")
	}
	git("-C", checkout, "add", "configuration")
	git("-C", checkout, "commit", "-qm", "local")
	commit("remote change")
	git("-C", seed, "push", "-q")
	if _, e = s.PullSource(context.Background(), func(string) bool { return true }); e == nil {
		t.Fatal("diverged branch merged")
	}
	b, _ = os.ReadFile(filepath.Join(checkout, "configuration"))
	if string(b) != "local edit" {
		t.Fatal("local commit lost")
	}
}
