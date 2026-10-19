package deploy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) *Manager {
	t.Helper()
	root := t.TempDir()
	return &Manager{Home: root, State: filepath.Join(root, "state"), Build: filepath.Join(root, "build"), Run: func(context.Context, ...string) (string, error) { return "", nil }}
}
func TestLegacyRecordRoundTripAndRestore(t *testing.T) {
	m := fixture(t)
	p := filepath.Join(m.Home, "config")
	os.Symlink("original-target", p)
	files := Files{p: File([]byte("new\n"), 0640)}
	plan, e := m.Plan("tmux", files)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Apply(context.Background(), plan, "", false); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(m.State, "installed.json"))
	if e != nil {
		t.Fatal(e)
	}
	var legacy map[string]struct {
		Files map[string]struct {
			Original  map[string]any `json:"original"`
			Installed map[string]any `json:"installed"`
		} `json:"files"`
	}
	if e = json.Unmarshal(b, &legacy); e != nil {
		t.Fatal(e)
	}
	if legacy["tmux"].Files[p].Original["link"] != "original-target" {
		t.Fatal(string(b))
	}
	plan, e = m.Plan("tmux", nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = m.Apply(context.Background(), plan, "", false); e != nil {
		t.Fatal(e)
	}
	link, e := os.Readlink(p)
	if e != nil || link != "original-target" {
		t.Fatal(link, e)
	}
}
func TestDriftAndChangedReviewAreRefused(t *testing.T) {
	for _, managed := range []bool{false, true} {
		t.Run(map[bool]string{false: "unmanaged", true: "managed"}[managed], func(t *testing.T) {
			m := fixture(t)
			p := filepath.Join(m.Home, "file")
			f := Files{p: File([]byte("source"), 0600)}
			plan, e := m.Plan("tmux", f)
			if e != nil {
				t.Fatal(e)
			}
			if managed {
				if e = m.Apply(context.Background(), plan, "", false); e != nil {
					t.Fatal(e)
				}
				plan, e = m.Plan("tmux", f)
				if e != nil {
					t.Fatal(e)
				}
			}
			os.WriteFile(p, []byte("edit"), 0600)
			if e = m.Apply(context.Background(), plan, "", false); e == nil {
				t.Fatal("overwrote edit")
			}
			b, _ := os.ReadFile(p)
			if string(b) != "edit" {
				t.Fatal(string(b))
			}
		})
	}
}
func TestRejectsMalformedLedgerAndSymlinkParents(t *testing.T) {
	m := fixture(t)
	os.MkdirAll(m.State, 0700)
	os.WriteFile(filepath.Join(m.State, "installed.json"), []byte(`{"tmux":{"files":{"/etc/passwd":{}}}}`), 0600)
	if _, e := m.Records(); e == nil {
		t.Fatal("accepted malformed state")
	}
	os.Remove(filepath.Join(m.State, "installed.json"))
	os.Symlink(t.TempDir(), filepath.Join(m.Home, "link"))
	if _, e := m.Plan("tmux", Files{filepath.Join(m.Home, "link", "file"): File([]byte("x"), 0600)}); e == nil {
		t.Fatal("followed parent symlink")
	}
}
func TestRecoveryPreservesUnexpectedEdits(t *testing.T) {
	m := fixture(t)
	p := filepath.Join(m.Home, "file")
	os.WriteFile(p, []byte("edit"), 0600)
	tx := Transaction{Records: Records{}, Changes: map[string]Change{p: {Before: nil, After: File([]byte("installed"), 0600)}}}
	os.MkdirAll(m.State, 0700)
	b, _ := json.Marshal(tx)
	os.WriteFile(filepath.Join(m.State, "transaction.json"), b, 0600)
	if e := m.WithLock(context.Background(), true, func() error { return nil }); e == nil || !strings.Contains(e.Error(), "unexpected edit") {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(m.State, "transaction.json")); e != nil {
		t.Fatal(e)
	}
}
func TestRollbackAndInterruptedRecovery(t *testing.T) {
	for _, rollbackFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "rollback", true: "rollback failure"}[rollbackFails], func(t *testing.T) {
			m := fixture(t)
			a := filepath.Join(m.Home, "a")
			b := filepath.Join(m.Home, "b")
			os.WriteFile(a, []byte("old"), 0644)
			plan, e := m.Plan("tmux", Files{a: File([]byte("new"), 0600), b: File([]byte("new b"), 0644)})
			if e != nil {
				t.Fatal(e)
			}
			m.WriteHook = func(p string, v *Value) error {
				if p == b && v != nil {
					return os.ErrPermission
				}
				if rollbackFails && p == a && string(v.Bytes()) == "old" {
					return os.ErrPermission
				}
				return nil
			}
			if e = m.Apply(context.Background(), plan, "", false); e == nil {
				t.Fatal("failure not returned")
			}
			if Exists(filepath.Join(m.State, "transaction.json")) != rollbackFails {
				t.Fatal("wrong journal lifecycle")
			}
			m.WriteHook = nil
			if e = m.WithLock(context.Background(), true, func() error { return nil }); e != nil {
				t.Fatal(e)
			}
			data, _ := os.ReadFile(a)
			if string(data) != "old" {
				t.Fatal(string(data))
			}
			if Exists(b) {
				t.Fatal("created file not rolled back")
			}
		})
	}
}
func TestDeterministicSnapshotsDetectTampering(t *testing.T) {
	m := fixture(t)
	files := Files{filepath.Join(m.Home, "a"): File([]byte("a"), 0644)}
	a, e := m.Snapshot("tmux", files)
	if e != nil {
		t.Fatal(e)
	}
	b, e := m.Snapshot("tmux", files)
	if e != nil || a != b {
		t.Fatal(a, b, e)
	}
	os.Chmod(filepath.Join(a, "0"), 0600)
	os.WriteFile(filepath.Join(a, "0"), []byte("tamper"), 0600)
	if _, e = m.Snapshot("tmux", files); e == nil {
		t.Fatal("accepted tamper")
	}
}
func TestOwnershipAcrossComponents(t *testing.T) {
	m := fixture(t)
	p := filepath.Join(m.Home, "a")
	plan, _ := m.Plan("tmux", Files{p: File([]byte("a"), 0644)})
	if e := m.Apply(context.Background(), plan, "", false); e != nil {
		t.Fatal(e)
	}
	if _, e := m.Plan("nvim", Files{p: File([]byte("b"), 0644)}); e == nil {
		t.Fatal("stole ownership")
	}
}
func TestPermissionBitsRoundTrip(t *testing.T) {
	m := fixture(t)
	for _, mode := range []uint32{0600, 0640, 0755, 04755} {
		p := filepath.Join(m.Home, "mode")
		if e := m.Write(p, File([]byte("x"), mode)); e != nil {
			t.Fatal(e)
		}
		v, e := Inspect(p)
		if e != nil || *v.Mode != mode {
			t.Fatalf("mode %o got %+v %v", mode, v, e)
		}
	}
}
func TestLegacyEmptyFileEncoding(t *testing.T) {
	b, _ := json.Marshal(File(nil, 0600))
	if !strings.Contains(string(b), `"data":""`) {
		t.Fatal(string(b))
	}
	target := "target"
	b, _ = json.Marshal(&Value{Link: &target})
	if string(b) != `{"link":"target"}` {
		t.Fatal(string(b))
	}
}
func TestUnknownLedgerFieldsRefuseMutation(t *testing.T) {
	m := fixture(t)
	os.MkdirAll(m.State, 0700)
	os.WriteFile(filepath.Join(m.State, "installed.json"), []byte(`{"tmux":{"files":{},"future_field":true}}`), 0600)
	if _, e := m.Records(); e == nil {
		t.Fatal("unknown state accepted")
	}
}
func TestLivePublishesPaletteBeforeConfig(t *testing.T) {
	m := fixture(t)
	palette := filepath.Join(m.Home, "palettes/new.json")
	config := filepath.Join(m.Home, "config.toml")
	plan, _ := m.Plan("niri", Files{palette: File([]byte("{}"), 0644), config: File([]byte("theme"), 0644)})
	plan.Live = true
	writes := []string{}
	m.WriteHook = func(p string, v *Value) error {
		if p == palette || p == config {
			writes = append(writes, p)
		}
		return nil
	}
	if e := m.Apply(context.Background(), plan, "generic", false); e != nil {
		t.Fatal(e)
	}
	if len(writes) != 2 || writes[0] != palette {
		t.Fatal(writes)
	}
}
func TestLockWaitCanBeCancelledWithoutWriting(t *testing.T) {
	m := fixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- m.WithLock(context.Background(), false, func() error { close(entered); <-release; return nil })
	}()
	<-entered
	cancel()
	called := false
	e := m.WithLock(ctx, false, func() error { called = true; return nil })
	close(release)
	if e != context.Canceled || called {
		t.Fatal(e, called)
	}
	if e = <-finished; e != nil {
		t.Fatal(e)
	}
}
