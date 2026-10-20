package app

import (
	"context"
	"dotawesome/internal/deploy"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type PackageRecord struct {
	Added    []string `json:"added"`
	Roots    []string `json:"roots"`
	Required []string `json:"required,omitempty"`
	Pending  bool     `json:"pending,omitempty"`
	Removing bool     `json:"removing,omitempty"`
}

func (p PackageRecord) MarshalJSON() ([]byte, error) {
	type wire PackageRecord
	if p.Added == nil {
		p.Added = []string{}
	}
	if p.Roots == nil {
		p.Roots = []string{}
	}
	return json.Marshal(wire(p))
}
func packageNames(out string) (map[string]bool, error) {
	r := map[string]bool{}
	if strings.TrimSpace(out) == "there is nothing to do" || strings.TrimSpace(out) == "" {
		return r, nil
	}
	for _, name := range strings.Split(strings.TrimSpace(out), "\n") {
		if !regexp.MustCompile(`^[a-zA-Z0-9@_+][a-zA-Z0-9@_.+:-]*$`).MatchString(name) {
			return nil, errors.New("unexpected pacman package-list output; operation refused")
		}
		r[name] = true
	}
	return r, nil
}
func (s *Service) packageRecords() (map[string]PackageRecord, error) {
	r := map[string]PackageRecord{}
	e := deploy.ReadStateJSON(filepath.Join(s.Manager.State, "packages.json"), &r)
	if errors.Is(e, os.ErrNotExist) {
		return r, nil
	}
	if e != nil {
		return nil, e
	}
	if r == nil {
		return nil, errors.New("null packages.json; inspect provenance before mutation")
	}
	for owner, v := range r {
		if owner != "niri" && owner != "tmux" && owner != "nvim" && owner != "shared" && owner != "syncthing" {
			return nil, fmt.Errorf("unknown package owner: %s", owner)
		}
		for _, list := range [][]string{v.Added, v.Roots, v.Required} {
			if _, e := packageNames(strings.Join(list, "\n")); e != nil {
				return nil, e
			}
		}
	}
	return r, nil
}
func keys(m map[string]bool) []string {
	r := []string{}
	for k, v := range m {
		if v {
			r = append(r, k)
		}
	}
	sort.Strings(r)
	return r
}
func set(v []string) map[string]bool {
	r := map[string]bool{}
	for _, x := range v {
		r[x] = true
	}
	return r
}
func (s *Service) installed(ctx context.Context) (map[string]bool, error) {
	out, e := s.Manager.Run(ctx, "pacman", "-Qq")
	if e != nil {
		return nil, e
	}
	return packageNames(out)
}
func (s *Service) PackagePlan(ctx context.Context, component string) ([]string, error) {
	r, e := s.packageRecords()
	if e != nil {
		return nil, e
	}
	entry := r[component]
	if entry.Pending || entry.Removing {
		return nil, errors.New("interrupted package operation: inspect packages.json and pacman logs")
	}
	if len(entry.Roots) == 0 {
		return nil, nil
	}
	installed, e := s.installed(ctx)
	if e != nil {
		return nil, e
	}
	protected := map[string]bool{}
	for owner, other := range r {
		if owner != component {
			for _, p := range other.Required {
				protected[p] = true
			}
		}
	}
	roots := []string{}
	for _, p := range entry.Roots {
		if installed[p] && !protected[p] {
			roots = append(roots, p)
		}
	}
	if len(roots) == 0 {
		return nil, nil
	}
	sort.Strings(roots)
	out, e := s.Manager.Run(ctx, append([]string{"pacman", "-Rsu", "--print-format", "%n"}, roots...)...)
	if e != nil {
		return nil, e
	}
	closure, e := packageNames(out)
	if e != nil {
		return nil, e
	}
	added := set(entry.Added)
	candidates := map[string]bool{}
	for p := range closure {
		if added[p] && installed[p] && !protected[p] {
			candidates[p] = true
		}
	}
	return keys(candidates), nil
}
func (s *Service) RecordRequirements(ctx context.Context, name string, required []string) error {
	return s.Manager.WithLock(ctx, true, func() error {
		r, e := s.packageRecords()
		if e != nil {
			return e
		}
		entry := r[name]
		entry.Required = required
		r[name] = entry
		return s.Manager.SaveJSON(filepath.Join(s.Manager.State, "packages.json"), r)
	})
}
func (s *Service) InstallPackages(ctx context.Context, requirements map[string][]string) error {
	return s.Manager.WithLock(ctx, true, func() error {
		r, e := s.packageRecords()
		if e != nil {
			return e
		}
		for _, entry := range r {
			if entry.Pending || entry.Removing {
				return errors.New("interrupted package operation: inspect packages.json and pacman logs")
			}
		}
		required := map[string]bool{}
		for _, names := range requirements {
			for _, name := range names {
				required[name] = true
			}
		}
		if _, e = packageNames(strings.Join(keys(required), "\n")); e != nil {
			return e
		}
		before, e := s.installed(ctx)
		if e != nil {
			return e
		}
		for owner, names := range requirements {
			entry := r[owner]
			entry.Required = names
			entry.Pending = true
			r[owner] = entry
		}
		path := filepath.Join(s.Manager.State, "packages.json")
		if e = s.Manager.SaveJSON(path, r); e != nil {
			return e
		}
		operation := s.Terminal(ctx, append([]string{"sudo", "pacman", "-Syu", "--needed"}, keys(required)...)...)
		after, e := s.installed(context.Background())
		if e != nil {
			return fmt.Errorf("package reconciliation failed; pending provenance retained: %w", e)
		}
		for owner, names := range requirements {
			entry := r[owner]
			added, roots := set(entry.Added), set(entry.Roots)
			req := set(names)
			for p := range after {
				if !before[p] {
					added[p] = true
					if req[p] {
						roots[p] = true
					}
				}
			}
			entry.Added = keys(added)
			entry.Roots = keys(roots)
			entry.Pending = false
			r[owner] = entry
		}
		if e = s.Manager.SaveJSON(path, r); e != nil {
			return e
		}
		if operation != nil {
			return fmt.Errorf("package update failed; additions recorded for optional cleanup: %w", operation)
		}
		return nil
	})
}
func (s *Service) RemovePackages(ctx context.Context, name string) ([]string, error) {
	var removed []string
	e := s.Manager.WithLock(ctx, true, func() error {
		records, e := s.Manager.Records()
		if e != nil {
			return e
		}
		if _, ok := records[name]; ok {
			return errors.New("remove the component before removing its packages")
		}
		candidates, e := s.PackagePlan(ctx, name)
		if e != nil || len(candidates) == 0 {
			return e
		}
		r, e := s.packageRecords()
		if e != nil {
			return e
		}
		entry := r[name]
		entry.Removing = true
		r[name] = entry
		path := filepath.Join(s.Manager.State, "packages.json")
		if e = s.Manager.SaveJSON(path, r); e != nil {
			return e
		}
		operation := s.Terminal(ctx, append([]string{"sudo", "pacman", "-Ru"}, candidates...)...)
		remaining, e := s.installed(context.Background())
		if e != nil {
			return e
		}
		added, roots := set(entry.Added), set(entry.Roots)
		for _, p := range candidates {
			if !remaining[p] {
				delete(added, p)
				delete(roots, p)
				removed = append(removed, p)
			}
		}
		entry.Added = keys(added)
		entry.Roots = keys(roots)
		entry.Removing = false
		r[name] = entry
		if e = s.Manager.SaveJSON(path, r); e != nil {
			return e
		}
		return operation
	})
	return removed, e
}
