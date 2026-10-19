package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"dotawesome/internal/app"
	"dotawesome/internal/deploy"
)

type setupFlow struct {
	Intent, Machine, Action                        string
	SkipHooks, Pulled, RestartRequired, Incomplete bool
	Report                                         *app.RigInspection
	Source                                         *app.SourceInspection
	Results                                        []app.Result
	Problem                                        string
	Accounts                                       []app.AccountCheck
}
type flowCheckMsg struct {
	report *app.RigInspection
	source *app.SourceInspection
	err    error
}
type flowStepMsg struct {
	result app.SetupResult
	err    error
}
type accountChecksMsg []app.AccountCheck
type syncRestartMsg struct{ err error }

func (m *Model) beginSetup(intent string, pulled bool) tea.Cmd {
	m.setup = &setupFlow{Intent: intent, Pulled: pulled}
	if m.profile == "g14" || m.profile == "desktop" {
		m.setup.Machine = m.profile
	}
	if pulled {
		m.setup.Results = []app.Result{{Component: "source", State: "completed", Detail: "Pulled latest Git configuration and reopened updated code."}}
	}
	m.err = nil
	m.cursor = 0
	return m.checkSetup()
}
func (m *Model) checkSetup() tea.Cmd {
	m.screen = "setup-check"
	m.loading = true
	m.cursor = 0
	s := m.s
	intent, machine, pulled := m.setup.Intent, m.setup.Machine, m.setup.Pulled
	return func() tea.Msg {
		if s == nil {
			return flowCheckMsg{err: fmt.Errorf("configuration service is unavailable")}
		}
		if intent == "sync" && !pulled {
			source := s.CheckSource(context.Background())
			return flowCheckMsg{source: &source}
		}
		var r *app.RigInspection
		var e error
		if intent == "sync" {
			r, e = s.CheckSyncRig(context.Background(), machine)
		} else {
			r, e = s.CheckRig(context.Background(), machine)
		}
		return flowCheckMsg{report: r, err: e}
	}
}
func (m Model) setupNext() string {
	f := m.setup
	if f.RestartRequired {
		return "restart"
	}
	if f.Problem != "" {
		return "blocked"
	}
	if f.Intent == "sync" && !f.Pulled {
		if f.Source == nil || f.Source.Issue != "" {
			return "blocked"
		}
		return "pull"
	}
	if f.Report == nil {
		return "blocked"
	}
	return f.Report.Next(f.SkipHooks)
}
func setupStepName(step string) string {
	switch step {
	case "tools":
		return "Install required tools"
	case "machine":
		return "Choose this machine"
	case "enroll":
		return "Create encryption recovery"
	case "recover-age":
		return "Recover this machine's key"
	case "shared":
		return "Review shared configuration"
	case "hooks":
		return "Keep your existing hooks"
	case "recover-files":
		return "Recover interrupted update"
	case "pull":
		return "Pull latest configuration"
	case "restart":
		return "Open updated sync flow"
	case "components":
		return "Choose app configuration"
	case "blocked":
		return "Inspect setup issues"
	}
	return step
}
func (m Model) setupExplanation(step string) string {
	switch step {
	case "tools":
		return "Install the missing shared tools: " + strings.Join(m.setup.Report.MissingTools, ", ") + ".\nThis runs sudo pacman -Syu --needed, including a full system upgrade."
	case "machine":
		return "Select G14 or desktop so shared templates use the right machine settings."
	case "enroll":
		return "Create a key to protect private configuration and a recovery copy for your other machines. Choose a recovery passphrase and save it separately."
	case "recover-age":
		return "Unlock your existing encryption key on this machine using your recovery passphrase."
	case "shared":
		return "Review shared Git, shell, terminal and AWS settings. Confirm in the terminal to save backups and apply changes."
	case "hooks":
		return m.setup.Report.HookDetail + "\nNothing will replace your existing hooks."
	case "recover-files":
		return "An unfinished file deployment must be recovered before setup continues. Edits made after interruption are protected."
	case "pull":
		return "Get the latest configuration from Git, then reopen Dotawesome to review changes. Local repository edits must be committed or resolved first."
	case "components":
		if m.setup.Intent == "sync" {
			return "Review the apps already managed on this machine. Shared files are included only when chezmoi is configured here."
		}
		if m.setup.Intent == "shared" {
			return "Shared setup is ready. Continue to the results."
		}
		return "Shared setup is ready. Choose Niri, Neovim or tmux, review their changes and confirm before deployment."
	case "restart":
		return "Source is updated. Reopen through bin/dotawesome so review and deployment use the updated code."
	default:
		return "Resolve the reported issue, then check this machine again."
	}
}
func (m *Model) continueSetup() tea.Cmd {
	next := m.setupNext()
	switch next {
	case "blocked":
		m.screen = "setup-issues"
		text := m.setup.Problem
		if m.setup.Report != nil {
			text += "\n" + strings.Join(m.setup.Report.Issues, "\n\n")
		}
		if m.setup.Source != nil {
			text += "\n" + m.setup.Source.Issue
		}
		m.view.SetContent(strings.TrimSpace(text))
		m.view.GotoTop()
	case "machine":
		m.screen = "setup-machine"
		m.cursor = 0
	case "hooks":
		m.screen = "setup-hooks"
		m.cursor = 0
	case "components":
		if m.setup.Intent == "sync" && len(m.setup.Report.Managed) == 0 {
			m.finishSetup([]app.Result{{Component: "apps", State: "skipped", Detail: "No apps are managed here. Choose Set up this machine to add apps."}})
			return nil
		}
		if m.setup.Intent == "shared" {
			m.finishSetup(nil)
		} else {
			m.screen = "select"
			m.selectionOrigin = "setup"
			m.restore = false
			m.cursor = 0
		}
	case "restart":
		return m.restartSync()
	default:
		m.setup.Action = next
		m.screen = "setup-handoff"
		m.cursor = 0
		return m.stepCommand()
	}
	return nil
}
func (m *Model) finishSetup(results []app.Result) {
	m.setup.Results = append(m.setup.Results, results...)
	m.results = m.setup.Results
	m.screen = "setup-finish"
	m.cursor = 0
	m.resize()
	m.showResults()
}
func (m Model) stepCommand() tea.Cmd {
	action, machine, skipHooks := m.setup.Action, m.setup.Machine, m.setup.SkipHooks
	exe, e := os.Executable()
	if e != nil {
		return func() tea.Msg { return flowStepMsg{err: e} }
	}
	work, e := os.MkdirTemp("", "dotawesome-setup-result-")
	if e != nil {
		return func() tea.Msg { return flowStepMsg{err: e} }
	}
	receipt := filepath.Join(work, "result.json")
	args := []string{"setup-step", "--action", action, "--receipt", receipt}
	if machine != "" {
		args = append(args, "--machine", machine)
	}
	if skipHooks {
		args = append(args, "--skip-hooks")
	}
	c := exec.Command(exe, args...)
	c.Env = append(os.Environ(), "DOTAWESOME_ROOT="+m.s.Manager.Repo)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.RemoveAll(work)
		var result app.SetupResult
		if e := deploy.ReadStateJSON(receipt, &result); e != nil {
			if err == nil {
				err = fmt.Errorf("setup command returned no usable outcome: %w", e)
			}
		}
		if result.Step != action || (result.Outcome != "completed" && result.Outcome != "current" && result.Outcome != "skipped" && result.Outcome != "failed") {
			err = fmt.Errorf("setup result does not match the requested step")
		}
		return flowStepMsg{result, err}
	})
}
func (m *Model) restartSync() tea.Cmd {
	m.screen = "setup-restarting"
	c := exec.Command(filepath.Join(m.s.Manager.Repo, "bin/dotawesome"), "sync", "--after-pull")
	c.Env = os.Environ()
	return tea.ExecProcess(c, func(e error) tea.Msg { return syncRestartMsg{e} })
}
func (m Model) checkAccounts() tea.Cmd {
	s := m.s
	return func() tea.Msg { return accountChecksMsg(s.CheckAccounts(context.Background())) }
}
func (m Model) setupBody() (string, bool) {
	if m.setup == nil {
		return "", false
	}
	f := m.setup
	switch m.screen {
	case "setup-check":
		return m.spin.View() + " Checking this machine…\nNo configuration is changed during this check.\n", true
	case "setup":
		var b strings.Builder
		title := "Set up this machine"
		if f.Intent == "sync" {
			title = "Sync this machine"
		}
		if f.Intent == "shared" {
			title = "Set up shared configuration"
		}
		b.WriteString(title + "\n\n")
		if f.Source != nil && !f.Pulled {
			b.WriteString("Branch: " + f.Source.Branch + "\nUpstream: " + f.Source.Upstream + "\n\n")
		} else if f.Report != nil {
			r := f.Report
			tools := "Ready"
			if len(r.MissingTools) > 0 {
				tools = "Missing " + strings.Join(r.MissingTools, ", ")
			}
			machine := r.Machine
			if machine == "" {
				machine = "Choose G14 or desktop"
			}
			encryption := "Key ready"
			if r.Encryption == "unused" {
				encryption = "Not configured here"
			}
			switch r.Encryption {
			case "enroll":
				encryption = "First enrollment needed"
			case "recover-age":
				encryption = "Existing key needs recovery"
			case "blocked":
				encryption = "Needs attention"
			}
			shared := "Setup needed"
			if r.Shared == "unmanaged" {
				shared = "Not managed here"
			}
			if r.Shared == "current" {
				shared = "Matches repository"
			} else if r.Shared == "changes" {
				shared = "Changes available"
			}
			fmt.Fprintf(&b, "Tools       %s\nMachine     %s\nEncryption  %s\nShared files  %s\n\n", tools, machine, encryption, shared)
		}
		next := m.setupNext()
		b.WriteString(m.setupExplanation(next) + "\n\n")
		b.WriteString(m.menu([]menuOption{{label: setupStepName(next)}, {label: "Check this machine again"}, {label: "Back home"}}, m.cursor))
		return b.String(), true
	case "setup-machine":
		return "Which machine are you setting up?\nExisting valid selections are reused automatically.\n\n" + m.menu([]menuOption{{"G14 laptop", "Use the laptop's shared machine settings.", ""}, {"Desktop", "Use the desktop's shared machine settings.", ""}}, m.cursor), true
	case "setup-hooks":
		return "Existing hooks need your decision\n\n" + m.setupExplanation("hooks") + "\n\n" + m.menu([]menuOption{{"I have chained hooks/pre-commit; keep existing hooks", "This preserves your current setup and skips installing Dotawesome's hook.", ""}, {label: "Back"}}, m.cursor), true
	case "setup-issues":
		return "Setup needs attention\nResolve these issues, then go back and check again.\n\n" + m.view.View() + "\n", true
	case "setup-handoff":
		return "Setup command is using the terminal…\n", true
	case "setup-restarting":
		return "Reopening the updated sync flow…\n", true
	case "setup-paused":
		options := []menuOption{{label: "Check and try again"}, {label: "Continue with apps only"}, {label: "Back home"}}
		if f.Action == "pull" || f.RestartRequired {
			options = []menuOption{{label: "Try again"}, {label: "Back home"}}
		}
		return "Setup is paused\n\n" + f.Problem + "\n\n" + m.menu(options, m.cursor), true
	case "setup-finish":
		title := "Selected configuration setup finished"
		if f.Incomplete {
			title = "Setup has remaining work"
		}
		for _, r := range f.Results {
			if r.State == "failed" || r.State == "deferred" || r.State == "skipped" && r.Component == "shared" {
				title = "Setup has remaining work"
			}
		}
		return title + "\nResults stay here until you continue.\n\n" + m.menu([]menuOption{{label: "Check account logins (optional)"}, {label: "Back home"}}, m.cursor) + "\n" + m.view.View() + "\nGraphical session and hardware checks remain manual.\n", true
	case "setup-accounts":
		var b strings.Builder
		b.WriteString("Machine-local accounts (optional)\n\n")
		for _, a := range f.Accounts {
			b.WriteString(a.Name + ": " + a.Detail + "\n")
		}
		b.WriteString("\n" + m.menu([]menuOption{{label: "Sign in to GitHub"}, {label: "Sign in to AWS"}, {label: "Back to setup results"}}, m.cursor))
		return b.String(), true
	}
	return "", false
}
func (m Model) updateSetup(msg tea.Msg) (Model, tea.Cmd, bool) {
	switch v := msg.(type) {
	case flowCheckMsg:
		if m.setup == nil || m.screen != "setup-check" {
			return m, nil, true
		}
		m.loading = false
		m.setup.Report = v.report
		m.setup.Source = v.source
		m.setup.Problem = ""
		if v.err != nil {
			m.setup.Problem = v.err.Error()
		}
		if v.report != nil {
			m.setup.Machine = v.report.Machine
			if m.setup.Intent == "sync" {
				m.selected = map[string]bool{}
				for _, name := range v.report.Managed {
					m.selected[name] = true
				}
			}
		}
		m.screen = "setup"
		m.cursor = 0
		return m, nil, true
	case flowStepMsg:
		if m.setup == nil {
			return m, nil, true
		}
		f := m.setup
		state := v.result.Outcome
		if v.err != nil {
			state = "failed"
		}
		detail := v.result.Detail
		if v.err != nil {
			detail = v.err.Error()
		}
		f.record(app.Result{Component: f.Action, State: state, Detail: detail, Backup: v.result.Backup})
		if state == "failed" || state == "skipped" {
			f.Problem = detail
			if f.Problem == "" {
				f.Problem = "This step was left unchanged and remains pending."
			}
			m.screen = "setup-paused"
			m.cursor = 0
			return m, nil, true
		}
		if f.Action == "pull" {
			f.RestartRequired = true
			cmd := m.restartSync()
			return m, cmd, true
		}
		if f.Action == "github" || f.Action == "aws" {
			return m, m.checkAccounts(), true
		}
		cmd := m.checkSetup()
		return m, cmd, true
	case syncRestartMsg:
		if v.err == nil {
			return m, tea.Quit, true
		}
		m.setup.Problem = "The updated app could not start. Fix the build error before applying configuration: " + v.err.Error()
		m.screen = "setup-paused"
		m.cursor = 0
		return m, nil, true
	case accountChecksMsg:
		if m.setup != nil {
			m.setup.Accounts = []app.AccountCheck(v)
			m.screen = "setup-accounts"
			m.cursor = 0
		}
		return m, nil, true
	}
	if m.setup == nil || !strings.HasPrefix(m.screen, "setup") {
		return m, nil, false
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil, false
	}
	k := key.String()
	if k == "esc" || k == "?" || k == "q" || k == "ctrl+c" || m.help {
		return m, nil, false
	}
	count := 3
	switch m.screen {
	case "setup-machine", "setup-hooks", "setup-finish":
		count = 2
	case "setup-paused":
		if m.setup.Action == "pull" || m.setup.RestartRequired {
			count = 2
		}
	case "setup-check", "setup-handoff", "setup-restarting":
		return m, nil, true
	case "setup-issues":
		var cmd tea.Cmd
		m.view, cmd = m.view.Update(msg)
		return m, cmd, true
	}
	if k == "up" || k == "shift+tab" {
		m.cursor = (m.cursor + count - 1) % count
		return m, nil, true
	}
	if k == "down" || k == "tab" {
		m.cursor = (m.cursor + 1) % count
		return m, nil, true
	}
	if k == "pgup" || k == "pgdown" {
		var cmd tea.Cmd
		m.view, cmd = m.view.Update(msg)
		return m, cmd, true
	}
	if k != "enter" {
		return m, nil, true
	}
	switch m.screen {
	case "setup":
		switch m.cursor {
		case 0:
			cmd := m.continueSetup()
			return m, cmd, true
		case 1:
			cmd := m.checkSetup()
			return m, cmd, true
		case 2:
			m.setup = nil
			m.screen = "overview"
			m.nav = 0
		}
	case "setup-machine":
		m.setup.Machine = []string{"g14", "desktop"}[m.cursor]
		if m.profile == "generic" || m.profile == "g14" || m.profile == "desktop" {
			m.profile = m.setup.Machine
		}
		cmd := m.checkSetup()
		return m, cmd, true
	case "setup-hooks":
		if m.cursor == 0 {
			m.setup.SkipHooks = true
		}
		m.screen = "setup"
		m.cursor = 0
	case "setup-paused":
		if m.cursor == 0 {
			if m.setup.RestartRequired {
				cmd := m.restartSync()
				return m, cmd, true
			}
			cmd := m.checkSetup()
			return m, cmd, true
		}
		if m.cursor == 1 && count == 3 {
			m.setup.Incomplete = true
			m.screen = "select"
			m.selectionOrigin = "setup"
			m.restore = false
			m.cursor = 0
		} else {
			m.setup = nil
			m.screen = "overview"
			m.nav = 0
		}
	case "setup-finish":
		if m.cursor == 0 {
			return m, m.checkAccounts(), true
		}
		m.setup = nil
		m.screen = "overview"
		m.nav = 0
		m.loading = true
		return m, m.inspectStatus(), true
	case "setup-accounts":
		if m.cursor < 2 {
			m.setup.Action = []string{"github", "aws"}[m.cursor]
			m.screen = "setup-handoff"
			return m, m.stepCommand(), true
		}
		m.screen = "setup-finish"
		m.cursor = 0
	}
	return m, nil, true
}

func (f *setupFlow) record(result app.Result) {
	for i, old := range f.Results {
		if old.Component == result.Component {
			f.Results[i] = result
			return
		}
	}
	f.Results = append(f.Results, result)
}
