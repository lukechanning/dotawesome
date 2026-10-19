package app

import (
	"context"
	"dotawesome/internal/deploy"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var componentTargets = []string{".tmux.conf", ".config/nvim", ".config/niri", ".config/dotawesome", ".config/systemd/user/dotawesome-niri-noctalia.service", ".config/systemd/user/niri.service.d", ".config/xdg-desktop-portal/niri-portals.conf", ".config/vivaldi-stable.conf", ".config/signal-desktop-flags.conf"}

func CheckOwnership(files, dirs []string) error {
	for i, list := range [][]string{files, dirs} {
		for _, p := range list {
			if filepath.IsAbs(p) || filepath.Clean(p) != p || p == ".." || strings.HasPrefix(p, "../") {
				return fmt.Errorf("unsafe shared target: %s", p)
			}
			for _, owned := range componentTargets {
				if p == owned || strings.HasPrefix(p, owned+"/") || i == 0 && strings.HasPrefix(owned, p+"/") {
					return fmt.Errorf("chezmoi/component ownership overlap: %s", p)
				}
			}
		}
	}
	return nil
}
func (s *Service) ValidateIdentity(ctx context.Context, p, recipient string) error {
	st, e := os.Lstat(p)
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() || st.Mode().Perm() != 0600 || st.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return errors.New("identity must be a regular file with mode 0600")
	}
	out, e := s.Manager.Run(ctx, "age-keygen", "-y", p)
	if e != nil || strings.TrimSpace(out) != recipient {
		return errors.New("invalid identity or recipient mismatch; existing identity preserved")
	}
	return nil
}
func publish(source, target string) error {
	if e := deploy.SafePath(target); e != nil {
		return e
	}
	return os.Link(source, target)
}
func (s *Service) RecoverIdentity(ctx context.Context, identity, recovery, recipient string) error {
	if deploy.Exists(identity) {
		return s.ValidateIdentity(ctx, identity, recipient)
	}
	if e := deploy.SafePath(identity); e != nil {
		return e
	}
	if e := os.MkdirAll(filepath.Dir(identity), 0700); e != nil {
		return e
	}
	work, e := os.MkdirTemp(filepath.Dir(identity), ".identity-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(work)
	key := filepath.Join(work, "key")
	if e = s.Terminal(ctx, "age", "--decrypt", "--output", key, recovery); e != nil {
		return e
	}
	if e = os.Chmod(key, 0600); e != nil {
		return e
	}
	if e = s.ValidateIdentity(ctx, key, recipient); e != nil {
		return e
	}
	return publish(key, identity)
}
func (s *Service) Enroll(ctx context.Context) error {
	return s.Manager.WithLock(ctx, true, func() error {
		if _, e := s.packageRecords(); e != nil {
			return e
		}
		return s.enroll(ctx)
	})
}
func (s *Service) enroll(ctx context.Context) error {
	directory := filepath.Join(s.Manager.Config, "chezmoi")
	identity := filepath.Join(directory, "key.txt")
	public := filepath.Join(s.Manager.Repo, "encryption/recipient.txt")
	recovery := filepath.Join(s.Manager.Repo, "encryption/identity.age")
	for _, p := range []string{identity, public, recovery} {
		if deploy.Exists(p) {
			return errors.New("enrollment already exists or is partial; inspect it manually; nothing overwritten")
		}
		if e := deploy.SafePath(p); e != nil {
			return e
		}
	}
	if e := os.MkdirAll(directory, 0700); e != nil {
		return e
	}
	if e := os.MkdirAll(filepath.Dir(public), 0700); e != nil {
		return e
	}
	work, e := os.MkdirTemp(directory, ".enroll-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(work)
	key := filepath.Join(work, "key.txt")
	if _, e = s.Manager.Run(ctx, "age-keygen", "-o", key); e != nil {
		return e
	}
	if e = os.Chmod(key, 0600); e != nil {
		return e
	}
	recipient, e := s.Manager.Run(ctx, "age-keygen", "-y", key)
	if e != nil {
		return e
	}
	recipient = strings.TrimSpace(recipient)
	if e = s.ValidateIdentity(ctx, key, recipient); e != nil {
		return e
	}
	encrypted := filepath.Join(work, "identity.age")
	fmt.Println("Choose a recovery passphrase; store it separately in an offline recovery record.")
	if e = s.Terminal(ctx, "age", "--passphrase", "--output", encrypted, key); e != nil {
		return e
	}
	verified := filepath.Join(work, "verified.txt")
	fmt.Println("Re-enter the passphrase to verify recovery before enrollment is published.")
	if e = s.Terminal(ctx, "age", "--decrypt", "--output", verified, encrypted); e != nil {
		return e
	}
	if e = os.Chmod(verified, 0600); e != nil {
		return e
	}
	if e = s.ValidateIdentity(ctx, verified, recipient); e != nil {
		return e
	}
	stage, e := os.MkdirTemp(filepath.Dir(public), ".enroll-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(stage)
	cipher, e := os.ReadFile(encrypted)
	if e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(stage, "identity.age"), cipher, 0600); e != nil {
		return e
	}
	if e = os.WriteFile(filepath.Join(stage, "recipient.txt"), []byte(recipient+"\n"), 0600); e != nil {
		return e
	}
	if e = publish(filepath.Join(stage, "identity.age"), recovery); e != nil {
		return e
	}
	if e = publish(filepath.Join(stage, "recipient.txt"), public); e != nil {
		return fmt.Errorf("partial enrollment; inspect encryption directory: %w", e)
	}
	if e = publish(key, identity); e != nil {
		return fmt.Errorf("partial enrollment; inspect identity and encryption directory: %w", e)
	}
	fmt.Println("Recovery verified. Commit only encryption/recipient.txt and encryption/identity.age.")
	return nil
}
func (s *Service) ScanSecrets(ctx context.Context, staged bool) error {
	if !s.Lookup("gitleaks") {
		return errors.New("gitleaks is required; install explicitly: sudo pacman -S gitleaks")
	}
	base := []string{"gitleaks", "--redact=100", "--no-banner", "--no-color", "--ignore-gitleaks-allow", "--config", filepath.Join(s.Manager.Repo, ".gitleaks.toml")}
	if staged {
		return s.Terminal(ctx, append(base, "git", "--pre-commit", "--staged", s.Manager.Repo)...)
	}
	out, e := s.Manager.Run(ctx, "git", "-C", s.Manager.Repo, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if e != nil {
		return e
	}
	work, e := os.MkdirTemp("", "dotawesome-secret-scan-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(work)
	for _, name := range strings.Split(out, "\x00") {
		if name == "" {
			continue
		}
		if filepath.IsAbs(name) || filepath.Clean(name) != name || strings.HasPrefix(name, "../") {
			return errors.New("unsafe tracked path")
		}
		src := filepath.Join(s.Manager.Repo, name)
		st, e := os.Lstat(src)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return e
		}
		if !st.Mode().IsRegular() {
			continue
		}
		b, e := os.ReadFile(src)
		if e != nil {
			return e
		}
		dst := filepath.Join(work, name)
		if e = os.MkdirAll(filepath.Dir(dst), 0700); e != nil {
			return e
		}
		if e = os.WriteFile(dst, b, 0600); e != nil {
			return e
		}
	}
	return s.Terminal(ctx, append(base, "dir", work)...)
}
func (s *Service) InstallHooks(ctx context.Context) error {
	if _, e := s.Manager.Run(ctx, "git", "-C", s.Manager.Repo, "config", "--get", "core.hooksPath"); e == nil {
		return errors.New("core.hooksPath already configured; chain hooks/pre-commit manually or use --skip-hooks")
	}
	path, e := s.Manager.Run(ctx, "git", "-C", s.Manager.Repo, "rev-parse", "--path-format=absolute", "--git-path", "hooks/pre-commit")
	if e != nil {
		return e
	}
	source, e := os.ReadFile(filepath.Join(s.Manager.Repo, "hooks/pre-commit"))
	if e != nil {
		return e
	}
	if deploy.Exists(path) {
		st, e := os.Lstat(path)
		if e != nil {
			return e
		}
		b, _ := os.ReadFile(path)
		if st.Mode().IsRegular() && string(b) == string(source) {
			return nil
		}
		return errors.New("existing pre-commit hook preserved; chain hooks/pre-commit manually or use --skip-hooks")
	}
	if e = os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0755)
	if e != nil {
		return e
	}
	_, e = f.Write(source)
	return errors.Join(e, f.Close())
}

// Initialize delegates encrypted content and passphrase handling to chezmoi and age.
func (s *Service) Initialize(ctx context.Context, machine string, skipHooks bool, ask func(string) bool) error {
	_, e := s.InitializeWithResult(ctx, machine, skipHooks, ask)
	return e
}
func (s *Service) InitializeWithResult(ctx context.Context, machine string, skipHooks bool, ask func(string) bool) (SetupResult, error) {
	result := SetupResult{Step: "shared"}
	e := s.Manager.WithLock(ctx, true, func() error {
		if _, e := s.packageRecords(); e != nil {
			return e
		}
		return s.initialize(ctx, machine, skipHooks, ask, &result)
	})
	return result, e
}
func (s *Service) initialize(ctx context.Context, machine string, skipHooks bool, ask func(string) bool, result *SetupResult) error {
	for _, tool := range []string{"git", "chezmoi", "age", "age-keygen", "gitleaks"} {
		if !s.Lookup(tool) {
			return fmt.Errorf("missing %s; install explicitly: sudo pacman -S git chezmoi age gitleaks", tool)
		}
	}
	if s.Manager.Config != filepath.Join(s.Manager.Home, ".config") {
		return errors.New("shared configuration requires XDG_CONFIG_HOME=$HOME/.config")
	}
	recipientBytes, e := os.ReadFile(filepath.Join(s.Manager.Repo, "encryption/recipient.txt"))
	if e != nil {
		return errors.New("age enrollment missing; run bin/enroll-age locally first")
	}
	recipient := strings.TrimSpace(string(recipientBytes))
	recovery := filepath.Join(s.Manager.Repo, "encryption/identity.age")
	if !deploy.Exists(recovery) {
		return errors.New("encrypted recovery copy missing; inspect enrollment")
	}
	config := filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml")
	cm := []string{"chezmoi", "--source", s.Manager.Repo, "--destination", s.Manager.Home, "--config", config, "--no-pager"}
	if !deploy.Exists(config) {
		if machine != "g14" && machine != "desktop" {
			return errors.New("--machine g14 or desktop is required for first initialization")
		}
		if e = os.MkdirAll(filepath.Dir(config), 0700); e != nil {
			return e
		}
		if e = s.Terminal(ctx, append(cm, "init", "--promptChoice", "Machine="+machine)...); e != nil {
			return e
		}
	}
	if _, e = s.readSharedConfig(machine, recipient); e != nil {
		return e
	}
	identity := filepath.Join(filepath.Dir(config), "key.txt")
	if !skipHooks {
		if e = s.InstallHooks(ctx); e != nil {
			return e
		}
	}
	if e = s.RecoverIdentity(ctx, identity, recovery, recipient); e != nil {
		return e
	}
	targets, _, e := s.sharedTargets(ctx)
	if e != nil {
		return e
	}
	current, e := s.sharedCurrent(ctx, cm)
	if e != nil {
		return e
	}
	if current {
		result.Outcome = "current"
		result.Detail = "Shared configuration already matches the repository."
		return nil
	}
	reviewed, e := s.sharedReviewStamp(targets)
	if e != nil {
		return e
	}
	fmt.Println("Preview shared configuration (may display decrypted encrypted targets):")
	if e = s.Terminal(ctx, append(cm, "diff")...); e != nil {
		return e
	}
	if !ask("Apply shared configuration?") {
		result.Outcome = "skipped"
		result.Detail = "Shared files left unchanged; setup is not complete."
		return nil
	}
	freshTargets, _, e := s.sharedTargets(ctx)
	if e != nil {
		return e
	}
	fresh, e := s.sharedReviewStamp(freshTargets)
	if e != nil {
		return e
	}
	if fresh != reviewed {
		return errors.New("shared inputs changed since review; preserve edits and inspect a fresh preview before applying")
	}
	state := filepath.Join(s.Manager.State, "adoption")
	if e = os.MkdirAll(state, 0700); e != nil {
		return e
	}
	backup, e := os.MkdirTemp(state, "shared-")
	if e != nil {
		return e
	}
	manifest := []string{}
	for _, name := range targets {
		p := filepath.Join(s.Manager.Home, name)
		v, e := deploy.Inspect(p)
		if e != nil {
			return e
		}
		if v == nil {
			continue
		}
		if v.Link == nil {
			v = deploy.File(v.Bytes(), 0600)
		}
		if e = s.Manager.Write(filepath.Join(backup, name), v); e != nil {
			return e
		}
		manifest = append(manifest, name)
	}
	if e = s.Manager.SaveJSON(filepath.Join(backup, "manifest.json"), manifest); e != nil {
		return e
	}
	if e = s.Terminal(ctx, append(cm, "apply", "--force")...); e != nil {
		return fmt.Errorf("shared apply failed; adoption backup: %s: %w", backup, e)
	}
	result.Outcome = "completed"
	result.Detail = "Shared configuration applied."
	result.Backup = backup
	fmt.Println("Shared configuration applied. Adoption backup:", backup)
	return nil
}

// Command prepares a process for Bubble Tea's ExecProcess terminal handoff.
func Command(args ...string) *exec.Cmd { return exec.Command(args[0], args[1:]...) }
