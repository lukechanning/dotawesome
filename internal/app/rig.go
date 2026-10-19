package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dotawesome/internal/deploy"
	"github.com/pelletier/go-toml/v2"
)

var sharedTools = []struct{ command, pkg string }{{"git", "git"}, {"chezmoi", "chezmoi"}, {"age", "age"}, {"age-keygen", "age"}, {"gitleaks", "gitleaks"}, {"fish", "fish"}, {"alacritty", "alacritty"}, {"gh", "github-cli"}, {"aws", "aws-cli-v2"}}

type RigInspection struct {
	Managed                               []string
	Machine                               string
	MissingTools, Packages, Issues        []string
	Encryption, Shared, Hooks, HookDetail string
	Interrupted                           bool
}

func (r *RigInspection) Next(skipHooks bool) string {
	if len(r.Issues) > 0 {
		return "blocked"
	}
	if r.Interrupted {
		return "recover-files"
	}
	if r.Shared == "unmanaged" {
		return "components"
	}
	if len(r.MissingTools) > 0 {
		return "tools"
	}
	if r.Machine == "" {
		return "machine"
	}
	if r.Encryption != "ready" {
		return r.Encryption
	}
	if r.Hooks == "conflict" && !skipHooks {
		return "hooks"
	}
	if r.Shared != "current" || r.Hooks == "install" && !skipHooks {
		return "shared"
	}
	return "components"
}
func (s *Service) sharedCommand() []string {
	return []string{"chezmoi", "--source", s.Manager.Repo, "--destination", s.Manager.Home, "--config", filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml"), "--no-pager"}
}

type sharedConfig struct {
	Source     string `toml:"sourceDir"`
	Encryption string `toml:"encryption"`
	Age        struct{ Recipient, Identity string }
	Data       struct{ Machine string }
}

func (s *Service) readSharedConfig(machine, recipient string) (sharedConfig, error) {
	var c sharedConfig
	b, e := os.ReadFile(filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml"))
	if e != nil {
		return c, e
	}
	if e = toml.Unmarshal(b, &c); e != nil {
		return c, fmt.Errorf("invalid chezmoi configuration: %w", e)
	}
	if c.Data.Machine != "g14" && c.Data.Machine != "desktop" {
		return c, errors.New("existing chezmoi config has no valid machine")
	}
	if machine != "" && machine != c.Data.Machine {
		return c, errors.New("machine already initialized with a different selection; inspect local chezmoi config")
	}
	source := c.Source
	if strings.HasPrefix(source, "~/") {
		source = filepath.Join(s.Manager.Home, source[2:])
	}
	source, _ = filepath.Abs(source)
	if source != s.Manager.Repo {
		return c, errors.New("chezmoi sourceDir belongs to another checkout; existing config preserved")
	}
	identity := filepath.Join(s.Manager.Config, "chezmoi/key.txt")
	if c.Encryption != "age" || c.Age.Identity != identity || recipient != "" && c.Age.Recipient != recipient {
		return c, errors.New("local age configuration does not match enrollment; inspect it before continuing")
	}
	return c, nil
}
func (s *Service) hookStatus(ctx context.Context) (string, string, error) {
	if _, e := s.Manager.Run(ctx, "git", "-C", s.Manager.Repo, "config", "--get", "core.hooksPath"); e == nil {
		return "conflict", "A custom core.hooksPath is configured. Chain hooks/pre-commit manually before retaining it.", nil
	}
	path, e := s.Manager.Run(ctx, "git", "-C", s.Manager.Repo, "rev-parse", "--path-format=absolute", "--git-path", "hooks/pre-commit")
	if e != nil {
		return "", "", e
	}
	if !deploy.Exists(path) {
		return "install", "The repository secret-scanning hook will be installed.", nil
	}
	st, e := os.Lstat(path)
	if e != nil {
		return "", "", e
	}
	source, e := os.ReadFile(filepath.Join(s.Manager.Repo, "hooks/pre-commit"))
	if e != nil {
		return "", "", e
	}
	b, e := os.ReadFile(path)
	if e == nil && st.Mode().IsRegular() && string(source) == string(b) {
		return "ready", "Secret-scanning hook already installed.", nil
	}
	return "conflict", "An existing pre-commit hook is preserved. Chain hooks/pre-commit manually before retaining it.", nil
}
func (s *Service) sharedTargets(ctx context.Context) ([]string, []string, error) {
	return s.sharedTargetsFor(ctx, s.sharedCommand())
}
func (s *Service) sharedTargetsFor(ctx context.Context, command []string) ([]string, []string, error) {
	lists := [][]string{}
	for _, include := range []string{"files,symlinks", "dirs"} {
		args := append(append([]string{}, command...), "--no-tty", "managed", "--nul-path-separator", "--include", include, "--path-style", "relative")
		out, e := s.Manager.Run(ctx, args...)
		if e != nil {
			return nil, nil, e
		}
		list := []string{}
		if out != "" {
			if !strings.HasSuffix(out, "\x00") {
				return nil, nil, errors.New("invalid chezmoi managed-file terminator")
			}
			list = strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
		}

		lists = append(lists, list)
	}
	if e := CheckOwnership(lists[0], lists[1]); e != nil {
		return nil, nil, e
	}
	for _, path := range lists[0] {
		if e := deploy.SafePath(filepath.Join(s.Manager.Home, path)); e != nil {
			return nil, nil, e
		}
	}
	return lists[0], lists[1], nil
}

// CheckRig inspects local readiness without creating directories, decrypting, installing, or signing in.
func (s *Service) CheckRig(ctx context.Context, machine string) (*RigInspection, error) {
	r := &RigInspection{Machine: machine, Encryption: "enroll", Shared: "prepare", Hooks: "install"}
	if machine != "" && machine != "g14" && machine != "desktop" {
		r.Issues = append(r.Issues, "Choose G14 or desktop for shared configuration.")
	}
	if records, e := s.Manager.Records(); e != nil {
		r.Issues = append(r.Issues, e.Error())
	} else {
		for name := range records {
			r.Managed = append(r.Managed, name)
		}
		sort.Strings(r.Managed)
	}
	packages, e := s.packageRecords()
	if e != nil {
		r.Issues = append(r.Issues, e.Error())
	}
	for _, entry := range packages {
		if entry.Pending || entry.Removing {
			r.Issues = append(r.Issues, "An interrupted package operation needs review of packages.json and pacman logs.")
		}
	}
	r.Interrupted = deploy.Exists(filepath.Join(s.Manager.State, "transaction.json"))
	needed := map[string]bool{}
	for _, tool := range sharedTools {
		if !s.Lookup(tool.command) {
			r.MissingTools = append(r.MissingTools, tool.command)
			needed[tool.pkg] = true
		}
	}
	r.Packages = keys(needed)
	if s.Manager.Config != filepath.Join(s.Manager.Home, ".config") {
		r.Issues = append(r.Issues, "Shared configuration requires XDG_CONFIG_HOME=$HOME/.config.")
	}
	public := filepath.Join(s.Manager.Repo, "encryption/recipient.txt")
	recovery := filepath.Join(s.Manager.Repo, "encryption/identity.age")
	identity := filepath.Join(s.Manager.Config, "chezmoi/key.txt")
	hasPublic, hasRecovery, hasKey := deploy.Exists(public), deploy.Exists(recovery), deploy.Exists(identity)
	recipient := ""
	if hasPublic != hasRecovery || hasKey && !hasPublic {
		r.Encryption = "blocked"
		r.Issues = append(r.Issues, "Age enrollment is partial or missing repository recovery files. Preserve existing files and inspect encryption/ and chezmoi/key.txt.")
	} else if hasPublic {
		r.Encryption = "recover-age"
		pub, e := deploy.Inspect(public)
		if e != nil || pub == nil || pub.Link != nil {
			r.Issues = append(r.Issues, "The public recipient must be a readable regular file.")
		} else {
			recipient = strings.TrimSpace(string(pub.Bytes()))
			if !strings.HasPrefix(recipient, "age1") || len(strings.Fields(recipient)) != 1 {
				r.Issues = append(r.Issues, "The public age recipient is invalid; inspect enrollment.")
			}
		}
		cipher, e := deploy.Inspect(recovery)
		if e != nil || cipher == nil || cipher.Link != nil || !strings.HasPrefix(string(cipher.Bytes()), "age-encryption.org/v1") {
			r.Issues = append(r.Issues, "The encrypted recovery copy is invalid; preserve it and inspect enrollment.")
		}
		if hasKey && s.Lookup("age-keygen") {
			if e = s.ValidateIdentity(ctx, identity, recipient); e != nil {
				r.Issues = append(r.Issues, e.Error())
			} else {
				r.Encryption = "ready"
			}
		}
	}
	config := filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml")
	if deploy.Exists(config) {
		c, e := s.readSharedConfig(machine, recipient)
		if e != nil {
			r.Issues = append(r.Issues, e.Error())
		} else {
			r.Machine = c.Data.Machine
		}
	}
	if s.Lookup("git") {
		var e error
		r.Hooks, r.HookDetail, e = s.hookStatus(ctx)
		if e != nil {
			r.Issues = append(r.Issues, "Cannot inspect repository hooks: "+e.Error())
		}
	}
	if len(r.MissingTools) == 0 && len(r.Issues) == 0 && r.Encryption == "ready" && deploy.Exists(config) {
		work, err := os.MkdirTemp("", "dotawesome-rig-check-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(work)
		command := append(s.sharedCommand(), "--dry-run", "--cache", filepath.Join(work, "cache"), "--persistent-state", filepath.Join(work, "state.boltdb"), "--refresh-externals=never")
		if _, _, e = s.sharedTargetsFor(ctx, command); e != nil {
			r.Issues = append(r.Issues, e.Error())
		} else {
			current, e := s.sharedCurrent(ctx, command)
			if e != nil {
				r.Issues = append(r.Issues, "Cannot inspect shared configuration: "+e.Error())
			} else if current {
				r.Shared = "current"
			} else {
				r.Shared = "changes"
			}
		}
	}
	return r, nil
}

func SharedRequirements() []string {
	names := map[string]bool{}
	for _, tool := range sharedTools {
		names[tool.pkg] = true
	}
	return keys(names)
}

// Sync only touches configuration already managed on this machine.
func (s *Service) CheckSyncRig(ctx context.Context, machine string) (*RigInspection, error) {
	if deploy.Exists(filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml")) {
		return s.CheckRig(ctx, machine)
	}
	r := &RigInspection{Machine: machine, Shared: "unmanaged", Encryption: "unused"}
	records, e := s.Manager.Records()
	if e != nil {
		r.Issues = append(r.Issues, e.Error())
	}
	for name := range records {
		r.Managed = append(r.Managed, name)
	}
	sort.Strings(r.Managed)
	packages, e := s.packageRecords()
	if e != nil {
		r.Issues = append(r.Issues, e.Error())
	}
	for owner, p := range packages {
		if owner != "shared" && (p.Pending || p.Removing) {
			r.Issues = append(r.Issues, "Interrupted package operation: inspect packages.json and pacman logs.")
		}
	}
	r.Interrupted = deploy.Exists(filepath.Join(s.Manager.State, "transaction.json"))
	return r, nil
}
