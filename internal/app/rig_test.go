package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rigService(t *testing.T) *Service {
	t.Helper()
	s := service(t)
	s.Manager.Repo = t.TempDir()
	s.Manager.Run = func(ctx context.Context, args ...string) (string, error) {
		if args[0] == "git" {
			if args[3] == "config" {
				return "", errors.New("unset")
			}
			return filepath.Join(s.Manager.Repo, ".git/hooks/pre-commit"), nil
		}
		if args[0] == "age-keygen" {
			return "age1test", nil
		}
		if args[0] == "chezmoi" {
			if args[len(args)-1] == "relative" {
				return "", nil
			}
			return "", nil
		}
		return "", nil
	}
	os.MkdirAll(filepath.Join(s.Manager.Repo, "hooks"), 0700)
	os.WriteFile(filepath.Join(s.Manager.Repo, "hooks/pre-commit"), []byte("hook\n"), 0755)
	return s
}
func enrolledRig(t *testing.T, s *Service) {
	t.Helper()
	os.MkdirAll(filepath.Join(s.Manager.Repo, "encryption"), 0700)
	os.WriteFile(filepath.Join(s.Manager.Repo, "encryption/recipient.txt"), []byte("age1test\n"), 0600)
	os.WriteFile(filepath.Join(s.Manager.Repo, "encryption/identity.age"), []byte("age-encryption.org/v1\nciphertext"), 0600)
}
func configuredRig(t *testing.T, s *Service) {
	t.Helper()
	enrolledRig(t, s)
	dir := filepath.Join(s.Manager.Config, "chezmoi")
	os.MkdirAll(dir, 0700)
	os.WriteFile(filepath.Join(dir, "key.txt"), []byte("disposable key"), 0600)
	text := "sourceDir = '" + s.Manager.Repo + "'\nencryption = 'age'\n[age]\nrecipient = 'age1test'\nidentity = '" + filepath.Join(dir, "key.txt") + "'\n[data]\nmachine = 'g14'\n"
	os.WriteFile(filepath.Join(dir, "chezmoi.toml"), []byte(text), 0600)
	hook := filepath.Join(s.Manager.Repo, ".git/hooks/pre-commit")
	os.MkdirAll(filepath.Dir(hook), 0700)
	os.WriteFile(hook, []byte("hook\n"), 0755)
}
func TestRigCheckIsReadOnlyAndOrdersFreshSetup(t *testing.T) {
	s := rigService(t)
	s.Lookup = func(cmd string) bool { return cmd != "chezmoi" }
	r, e := s.CheckRig(context.Background(), "")
	if e != nil {
		t.Fatal(e)
	}
	if r.Next(false) != "tools" || len(r.MissingTools) != 1 {
		t.Fatal(r)
	}
	if _, e = os.Lstat(s.Manager.State); !os.IsNotExist(e) {
		t.Fatal("check created state")
	}
	if _, e = os.Lstat(s.Manager.Config); !os.IsNotExist(e) {
		t.Fatal("check created config")
	}
	s.Lookup = func(string) bool { return true }
	r, e = s.CheckRig(context.Background(), "")
	if e != nil || r.Next(false) != "machine" {
		t.Fatal(r, e)
	}
	r, e = s.CheckRig(context.Background(), "g14")
	if e != nil || r.Next(false) != "enroll" {
		t.Fatal(r, e)
	}
}
func TestRigResumesRecoveryAndSkipsCurrentSharedConfig(t *testing.T) {
	s := rigService(t)
	enrolledRig(t, s)
	r, e := s.CheckRig(context.Background(), "desktop")
	if e != nil || r.Next(false) != "recover-age" {
		t.Fatal(r, e)
	}
	configuredRig(t, s)
	r, e = s.CheckRig(context.Background(), "")
	if e != nil || r.Machine != "g14" || r.Next(false) != "components" || r.Shared != "current" {
		t.Fatal(r, e)
	}
}
func TestRigStopsOnPartialEnrollmentAndForeignConfig(t *testing.T) {
	for _, kind := range []string{"partial", "foreign", "bad-key", "machine-conflict"} {
		t.Run(kind, func(t *testing.T) {
			s := rigService(t)
			machine := "g14"
			if kind == "partial" {
				enrolledRig(t, s)
				os.Remove(filepath.Join(s.Manager.Repo, "encryption/identity.age"))
			} else {
				configuredRig(t, s)
				switch kind {
				case "foreign":
					p := filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml")
					b, _ := os.ReadFile(p)
					os.WriteFile(p, []byte(strings.Replace(string(b), s.Manager.Repo, "/another/checkout", 1)), 0600)
				case "bad-key":
					os.Chmod(filepath.Join(s.Manager.Config, "chezmoi/key.txt"), 0644)
				case "machine-conflict":
					machine = "desktop"
				}
			}
			r, e := s.CheckRig(context.Background(), machine)
			if e != nil {
				t.Fatal(e)
			}
			if r.Next(false) != "blocked" || len(r.Issues) == 0 {
				t.Fatal(r)
			}
		})
	}
}
func TestHookConflictNeedsExplicitChoice(t *testing.T) {
	s := rigService(t)
	configuredRig(t, s)
	p := filepath.Join(s.Manager.Repo, ".git/hooks/pre-commit")
	os.WriteFile(p, []byte("custom hook"), 0755)
	r, e := s.CheckRig(context.Background(), "g14")
	if e != nil || r.Next(false) != "hooks" || r.Next(true) != "components" {
		t.Fatal(r, e)
	}
}
func TestSharedPackageProvenanceIsSupported(t *testing.T) {
	s := rigService(t)
	if e := s.RecordRequirements(context.Background(), "shared", []string{"chezmoi", "age"}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.packageRecords(); e != nil {
		t.Fatal(e)
	}
}
func TestSharedDeclineIsReportedAsSkipped(t *testing.T) {
	s := rigService(t)
	configuredRig(t, s)
	s.Manager.Run = func(ctx context.Context, args ...string) (string, error) {
		if args[0] == "git" {
			if args[3] == "config" {
				return "", errors.New("unset")
			}
			return filepath.Join(s.Manager.Repo, ".git/hooks/pre-commit"), nil
		}
		if args[0] == "age-keygen" {
			return "age1test", nil
		}
		if args[len(args)-1] == "relative" {
			return "", nil
		}
		if args[len(args)-1] == "verify" {
			return "", testExitStatus(1)
		}
		return " M .gitconfig", nil
	}
	applied := false
	s.Terminal = func(ctx context.Context, args ...string) error {
		if args[len(args)-1] == "--force" {
			applied = true
		}
		return nil
	}
	r, e := s.InitializeWithResult(context.Background(), "g14", false, func(string) bool { return false })
	if e != nil || r.Outcome != "skipped" || applied {
		t.Fatal(r, e)
	}
}
func TestCurrentSharedSetupDoesNotAskOrBackUp(t *testing.T) {
	s := rigService(t)
	configuredRig(t, s)
	s.Terminal = func(context.Context, ...string) error {
		t.Fatal("current files need no terminal operation")
		return nil
	}
	r, e := s.InitializeWithResult(context.Background(), "g14", false, func(string) bool { t.Fatal("current files need no confirmation"); return false })
	if e != nil || r.Outcome != "current" || r.Backup != "" {
		t.Fatal(r, e)
	}
}
func TestSyncRefusesDirtyCheckoutAndRequiresConfirmation(t *testing.T) {
	for _, dirty := range []bool{true, false} {
		s := rigService(t)
		s.Manager.Run = func(ctx context.Context, args ...string) (string, error) {
			if args[len(args)-1] == "--untracked-files=normal" {
				if dirty {
					return " M home/config", nil
				}
				return "", nil
			}
			return "main", nil
		}
		called := false
		s.Terminal = func(ctx context.Context, args ...string) error {
			called = true
			if args[len(args)-1] != "--ff-only" {
				t.Fatal(args)
			}
			return nil
		}
		r, e := s.PullSource(context.Background(), func(string) bool { return false })
		if dirty && e == nil {
			t.Fatal("dirty checkout accepted")
		}
		if !dirty && (e != nil || r.Outcome != "skipped") {
			t.Fatal(r, e)
		}
		if called {
			t.Fatal("pull ran without a clean checkout and confirmation")
		}
	}
}
func TestSetupActionCannotOverwriteExistingEnrollment(t *testing.T) {
	s := rigService(t)
	configuredRig(t, s)
	called := false
	s.Terminal = func(context.Context, ...string) error { called = true; return nil }
	r, e := s.PerformSetupStep(context.Background(), "enroll", "g14", false, func(string) bool { return true })
	if e != nil || r.Outcome != "current" || called {
		t.Fatal(r, e)
	}
}
func TestSharedManagedUsesUnambiguousNULPaths(t *testing.T) {
	s := rigService(t)
	s.Manager.Run = func(ctx context.Context, args ...string) (string, error) {
		hasNUL := false
		for _, a := range args {
			if a == "--nul-path-separator" {
				hasNUL = true
			}
		}
		if !hasNUL {
			t.Fatal("managed paths must use the NUL protocol")
		}
		if args[len(args)-3] == "dirs" {
			return "", nil
		}
		return ".gitconfig\x00filename\nwith-newline\x00", nil
	}
	files, _, e := s.sharedTargets(context.Background())
	if e != nil || len(files) != 2 || files[1] != "filename\nwith-newline" {
		t.Fatal(files, e)
	}
}
func TestSharedReviewRechecksEditsBeforeApplying(t *testing.T) {
	for _, kind := range []string{"target", "source"} {
		t.Run(kind, func(t *testing.T) {
			s := rigService(t)
			configuredRig(t, s)
			os.MkdirAll(filepath.Join(s.Manager.Repo, "home"), 0700)
			source := filepath.Join(s.Manager.Repo, "home/dot_gitconfig")
			target := filepath.Join(s.Manager.Home, ".gitconfig")
			os.WriteFile(source, []byte("repository"), 0644)
			os.WriteFile(target, []byte("original"), 0644)
			base := s.Manager.Run
			s.Manager.Run = func(ctx context.Context, args ...string) (string, error) {
				if args[0] != "chezmoi" {
					return base(ctx, args...)
				}
				if args[len(args)-1] == "relative" {
					if args[len(args)-3] == "dirs" {
						return "", nil
					}
					return ".gitconfig\x00", nil
				}
				if args[len(args)-1] == "verify" {
					return "", testExitStatus(1)
				}
				return " M .gitconfig", nil
			}
			applied := false
			s.Terminal = func(ctx context.Context, args ...string) error {
				if args[len(args)-1] == "diff" {
					path := target
					if kind == "source" {
						path = source
					}
					return os.WriteFile(path, []byte("late edit"), 0644)
				}
				if args[len(args)-1] == "--force" {
					applied = true
				}
				return nil
			}
			_, e := s.InitializeWithResult(context.Background(), "g14", false, func(string) bool { return true })
			if e == nil || applied {
				t.Fatal("applied a changed review", e)
			}
		})
	}
}

type testExitStatus int

func (e testExitStatus) Error() string { return "files differ" }
func (e testExitStatus) ExitCode() int { return int(e) }
func TestSyncReadinessOnlyIncludesManagedConfiguration(t *testing.T) {
	s := rigService(t)
	s.Lookup = func(string) bool { return false }
	r, e := s.CheckSyncRig(context.Background(), "")
	if e != nil || r.Shared != "unmanaged" || r.Next(false) != "components" || len(r.MissingTools) != 0 {
		t.Fatal(r, e)
	}
}
