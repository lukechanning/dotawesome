package tui

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"dotawesome/internal/actions"
	"dotawesome/internal/app"
	"dotawesome/internal/deploy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMachineSetupStartsWithReadOnlyCheck(t *testing.T) {
	m := New(nil)
	m.screen = "more"
	m.cursor = 0
	next, cmd := m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "setup-check" || m.setup == nil || cmd == nil {
		t.Fatal(m.screen)
	}
	next, _ = m.Update(flowCheckMsg{report: &app.RigInspection{Encryption: "enroll", Shared: "prepare", Hooks: "install"}})
	m = next.(Model)
	if !strings.Contains(m.View().Content, "Choose this machine") {
		t.Fatal(m.View().Content)
	}
	next, _ = m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "setup-machine" {
		t.Fatal(m.screen)
	}
	next, _ = m.Update(key("enter"))
	if next.(Model).setup.Machine != "g14" {
		t.Fatal("machine choice lost")
	}
}
func TestAppConfigurationDoesNotRequireMachineSetup(t *testing.T) {
	m := New(nil)
	m.screen = "more"
	m.cursor = len(moreOptions) - 2
	next, _ := m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "components" || m.setup != nil {
		t.Fatal(m.screen)
	}
	m.cursor = 2
	next, _ = m.Update(key("enter"))
	next, _ = next.Update(key("enter"))
	m = next.(Model)
	if m.screen != "preflight" || !m.selected["tmux"] || m.setup != nil {
		t.Fatal(m.screen)
	}
}
func TestSyncPullsBeforeConfigurationInspection(t *testing.T) {
	m := New(&app.Service{Manager: deploy.New(t.TempDir(), t.TempDir())})
	m.nav = actions.Index("config sync")
	next, cmd := m.Update(key("enter"))
	m = next.(Model)
	if m.setup == nil || m.setup.Intent != "sync" || cmd == nil {
		t.Fatal(m.screen)
	}
	next, _ = m.Update(flowCheckMsg{source: &app.SourceInspection{Branch: "main", Upstream: "origin/main"}})
	m = next.(Model)
	if !strings.Contains(m.View().Content, "Pull latest configuration") {
		t.Fatal(m.View().Content)
	}
	next, cmd = m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "setup-handoff" || cmd == nil || m.setup.Action != "pull" {
		t.Fatal(m.screen)
	}
}
func TestReadySetupSkipsEnrollmentAndSharedSteps(t *testing.T) {
	m := New(nil)
	m.setup = &setupFlow{Intent: "setup", Machine: "g14", Report: &app.RigInspection{Machine: "g14", Encryption: "ready", Shared: "current", Hooks: "ready"}}
	m.screen = "setup"
	next, _ := m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "select" || m.selectionOrigin != "setup" {
		t.Fatal(m.screen)
	}
}
func TestDeclinedStepStaysPending(t *testing.T) {
	m := New(nil)
	m.setup = &setupFlow{Intent: "setup", Action: "shared"}
	m.screen = "setup-handoff"
	next, _ := m.Update(flowStepMsg{result: app.SetupResult{Step: "shared", Outcome: "skipped", Detail: "Shared files left unchanged."}})
	m = next.(Model)
	if m.screen != "setup-paused" || !strings.Contains(m.View().Content, "left unchanged") {
		t.Fatal(m.screen, m.View().Content)
	}
}
func TestDirtySyncCannotContinueToPull(t *testing.T) {
	m := New(nil)
	m.setup = &setupFlow{Intent: "sync", Source: &app.SourceInspection{Issue: "local changes"}}
	m.screen = "setup"
	next, _ := m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "setup-issues" {
		t.Fatal("dirty checkout reached pull")
	}
}
func TestFinishedFlowKeepsDeferredWorkVisible(t *testing.T) {
	m := New(nil)
	m.setup = &setupFlow{Intent: "setup", Report: &app.RigInspection{Shared: "current"}}
	m.screen = "execute"
	next, _ := m.Update(finishedMsg{results: []app.Result{{Component: "niri", State: "deferred", Detail: "log out of Niri"}}})
	m = next.(Model)
	if m.screen != "setup-finish" || !strings.Contains(m.View().Content, "log out of Niri") {
		t.Fatal(m.screen, m.View().Content)
	}
}
func TestSetupBackDoesNotAbandonActiveWrites(t *testing.T) {
	m := New(nil)
	m.setup = &setupFlow{}
	m.screen = "execute"
	cancelled := false
	m.cancel = func() { cancelled = true }
	next, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !cancelled || cmd != nil || next.(Model).screen != "execute" {
		t.Fatal("unsafe setup cancellation")
	}
}
func TestConfiguringMatchingFilesStillEnrollsOwnership(t *testing.T) {
	repo, _ := filepath.Abs("../..")
	home := t.TempDir()
	manager := deploy.New(repo, home)
	manager.Config = filepath.Join(home, ".config")
	manager.State = filepath.Join(home, "state")
	manager.Build = filepath.Join(home, "snapshots")
	manager.Run = func(context.Context, ...string) (string, error) { return "", nil }
	s := &app.Service{Manager: manager, Lookup: func(string) bool { return true }}
	content, e := os.ReadFile(filepath.Join(repo, "components/tmux/tmux.conf"))
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(home, ".tmux.conf"), content, 0644)
	review, e := s.Inspect(context.Background(), "tmux", "", false)
	if e != nil || len(review.Plan.Changes) != 0 {
		t.Fatal(review, e)
	}
	m := New(s)
	m.reviews = []*app.Inspection{review}
	cmd := m.start()
	batch := cmd().(tea.BatchMsg)
	batch[0]()
	records, e := manager.Records()
	if e != nil || records["tmux"].Files == nil {
		t.Fatal("matching files never became managed", e)
	}
}
func TestSyncSelectsOnlyManagedApps(t *testing.T) {
	m := New(nil)
	m.setup = &setupFlow{Intent: "sync", Pulled: true}
	m.screen = "setup-check"
	next, _ := m.Update(flowCheckMsg{report: &app.RigInspection{Shared: "unmanaged", Managed: []string{"tmux"}}})
	m = next.(Model)
	if !m.selected["tmux"] || m.selected["niri"] || m.setupNext() != "components" {
		t.Fatal(m.selected, m.setupNext())
	}
}
