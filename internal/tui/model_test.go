package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"dotawesome/internal/actions"
	"dotawesome/internal/app"
	"dotawesome/internal/deploy"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
}
func TestDashboardRendersImmediatelyAndResizes(t *testing.T) {
	m := New(nil)
	if !strings.Contains(m.View().Content, "Inspecting") {
		t.Fatal(m.View().Content)
	}
	next, _ := m.Update(tea.WindowSizeMsg{Width: 35, Height: 18})
	m = next.(Model)
	if !strings.Contains(m.View().Content, "DOTAWESOME") {
		t.Fatal(m.View().Content)
	}
}
func TestCompletionStaysOpenAndReturnsToDashboard(t *testing.T) {
	m := New(nil)
	m.screen = "execute"
	m.started = time.Now()
	next, cmd := m.Update(finishedMsg{results: []app.Result{{Component: "tmux", State: "completed"}}})
	m = next.(Model)
	if cmd != nil || m.screen != "results" || !strings.Contains(m.View().Content, "completed") {
		t.Fatal(m.screen)
	}
	next, _ = m.Update(key("esc"))
	if next.(Model).screen != "overview" {
		t.Fatal("did not return")
	}
}
func TestCancellationRequestsSafeStop(t *testing.T) {
	m := New(nil)
	m.screen = "execute"
	cancelled := false
	m.cancel = func() { cancelled = true }
	next, cmd := m.Update(key("q"))
	if cmd != nil || cancelled {
		t.Fatal("q abandoned operation")
	}
	next, cmd = next.Update(key("esc"))
	if !cancelled || cmd != nil || next.(Model).screen != "execute" {
		t.Fatal("unsafe cancellation")
	}
}
func TestNavigationUsesSavedMachineAndComponents(t *testing.T) {
	home := t.TempDir()
	mngr := deploy.New("", home)
	mngr.State = filepath.Join(home, "state")
	os.MkdirAll(mngr.State, 0700)
	mngr.SaveJSON(filepath.Join(mngr.State, "preferences.json"), Preferences{Machine: "g14", Components: []string{"tmux"}})
	m := New(&app.Service{Manager: mngr})
	if m.profile != "g14" || !m.selected["tmux"] || m.selected["niri"] {
		t.Fatal(m.profile, m.selected)
	}
	next, _ := m.Update(statusMsg{results: []app.Result{{Component: "tmux", State: "current"}}})
	m = next.(Model)
	if m.nav != actions.Index("config sync") {
		t.Fatal("managed machine did not recommend sync")
	}
}
func TestComponentsEnterPreservesSelectedRow(t *testing.T) {
	m := New(nil)
	m.screen = "components"
	m.cursor = 2
	next, _ := m.Update(key("enter"))
	got := next.(Model)
	if got.screen != "component" || got.component != "tmux" {
		t.Fatal(got.screen, got.component)
	}
	next, _ = got.Update(key("enter"))
	got = next.(Model)
	if !got.selected["tmux"] || got.selected["niri"] {
		t.Fatal(got.selected)
	}
}
func TestGuidedHomeShowsPurposeAndOnlyMainChoices(t *testing.T) {
	m := New(nil)
	m.loading = false
	m.status = []app.Result{{Component: "niri", State: "inactive", Detail: "/private/path"}}
	text := m.View().Content
	for _, want := range []string{"What would you like to do?", "Syncthing Setup", "Syncthing Status", "Config Setup", "Config Sync", "App Setup", "Troubleshoot or Restore", "review changes"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	for _, hidden := range []string{"inactive", "/private/path", "enroll", "AWS", "[Overview]", "Configure an app", "More options"} {
		if strings.Contains(text, hidden) {
			t.Fatalf("home exposes %q:\n%s", hidden, text)
		}
	}
	next, _ := m.Update(key("enter"))
	if next.(Model).screen != "external-confirm" {
		t.Fatal(next.(Model).screen)
	}
}
func TestHomeMaintenanceIsInsideMoreOptions(t *testing.T) {
	m := New(nil)
	next, _ := m.Update(key("e"))
	if next.(Model).screen != "overview" {
		t.Fatal("home launched advanced action")
	}
	m.nav = actions.Index("troubleshoot")
	next, _ = m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "more" {
		t.Fatal(m.screen)
	}
	m.cursor = 3
	next, _ = m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "external-confirm" || m.externalAction != "enroll-age" {
		t.Fatal(m.screen, m.externalAction)
	}
	if !strings.Contains(m.View().Content, "recovery passphrase") {
		t.Fatal(m.View().Content)
	}
	next, _ = m.Update(key("esc"))
	if next.(Model).screen != "more" {
		t.Fatal("back lost menu context")
	}
}
func TestStatusExplanationsDescribeConfigurationNotAppHealth(t *testing.T) {
	for _, tc := range []struct{ state, label, explanation string }{{"inactive", "Not managed yet", "not changed"}, {"current", "Configuration matches", "repository"}, {"source-changed", "Repository changes available", "review"}, {"locally-modified", "Local edits found", "protect"}, {"mystery", "Unable to check", "check"}} {
		m := New(nil)
		m.screen = "component"
		m.component = "nvim"
		m.loading = false
		m.status = []app.Result{{Component: "nvim", State: tc.state}}
		text := m.View().Content
		for _, want := range []string{tc.label, tc.explanation} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q: %s", tc.state, want, text)
			}
		}
	}
}
func TestRecoveryBecomesTheNextStep(t *testing.T) {
	m := New(nil)
	m.loading = false
	m.status = []app.Result{{State: "recovery pending"}}
	m.screen = "more"
	m.cursor = 0
	if !strings.Contains(m.homeSummary(), "An update was interrupted") {
		t.Fatal(m.View().Content)
	}
	next, _ := m.Update(key("enter"))
	got := next.(Model)
	if got.screen != "setup-check" || got.setup == nil {
		t.Fatal(got.screen, got.externalAction)
	}
}
func TestUnavailableSourceNeverLooksCurrent(t *testing.T) {
	m := New(nil)
	m.screen = "component"
	m.component = "nvim"
	m.status = []app.Result{{Component: "nvim", State: "current", Detail: "source unavailable: missing source"}}
	text := m.View().Content
	if strings.Contains(text, "Configuration matches") || !strings.Contains(text, "Unable to check") {
		t.Fatal(text)
	}
}
func TestEditedFilesCanBeInspectedWithoutDeploying(t *testing.T) {
	home := t.TempDir()
	manager := deploy.New("", home)
	manager.Config = filepath.Join(home, ".config")
	manager.State = filepath.Join(home, "state")
	path := filepath.Join(home, ".tmux.conf")
	os.WriteFile(path, []byte("edited\n"), 0644)
	saved := deploy.File([]byte("saved\n"), 0644)
	os.MkdirAll(manager.State, 0700)
	manager.SaveJSON(filepath.Join(manager.State, "installed.json"), deploy.Records{"tmux": {Files: map[string]deploy.Entry{path: {Installed: saved}}}})
	m := New(&app.Service{Manager: manager})
	m.screen = "component"
	m.component = "tmux"
	m.loading = false
	m.status = []app.Result{{Component: "tmux", State: "locally-modified"}}
	next, cmd := m.Update(key("enter"))
	m = next.(Model)
	if m.screen != "differences" || cmd == nil {
		t.Fatal(m.screen)
	}
	next, _ = m.Update(cmd())
	text := next.(Model).View().Content
	if !strings.Contains(text, "-saved") || !strings.Contains(text, "+edited") {
		t.Fatal(text)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "edited\n" {
		t.Fatal("inspection modified local edits")
	}
}
func TestBackKeepsReviewAndParentContext(t *testing.T) {
	m := New(nil)
	m.screen = "confirm"
	m.files = []string{"example"}
	m.diffs["example"] = "reviewed content"
	m.showDiff()
	next, _ := m.Update(key("esc"))
	m = next.(Model)
	if m.screen != "review" || !strings.Contains(m.view.View(), "reviewed content") {
		t.Fatal(m.screen)
	}
	m.component = "tmux"
	m.screen = "technical"
	next, _ = m.Update(key("esc"))
	if next.(Model).screen != "component" {
		t.Fatal(next.(Model).screen)
	}
}
func TestLateInspectionCannotReopenWorkflow(t *testing.T) {
	m := New(nil)
	m.screen = "preflight"
	next, _ := m.Update(key("esc"))
	m = next.(Model)
	next, _ = m.Update(inspectedMsg{})
	if next.(Model).screen == "review" {
		t.Fatal("late result reopened cancelled review")
	}
}
func TestMaintenanceActionsStayReachable(t *testing.T) {
	for i, option := range moreOptions {
		m := New(nil)
		m.screen = "more"
		m.cursor = i
		next, _ := m.Update(key("enter"))
		got := next.(Model)
		switch option.action {
		case "components":
			if got.screen != "components" {
				t.Fatal(got.screen)
			}
		case "restore":
			if got.screen != "select" || !got.restore {
				t.Fatal(option, got.screen)
			}
		case "init", "full-setup":
			if got.screen != "setup-check" || got.setup == nil {
				t.Fatal(got.screen)
			}
		case "help":
			if got.screen != "help" {
				t.Fatal(got.screen)
			}
		case "back", "refresh":
			if got.screen != "overview" {
				t.Fatal(got.screen)
			}
		default:
			if got.screen != "external-confirm" || got.externalAction != option.action {
				t.Fatal(option, got.externalAction)
			}
		}
	}
}
func TestReviewPutsSelectedDiffBeforeRepeatedFileLists(t *testing.T) {
	m := New(nil)
	m.screen = "review"
	m.reviews = []*app.Inspection{{Plan: &deploy.Plan{Component: "tmux", Before: deploy.Files{"/home/example/.tmux.conf": nil}, Changes: map[string]deploy.Change{"/home/example/.tmux.conf": {After: deploy.File([]byte("set -g mouse on\n"), 0644)}}}}}
	m.prepareDiffs()
	if strings.Contains(m.view.View(), "managed:") || !strings.Contains(m.view.View(), "+set -g mouse on") {
		t.Fatal(m.view.View())
	}
}
func TestHelpStillAllowsQuittingNavigation(t *testing.T) {
	m := New(nil)
	next, _ := m.Update(key("?"))
	_, cmd := next.Update(key("q"))
	if cmd == nil {
		t.Fatal("help trapped the quit key")
	}
}
func TestSmallScreensKeepSelectionAndFooterVisible(t *testing.T) {
	for _, screen := range []string{"overview", "more", "component"} {
		m := New(nil)
		m.width = 40
		m.height = 18
		m.mono = true
		m.loading = false
		m.screen = screen
		m.component = "nvim"
		m.status = []app.Result{{Component: "nvim", State: "locally-modified"}}
		m.resize()
		if screen == "more" {
			m.cursor = 7
		}
		text := m.View().Content
		if lipgloss.Width(text) > 40 || lipgloss.Height(text) > 18 {
			t.Fatalf("%s overflow: %dx%d\\n%s", screen, lipgloss.Width(text), lipgloss.Height(text), text)
		}
		if !strings.Contains(text, ">") || !strings.Contains(text, "Enter") {
			t.Fatal(text)
		}
	}
}

func TestProjectLogoStaysAboveTheGuidedInterface(t *testing.T) {
	for _, screen := range []string{"overview", "review", "results"} {
		m := New(nil)
		m.width = 100
		m.height = 35
		m.screen = screen
		m.mono = true
		m.resize()
		text := m.View().Content
		if !strings.HasPrefix(text, "  ____   ___ _____") {
			t.Fatalf("%s lost project logo:\n%s", screen, text)
		}
		if lipgloss.Height(text) > m.height {
			t.Fatalf("%s overflowed after adding logo", screen)
		}
	}
	m := New(nil)
	m.width = 40
	m.height = 18
	m.mono = true
	m.resize()
	text := m.View().Content
	if strings.TrimSpace(strings.Split(text, "\n")[0]) != "DOTAWESOME" || strings.Contains(text, "  ____") {
		t.Fatal("small terminal did not retain compact title")
	}
}

func TestDirectActionsAndNamedApp(t *testing.T) {
	for i, a := range actions.All {
		m := New(nil)
		m.nav = i
		next, _ := m.Update(key("enter"))
		got := next.(Model)
		switch a.Command {
		case "syncthing setup", "syncthing status", "config setup":
			if got.screen != "external-confirm" || got.externalAction != a.Command {
				t.Fatal(a, got.screen)
			}
		case "app setup":
			if got.screen != "select" || got.setup != nil {
				t.Fatal(got.screen)
			}
		case "troubleshoot":
			if got.screen != "more" {
				t.Fatal(got.screen)
			}
		}
	}
	m := newAppModel(nil, "tmux")
	if m.screen != "preflight" || !m.selected["tmux"] || m.selected["niri"] || m.setup != nil {
		t.Fatal(m.screen, m.selected)
	}
}
func TestActionMenuKeyboardWrapAndCancel(t *testing.T) {
	m := New(nil)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = next.(Model)
	if m.nav != len(actions.All)-1 {
		t.Fatal(m.nav)
	}
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(Model)
	if m.nav != 0 {
		t.Fatal(m.nav)
	}
	next, _ = m.Update(key("enter"))
	next, _ = next.Update(key("esc"))
	if next.(Model).screen != "overview" {
		t.Fatal(next.(Model).screen)
	}
}

func TestEveryActionRemainsVisibleInNarrowTerminal(t *testing.T) {
	for i, a := range actions.All {
		m := New(nil)
		m.loading = false
		m.nav = i
		m.width = 35
		m.height = 18
		text := m.View().Content
		if !strings.Contains(text, strings.Fields(a.Label)[0]) || !strings.Contains(text, "> ") {
			t.Fatal(a.Label, text)
		}
		for _, line := range strings.Split(text, "\n") {
			if lipgloss.Width(line) > 35 {
				t.Fatal("overflow", line)
			}
		}
	}
	for _, name := range []string{"niri", "nvim", "tmux"} {
		m := newAppModel(nil, name)
		if !m.selected[name] || m.screen != "preflight" || m.startup == nil {
			t.Fatal(name, m.screen)
		}
	}
}

func TestTerminalFailureRetainsStructuredExplanation(t *testing.T) {
	m := New(nil)
	m.externalAction = "syncthing setup"
	next, _ := m.Update(handoffMsg{err: fmt.Errorf("exit status 1"), result: app.SetupResult{Outcome: "failed", Detail: "tailscaled service startup failed; inspect journalctl -u tailscaled"}})
	text := next.(Model).View().Content
	if !strings.Contains(text, "tailscaled service startup failed") || !strings.Contains(text, "journalctl") {
		t.Fatal(text)
	}
}
