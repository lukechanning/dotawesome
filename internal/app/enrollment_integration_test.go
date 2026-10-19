package app

import (
	"context"
	"dotawesome/internal/deploy"
	"dotawesome/internal/external"
	"fmt"
	"github.com/creack/pty"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// age's real terminal prompts stay inside a disposable PTY. No passphrase goes in argv.
func ageTerminal(t *testing.T) func(context.Context, ...string) error {
	t.Helper()
	return func(ctx context.Context, args ...string) error {
		c := exec.CommandContext(ctx, args[0], args[1:]...)
		c.Env = append(os.Environ(), "TERM=dumb")
		p, e := pty.Start(c)
		if e != nil {
			return e
		}
		defer p.Close()
		done := make(chan error, 1)
		go func() { done <- c.Wait() }()
		data := make(chan string, 32)
		go func() {
			b := make([]byte, 4096)
			for {
				n, e := p.Read(b)
				if n > 0 {
					data <- string(b[:n])
				}
				if e != nil {
					return
				}
			}
		}()
		pending := ""
		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		for {
			select {
			case chunk := <-data:
				pending += chunk
				if strings.Contains(pending, ":") {
					if strings.Contains(pending, "passphrase") || strings.Contains(pending, "Passphrase") {
						io.WriteString(p, "disposable test passphrase\n")
						pending = ""
					}
				}
			case e := <-done:
				return e
			case <-timer.C:
				c.Process.Kill()
				<-done
				return fmt.Errorf("age terminal timed out")
			}
		}
	}
}
func TestEnrollmentRecoveryIntegration(t *testing.T) {
	if os.Getenv("DOTAWESOME_INTEGRATION") == "" {
		t.Skip("bin/ci --integration")
	}
	for _, tool := range []string{"age", "age-keygen", "chezmoi", "gitleaks"} {
		if _, e := exec.LookPath(tool); e != nil {
			t.Fatal("missing integration prerequisite:", tool)
		}
	}
	repo := t.TempDir()
	source, _ := filepath.Abs("../..")
	for _, name := range []string{"home", "hooks"} {
		e := filepath.WalkDir(filepath.Join(source, name), func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			rel, _ := filepath.Rel(source, p)
			dst := filepath.Join(repo, rel)
			if d.IsDir() {
				return os.MkdirAll(dst, 0700)
			}
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			return os.WriteFile(dst, b, 0600)
		})
		if e != nil {
			t.Fatal(e)
		}
	}
	os.WriteFile(filepath.Join(repo, ".chezmoiroot"), []byte("home\n"), 0600)
	if _, e := external.Run(context.Background(), "git", "init", "-q", repo); e != nil {
		t.Fatal(e)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local/state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local/share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	m := deploy.New(repo, home)
	m.Config = filepath.Join(home, ".config")
	m.State = filepath.Join(home, ".local/state/dotawesome")
	s := New(m, nil)
	s.Terminal = ageTerminal(t)
	ctx := context.Background()
	if e := s.Enroll(ctx); e != nil {
		t.Fatal(e)
	}
	identity := filepath.Join(m.Config, "chezmoi/key.txt")
	recipient, _ := os.ReadFile(filepath.Join(repo, "encryption/recipient.txt"))
	if e := s.ValidateIdentity(ctx, identity, strings.TrimSpace(string(recipient))); e != nil {
		t.Fatal(e)
	}
	original, _ := os.ReadFile(identity)
	os.Remove(identity)
	if e := s.RecoverIdentity(ctx, identity, filepath.Join(repo, "encryption/identity.age"), strings.TrimSpace(string(recipient))); e != nil {
		t.Fatal(e)
	}
	recovered, _ := os.ReadFile(identity)
	if string(recovered) != string(original) {
		t.Fatal("recovery mismatch")
	}
	s.Terminal = func(ctx context.Context, args ...string) error {
		if args[0] == "chezmoi" {
			_, e := external.Run(ctx, args...)
			return e
		}
		return ageTerminal(t)(ctx, args...)
	}
	if e := s.Initialize(ctx, "g14", true, func(string) bool { return true }); e != nil {
		t.Fatal(e)
	}
	if !deploy.Exists(filepath.Join(home, ".gitconfig")) {
		t.Fatal("shared file not deployed")
	}
	if e := s.Initialize(ctx, "g14", true, func(string) bool { return true }); e != nil {
		t.Fatal("repeat initialization:", e)
	}
	before := treeContents(t, home)
	s.Lookup = func(string) bool { return true }
	r, e := s.CheckRig(ctx, "")
	if e != nil || r.Shared != "current" || r.Next(true) != "components" {
		t.Fatal(r, e)
	}
	if !reflect.DeepEqual(before, treeContents(t, home)) {
		t.Fatal("readiness check changed the home")
	}

}

func treeContents(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		st, e := os.Lstat(p)
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, p)
		if d.IsDir() {
			files[rel] = fmt.Sprint(st.Mode())
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		files[rel] = fmt.Sprint(st.Mode()) + string(b)
		return nil
	})
	return files
}
