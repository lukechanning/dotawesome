package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"dotawesome/internal/deploy"
)

func (s *Service) sharedCurrent(ctx context.Context, command []string) (bool, error) {
	_, e := s.Manager.Run(ctx, append(append([]string{}, command...), "--no-tty", "verify")...)
	if e == nil {
		return true, nil
	}
	var exit interface{ ExitCode() int }
	if errors.As(e, &exit) && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, e
}

// A preview stamps both repository inputs and current destination entries.
func (s *Service) sharedReviewStamp(targets []string) (string, error) {
	values := map[string]*deploy.Value{}
	add := func(p string) error {
		v, e := deploy.Inspect(p)
		if e == nil {
			values[p] = v
		}
		return e
	}
	root := filepath.Join(s.Manager.Repo, "home")
	e := filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		return add(p)
	})
	if e != nil {
		return "", e
	}
	for _, p := range []string{filepath.Join(s.Manager.Repo, ".chezmoiroot"), filepath.Join(s.Manager.Repo, "encryption/recipient.txt"), filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml")} {
		if e = add(p); e != nil {
			return "", e
		}
	}
	for _, name := range targets {
		if e = add(filepath.Join(s.Manager.Home, name)); e != nil {
			return "", e
		}
	}
	b, e := json.Marshal(values)
	if e != nil {
		return "", e
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
