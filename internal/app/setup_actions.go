package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SetupResult is the structured outcome returned across a terminal handoff.
// It contains no decrypted configuration or identity material.
type SetupResult struct {
	Step    string `json:"step"`
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
	Backup  string `json:"backup,omitempty"`
}

type SourceInspection struct{ Branch, Upstream, Issue string }

func (s *Service) CheckSource(ctx context.Context) SourceInspection {
	r := SourceInspection{}
	base := []string{"git", "-C", s.Manager.Repo}
	branch, e := s.Manager.Run(ctx, append(base, "symbolic-ref", "--short", "HEAD")...)
	if e != nil {
		r.Issue = "Choose a Git branch before syncing; detached checkouts are preserved."
		return r
	}
	r.Branch = branch
	out, e := s.Manager.Run(ctx, append(base, "status", "--porcelain=v1", "--untracked-files=normal")...)
	if e != nil {
		r.Issue = "Cannot inspect the checkout: " + e.Error()
		return r
	}
	if strings.TrimSpace(out) != "" {
		r.Issue = "This checkout has local changes. Commit or preserve them before syncing; nothing will be stashed or overwritten."
		return r
	}
	r.Upstream, e = s.Manager.Run(ctx, append(base, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")...)
	if e != nil {
		r.Issue = "This branch has no upstream. Configure its Git tracking branch before syncing."
	}
	return r
}
func (s *Service) PullSource(ctx context.Context, ask func(string) bool) (SetupResult, error) {
	r := SetupResult{Step: "pull"}
	e := s.Manager.WithLock(ctx, false, func() error {
		check := s.CheckSource(ctx)
		if check.Issue != "" {
			return errors.New(check.Issue)
		}
		if !ask("Pull the latest configuration with git pull --ff-only?") {
			r.Outcome = "skipped"
			r.Detail = "Source left unchanged; sync is still pending."
			return nil
		}
		if e := s.Terminal(ctx, "git", "-C", s.Manager.Repo, "pull", "--ff-only"); e != nil {
			return fmt.Errorf("pull did not complete; resolve the Git error before syncing: %w", e)
		}
		r.Outcome = "completed"
		r.Detail = "Source updated. Relaunch through bin/dotawesome before reviewing configuration."
		return nil
	})
	return r, e
}
func (s *Service) PerformSetupStep(ctx context.Context, action, machine string, skipHooks bool, ask func(string) bool) (SetupResult, error) {
	if action == "pull" {
		return s.PullSource(ctx, ask)
	}
	r := SetupResult{Step: action}
	if action == "github" || action == "aws" {
		if !ask("Sign in to " + action + " on this machine?") {
			r.Outcome = "skipped"
			r.Detail = "Optional account login is still pending."
			return r, nil
		}
		args := []string{"gh", "auth", "login"}
		if action == "aws" {
			args = []string{"aws", "login", "--profile", "cof-aws"}
		}
		if e := s.Terminal(ctx, args...); e != nil {
			return r, e
		}
		r.Outcome = "completed"
		r.Detail = "Login command finished; verify the account in the final check."
		return r, nil
	}
	check, e := s.CheckRig(ctx, machine)
	if e != nil {
		return r, e
	}
	if action == "recover-files" {
		if !ask("Recover the interrupted file transaction before setup?") {
			r.Outcome = "skipped"
			return r, nil
		}
		e = s.Manager.WithLock(ctx, true, func() error { return nil })
		r.Outcome = "completed"
		return r, e
	}
	if len(check.Issues) > 0 {
		return r, errors.New(strings.Join(check.Issues, "; "))
	}
	switch action {
	case "tools":
		if len(check.MissingTools) == 0 {
			r.Outcome = "current"
			r.Detail = "Required tools are already available."
			return r, nil
		}
		if !s.Lookup("pacman") {
			return r, errors.New("Automatic package setup requires Arch/CachyOS with pacman. Install the named prerequisites manually and check again.")
		}
		if !ask("Install missing shared tools with sudo pacman -Syu --needed (full system upgrade)?") {
			r.Outcome = "skipped"
			r.Detail = "Required tools are still missing."
			return r, nil
		}
		if e = s.InstallPackages(ctx, map[string][]string{"shared": SharedRequirements()}); e != nil {
			return r, e
		}
	case "enroll", "recover-age":
		if check.Encryption == "ready" {
			r.Outcome = "current"
			r.Detail = "Existing encryption identity reused."
			return r, nil
		}
		if check.Machine == "" {
			return r, errors.New("Choose this machine's profile before encryption setup.")
		}
		if action != check.Encryption {
			return r, errors.New("Encryption state changed; check this machine again before continuing.")
		}
		if !ask("Continue encryption setup in the local terminal?") {
			r.Outcome = "skipped"
			r.Detail = "Encryption setup remains pending."
			return r, nil
		}
		if action == "enroll" {
			e = s.Enroll(ctx)
		} else {
			recipient, e2 := os.ReadFile(filepath.Join(s.Manager.Repo, "encryption/recipient.txt"))
			if e2 != nil {
				return r, e2
			}
			e = s.Manager.WithLock(ctx, true, func() error {
				return s.RecoverIdentity(ctx, filepath.Join(s.Manager.Config, "chezmoi/key.txt"), filepath.Join(s.Manager.Repo, "encryption/identity.age"), strings.TrimSpace(string(recipient)))
			})
		}
		if e != nil {
			return r, e
		}
	case "shared":
		if check.Machine == "" || check.Encryption != "ready" || len(check.MissingTools) > 0 {
			return r, errors.New("Complete machine, tool and encryption setup before shared configuration.")
		}
		if !ask("Prepare chezmoi and review shared configuration for " + check.Machine + "?") {
			r.Outcome = "skipped"
			r.Detail = "Shared setup left unchanged."
			return r, nil
		}
		result, e := s.InitializeWithResult(ctx, check.Machine, skipHooks, ask)
		if e == nil && (result.Outcome == "current" || result.Outcome == "completed") {
			e = s.RecordRequirements(ctx, "shared", SharedRequirements())
		}
		return result, e
	default:
		return r, fmt.Errorf("unknown setup action: %s", action)
	}
	r.Outcome = "completed"
	r.Detail = "Step finished. Check this machine again to find the next step."
	if action == "enroll" {
		r.Detail = "Recovery verified. Commit encryption/recipient.txt and encryption/identity.age before another machine; keep the plaintext key private."
	}
	return r, nil
}

// WriteSetupReceipt publishes a private result file without replacing an existing path.
func WriteSetupReceipt(path string, result SetupResult) error {
	b, e := json.Marshal(result)
	if e != nil {
		return e
	}
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, writeErr := f.Write(append(b, '\n'))
	return errors.Join(writeErr, f.Close())
}

type AccountCheck struct{ Name, Action, State, Detail string }

func (s *Service) CheckAccounts(ctx context.Context) []AccountCheck {
	out := []AccountCheck{}
	for _, item := range []struct {
		name, action string
		args         []string
	}{{"GitHub", "github", []string{"gh", "auth", "status"}}, {"AWS", "aws", []string{"aws", "sts", "get-caller-identity", "--profile", "cof-tf"}}} {
		entry := AccountCheck{Name: item.name, Action: item.action, State: "pending", Detail: "Login was not verified; credentials or network may be unavailable."}
		if !s.Lookup(item.args[0]) {
			entry.Detail = "Install " + item.args[0] + " to sign in."
		} else {
			bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, e := s.Manager.Run(bounded, item.args...)
			cancel()
			if e == nil {
				entry.State = "ready"
				entry.Detail = "Account is authenticated."
			}
		}
		out = append(out, entry)
	}
	return out
}
