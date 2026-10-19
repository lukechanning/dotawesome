// Package deploy implements the existing JSON ledger and recoverable file transactions.
package deploy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"time"
)

type Value struct {
	Data string  `json:"data,omitempty"`
	Mode *uint32 `json:"mode,omitempty"`
	Link *string `json:"link,omitempty"`
}

func (v Value) MarshalJSON() ([]byte, error) {
	if v.Link != nil {
		return json.Marshal(struct {
			Link string `json:"link"`
		}{*v.Link})
	}
	return json.Marshal(struct {
		Data string  `json:"data"`
		Mode *uint32 `json:"mode"`
	}{v.Data, v.Mode})
}
func ReadStateJSON(path string, out any) error {
	st, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("state must be a regular file: %s", path)
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if !json.Valid(b) {
		return fmt.Errorf("malformed state %s; preserve it for recovery", path)
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(out); e != nil {
		return fmt.Errorf("malformed or unknown state %s: %w; preserve it for recovery", path, e)
	}
	if decoder.More() {
		return fmt.Errorf("trailing state data: %s", path)
	}
	return nil
}
func File(b []byte, mode uint32) *Value {
	return &Value{Data: base64.StdEncoding.EncodeToString(b), Mode: &mode}
}
func (v *Value) Bytes() []byte {
	if v == nil {
		return nil
	}
	if v.Link != nil {
		return []byte("symlink -> " + *v.Link + "\n")
	}
	b, _ := base64.StdEncoding.DecodeString(v.Data)
	return b
}
func (v *Value) validate() error {
	if v == nil {
		return nil
	}
	if v.Link != nil {
		if v.Mode != nil || v.Data != "" || *v.Link == "" {
			return errors.New("invalid symlink record")
		}
		return nil
	}
	if v.Mode == nil || *v.Mode > 07777 {
		return errors.New("missing or invalid file mode")
	}
	_, e := base64.StdEncoding.DecodeString(v.Data)
	return e
}

type Files map[string]*Value
type Entry struct {
	Original  *Value `json:"original"`
	Installed *Value `json:"installed"`
}
type Record struct {
	Snapshot   string           `json:"snapshot"`
	Profile    *string          `json:"profile_source"`
	Files      map[string]Entry `json:"files"`
	Validation string           `json:"native_validation"`
	Reload     *bool            `json:"reload_units,omitempty"`
}
type Records map[string]Record
type Change struct {
	Before *Value `json:"before"`
	After  *Value `json:"after"`
}
type Transaction struct {
	Records Records           `json:"records"`
	Changes map[string]Change `json:"changes"`
	Reload  *bool             `json:"reload_units,omitempty"`
}
type Plan struct {
	Component string
	Files     Files
	Before    Files
	Ledger    Records
	Changes   map[string]Change
	Live      bool
}
type Runner func(context.Context, ...string) (string, error)
type Manager struct {
	Repo, Home, Config, State, Build string
	Run                              Runner
	BeforeRecovery                   func() error
	WriteHook                        func(string, *Value) error
}

func New(repo, home string) *Manager {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	home, _ = filepath.Abs(home)
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	state := os.Getenv("XDG_STATE_HOME")
	if state == "" {
		state = filepath.Join(home, ".local/state")
	}
	return &Manager{Repo: repo, Home: home, Config: config, State: filepath.Join(state, "dotawesome"), Build: filepath.Join(repo, ".build/dotawesome")}
}
func ReadJSON(path string, out any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, out); e != nil {
		return fmt.Errorf("malformed state %s: %w; preserve this file for recovery", path, e)
	}
	return nil
}
func Exists(path string) bool { _, e := os.Lstat(path); return e == nil }
func Inspect(path string) (*Value, error) {
	st, e := os.Lstat(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if st.Mode()&os.ModeSymlink != 0 {
		link, e := os.Readlink(path)
		return &Value{Link: &link}, e
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	b, e := os.ReadFile(path)
	return File(b, uint32(st.Mode().Perm())|uint32(st.Mode()&os.ModeSetuid)>>12|uint32(st.Mode()&os.ModeSetgid)>>12|uint32(st.Mode()&os.ModeSticky)>>11), e
}
func SafePath(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("invalid destination: %s", path)
	}
	for p := filepath.Dir(path); ; p = filepath.Dir(p) {
		st, e := os.Lstat(p)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if e == nil && !st.IsDir() {
			return fmt.Errorf("symlink parent or not a directory: %s", p)
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return nil
}
func overlaps(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
func (m *Manager) validateRecords(r Records) error {
	if r == nil {
		return errors.New("null installed ledger; inspect installed.json")
	}
	for owner, record := range r {
		if owner != "niri" && owner != "nvim" && owner != "tmux" {
			return fmt.Errorf("unknown component %q in installed.json", owner)
		}
		if record.Files == nil {
			return fmt.Errorf("missing files for %s", owner)
		}
		for p, e := range record.Files {
			if !strings.HasPrefix(p, m.Home+"/") && !(m.Config != "" && strings.HasPrefix(p, m.Config+"/")) {
				return fmt.Errorf("record outside home/config ownership: %s", p)
			}
			if err := SafePath(p); err != nil {
				return err
			}
			if e.Installed == nil {
				return fmt.Errorf("missing installed record: %s", p)
			}
			if err := e.Installed.validate(); err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
			if err := e.Original.validate(); err != nil {
				return err
			}
			for other, otherRecord := range r {
				if other == owner {
					continue
				}
				for q := range otherRecord.Files {
					if overlaps(p, q) {
						return fmt.Errorf("conflicting ownership: %s, %s", p, q)
					}
				}
			}
		}
	}
	return nil
}
func (m *Manager) Records() (Records, error) {
	r := Records{}
	e := ReadStateJSON(filepath.Join(m.State, "installed.json"), &r)
	if e != nil && !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	if e = m.validateRecords(r); e != nil {
		return nil, e
	}
	return r, nil
}
func (m *Manager) Plan(component string, files Files) (*Plan, error) {
	if component != "tmux" && component != "nvim" && component != "niri" {
		return nil, fmt.Errorf("unknown component %s", component)
	}
	r, e := m.Records()
	if e != nil {
		return nil, e
	}
	p := &Plan{Component: component, Files: Files{}, Before: Files{}, Ledger: r, Changes: map[string]Change{}}
	for path, v := range files {
		if v == nil {
			return nil, errors.New("nil planned file")
		}
		if e = v.validate(); e != nil {
			return nil, e
		}
		if e = SafePath(path); e != nil {
			return nil, e
		}
		if !strings.HasPrefix(path, m.Home+"/") && !(m.Config != "" && strings.HasPrefix(path, m.Config+"/")) {
			return nil, fmt.Errorf("destination outside home/config: %s", path)
		}
		for q := range files {
			if path != q && overlaps(path, q) {
				return nil, fmt.Errorf("overlapping destinations: %s, %s", path, q)
			}
		}
		for owner, rec := range r {
			if owner != component {
				for q := range rec.Files {
					if overlaps(path, q) {
						return nil, fmt.Errorf("%s owned by %s", path, owner)
					}
				}
			}
		}
		before, e := Inspect(path)
		if e != nil {
			return nil, e
		}
		p.Before[path] = before
		copyValue := *v
		p.Files[path] = &copyValue
		if !reflect.DeepEqual(before, v) {
			p.Changes[path] = Change{before, v}
		}
	}
	for path, entry := range r[component].Files {
		current, e := Inspect(path)
		if e != nil {
			return nil, e
		}
		if !reflect.DeepEqual(current, entry.Installed) {
			return nil, fmt.Errorf("modified managed file: %s; preserve your edit and restore the recorded file before continuing", path)
		}
		p.Before[path] = current
		if _, ok := files[path]; !ok {
			p.Changes[path] = Change{current, entry.Original}
		}
	}
	return p, nil
}
func (m *Manager) Write(path string, v *Value) error {
	if e := SafePath(path); e != nil {
		return e
	}
	if e := v.validate(); e != nil {
		return e
	}
	if m.WriteHook != nil {
		if e := m.WriteHook(path, v); e != nil {
			return e
		}
	}
	if v == nil {
		e := os.Remove(path)
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		return e
	}
	if e := os.MkdirAll(filepath.Dir(path), 0755); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".dotawesome-")
	if e != nil {
		return e
	}
	temp := f.Name()
	defer os.Remove(temp)
	if v.Link != nil {
		f.Close()
		if e = os.Remove(temp); e == nil {
			e = os.Symlink(*v.Link, temp)
		}
	} else {
		_, e = f.Write(v.Bytes())
		if e == nil {
			mode := os.FileMode(*v.Mode & 0777)
			if *v.Mode&04000 != 0 {
				mode |= os.ModeSetuid
			}
			if *v.Mode&02000 != 0 {
				mode |= os.ModeSetgid
			}
			if *v.Mode&01000 != 0 {
				mode |= os.ModeSticky
			}
			e = f.Chmod(mode)
		}
		if e == nil {
			e = f.Sync()
		}
		e = errors.Join(e, f.Close())
	}
	if e != nil {
		return e
	}
	if e = os.Rename(temp, path); e != nil {
		return e
	}
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
func (m *Manager) SaveJSON(path string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return m.Write(path, File(append(b, '\n'), 0600))
}
func (m *Manager) WithLock(ctx context.Context, recover bool, fn func() error) error {
	if e := SafePath(filepath.Join(m.State, "lock")); e != nil {
		return e
	}
	if e := os.MkdirAll(m.State, 0700); e != nil {
		return e
	}
	fd, e := syscall.Open(filepath.Join(m.State, "lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return e
	}
	defer syscall.Close(fd)
	for {
		e = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if e == nil {
			break
		}
		if e != syscall.EWOULDBLOCK {
			return e
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	if _, e = m.Records(); e != nil {
		return e
	}
	if recover {
		if e = m.Recover(); e != nil {
			return e
		}
	}
	return fn()
}
func (m *Manager) reload(enabled *bool) error {
	if enabled == nil || *enabled {
		if m.Run == nil {
			return errors.New("systemctl runner required for recovery")
		}
		_, e := m.Run(context.Background(), "systemctl", "--user", "daemon-reload")
		return e
	}
	return nil
}
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (m *Manager) Recover() error {
	path := filepath.Join(m.State, "transaction.json")
	var tx Transaction
	if e := ReadStateJSON(path, &tx); e != nil {
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		return e
	}
	if tx.Changes == nil {
		return errors.New("malformed transaction.json: missing changes")
	}
	if e := m.validateRecords(tx.Records); e != nil {
		return e
	}
	for p, c := range tx.Changes {
		if !strings.HasPrefix(p, m.Home+"/") && !(m.Config != "" && strings.HasPrefix(p, m.Config+"/")) {
			return fmt.Errorf("transaction outside ownership: %s", p)
		}
		if e := c.Before.validate(); e != nil {
			return e
		}
		if e := c.After.validate(); e != nil {
			return e
		}
		current, e := Inspect(p)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(current, c.Before) && !reflect.DeepEqual(current, c.After) {
			return fmt.Errorf("interrupted transaction: unexpected edit at %s; keep transaction.json for recovery", p)
		}
	}
	if (tx.Reload == nil || *tx.Reload) && m.BeforeRecovery != nil {
		if e := m.BeforeRecovery(); e != nil {
			return e
		}
	}
	keys := sortedKeys(tx.Changes)
	for i := len(keys) - 1; i >= 0; i-- {
		p := keys[i]
		if e := m.Write(p, tx.Changes[p].Before); e != nil {
			return e
		}
	}
	if e := m.SaveJSON(filepath.Join(m.State, "installed.json"), tx.Records); e != nil {
		return e
	}
	if e := m.reload(tx.Reload); e != nil {
		return e
	}
	return os.Remove(path)
}
func (m *Manager) Apply(ctx context.Context, review *Plan, profile string, reload bool, guards ...func() error) error {
	return m.WithLock(ctx, !review.Live, func() error {
		fresh, e := m.Plan(review.Component, review.Files)
		if e != nil {
			return e
		}
		if !reflect.DeepEqual(fresh.Before, review.Before) || !reflect.DeepEqual(fresh.Ledger, review.Ledger) {
			return errors.New("inputs changed since review; inspect and confirm a fresh plan")
		}
		for _, guard := range guards {
			if e = guard(); e != nil {
				return e
			}
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		snapshot, e := m.Snapshot(review.Component, review.Files)
		if e != nil {
			return e
		}
		records := fresh.Ledger
		updated := Records{}
		for k, v := range records {
			updated[k] = v
		}
		entries := map[string]Entry{}
		for p, v := range review.Files {
			original := fresh.Before[p]
			if old, ok := records[review.Component].Files[p]; ok {
				original = old.Original
			}
			entries[p] = Entry{original, v}
		}
		if len(entries) == 0 {
			delete(updated, review.Component)
		} else {
			var source *string
			if profile != "" {
				source = &profile
			}
			updated[review.Component] = Record{snapshot, source, entries, "passed", &reload}
		}
		tx := Transaction{records, fresh.Changes, &reload}
		journal := filepath.Join(m.State, "transaction.json")
		if e = m.SaveJSON(journal, tx); e != nil {
			return e
		}
		order := sortedKeys(fresh.Changes)
		if review.Live {
			priority := func(p string) int {
				if strings.Contains(p, "/palettes/") {
					return 0
				}
				if strings.HasSuffix(p, "/niri/config.kdl") {
					return 2
				}
				return 1
			}
			sort.SliceStable(order, func(i, j int) bool { return priority(order[i]) < priority(order[j]) })
		}
		for _, p := range order {
			if e = ctx.Err(); e == nil {
				e = m.Write(p, fresh.Changes[p].After)
			}
			if e != nil {
				break
			}
		}
		if e == nil {
			e = m.reload(&reload)
		}
		if e == nil {
			e = m.SaveJSON(filepath.Join(m.State, "installed.json"), updated)
		}
		if e != nil {
			recovery := m.Recover()
			if recovery != nil {
				return fmt.Errorf("deployment failed: %w; rollback failed: %v; keep transaction.json", e, recovery)
			}
			return fmt.Errorf("deployment rolled back: %w", e)
		}
		return os.Remove(journal)
	})
}

// Snapshot manifests use legacy Value objects and immutable numbered content.
func (m *Manager) Snapshot(component string, files Files) (string, error) {
	b, e := json.Marshal(files)
	if e != nil {
		return "", e
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(b))
	target := filepath.Join(m.Build, component, digest)
	keys := sortedKeys(files)
	if Exists(target) {
		var manifest Files
		if e = ReadJSON(filepath.Join(target, "manifest.json"), &manifest); e != nil || !reflect.DeepEqual(manifest, files) {
			return "", fmt.Errorf("tampered snapshot manifest: %s", target)
		}
		for i, p := range keys {
			value, e := Inspect(filepath.Join(target, fmt.Sprint(i)))
			expected := File(files[p].Bytes(), 0444)
			if e != nil || !reflect.DeepEqual(value, expected) {
				return "", fmt.Errorf("tampered snapshot: %s", target)
			}
		}
		return target, nil
	}
	if e = os.MkdirAll(filepath.Dir(target), 0700); e != nil {
		return "", e
	}
	stage, e := os.MkdirTemp(filepath.Dir(target), ".snapshot-")
	if e != nil {
		return "", e
	}
	defer os.RemoveAll(stage)
	for i, p := range keys {
		if e = m.Write(filepath.Join(stage, fmt.Sprint(i)), File(files[p].Bytes(), 0444)); e != nil {
			return "", e
		}
	}
	if e = m.Write(filepath.Join(stage, "manifest.json"), File(append(b, '\n'), 0444)); e != nil {
		return "", e
	}
	if e = os.Rename(stage, target); e != nil {
		return "", e
	}
	return target, nil
}
