package app

import (
	"context"
	"dotawesome/internal/deploy"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedOwnershipBoundary(t *testing.T) {
	for _, p := range []string{".config/niri/config.kdl", ".config/nvim", ".config", "/tmp/escape", "../escape"} {
		if e := CheckOwnership([]string{p}, nil); e == nil {
			t.Fatal(p)
		}
	}
	if e := CheckOwnership([]string{".gitconfig", ".config/fish/conf.d/local.fish"}, []string{".config", ".config/fish"}); e != nil {
		t.Fatal(e)
	}
}
func TestIdentityValidationPreservesUnsafeIdentity(t *testing.T) {
	s := service(t)
	p := filepath.Join(s.Manager.Home, "key")
	os.WriteFile(p, []byte("private"), 0644)
	if e := s.ValidateIdentity(context.Background(), p, "age1test"); e == nil {
		t.Fatal("accepted public identity mode")
	}
	os.Chmod(p, 0600)
	s.Manager.Run = func(context.Context, ...string) (string, error) { return "age1other", nil }
	if e := s.ValidateIdentity(context.Background(), p, "age1test"); e == nil {
		t.Fatal("accepted wrong recipient")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "private" {
		t.Fatal("overwrote identity")
	}
}
func TestEnrollmentRefusesPartialState(t *testing.T) {
	s := service(t)
	s.Manager.Repo = t.TempDir()
	os.MkdirAll(filepath.Join(s.Manager.Repo, "encryption"), 0700)
	os.WriteFile(filepath.Join(s.Manager.Repo, "encryption/recipient.txt"), []byte("existing"), 0600)
	if e := s.Enroll(context.Background()); e == nil {
		t.Fatal("replaced partial enrollment")
	}
	if deploy.Exists(filepath.Join(s.Manager.Config, "chezmoi/key.txt")) {
		t.Fatal("generated identity")
	}
}
func TestSecretScanUsesRedactionAndSkipsSymlinks(t *testing.T) {
	s := service(t)
	s.Manager.Repo = t.TempDir()
	os.WriteFile(filepath.Join(s.Manager.Repo, "tracked"), []byte("safe fixture"), 0600)
	outside := filepath.Join(t.TempDir(), "private")
	os.WriteFile(outside, []byte("outside"), 0600)
	os.Symlink(outside, filepath.Join(s.Manager.Repo, "link"))
	s.Manager.Run = func(context.Context, ...string) (string, error) { return "tracked\x00link\x00", nil }
	s.Terminal = func(ctx context.Context, args ...string) error {
		if args[1] != "--redact=100" {
			t.Fatal(args)
		}
		dir := args[len(args)-1]
		if !deploy.Exists(filepath.Join(dir, "tracked")) || deploy.Exists(filepath.Join(dir, "link")) {
			t.Fatal("incorrect staged scan files")
		}
		return nil
	}
	if e := s.ScanSecrets(context.Background(), false); e != nil {
		t.Fatal(e)
	}
}
