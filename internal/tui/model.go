// Package tui renders a persistent dashboard over the same services as explicit commands.
package tui

import (
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"context"
	"dotawesome/internal/app"
	"dotawesome/internal/components"
	"dotawesome/internal/deploy"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type statusMsg struct {
	results []app.Result
	err     error
}
type inspectedMsg struct {
	reviews []*app.Inspection
	err     error
}
type finishedMsg struct{ results []app.Result }
type eventMsg app.Event
type handoffMsg struct{ err error }
type tickMsg time.Time
type Preferences struct {
	Machine    string   `json:"machine"`
	Components []string `json:"components"`
}
type Model struct {
	s                                                   *app.Service
	screen                                              string
	nav, cursor, profileIndex, width, height, fileIndex int
	selected                                            map[string]bool
	profile                                             string
	status, results                                     []app.Result
	reviews                                             []*app.Inspection
	files                                               []string
	diffs                                               map[string]string
	summary                                             string
	err                                                 error
	loading, help, stopping                             bool
	spin                                                spinner.Model
	view                                                viewport.Model
	started                                             time.Time
	cancel                                              context.CancelFunc
	events                                              chan app.Event
	mono                                                bool
	restore                                             bool
	externalAction                                      string
	component, confirmOrigin, selectionOrigin           string
	setup                                               *setupFlow
	startup                                             tea.Cmd
}

func New(s *app.Service) Model {
	spin := spinner.New()
	spin.Spinner = spinner.Line
	m := Model{s: s, screen: "overview", selected: map[string]bool{"niri": true, "nvim": true, "tmux": true}, profile: "generic", width: 80, height: 28, loading: true, spin: spin, view: viewport.New(), mono: os.Getenv("NO_COLOR") != "", diffs: map[string]string{}}
	if s != nil {
		var p Preferences
		if e := deploy.ReadJSON(filepath.Join(s.Manager.State, "preferences.json"), &p); e == nil {
			if p.Machine != "g14" && p.Machine != "desktop" && p.Machine != "" {
				m.err = fmt.Errorf("invalid saved machine; inspect preferences.json")
			} else if p.Machine != "" {
				m.profile = p.Machine
			}
			if p.Components != nil {
				m.selected = map[string]bool{}
				for _, n := range p.Components {
					m.selected[n] = true
				}
			}
		} else if !os.IsNotExist(e) {
			m.err = e
		}
	}
	if s != nil {
		var local struct{ Data struct{ Machine string } }
		if b, e := os.ReadFile(filepath.Join(s.Manager.Config, "chezmoi/chezmoi.toml")); e == nil {
			if e = toml.Unmarshal(b, &local); e != nil {
				m.err = e
			} else if local.Data.Machine == "g14" || local.Data.Machine == "desktop" {
				m.profile = local.Data.Machine
			}
		}
		if records, e := s.Manager.Records(); e == nil {
			if niri, ok := records["niri"]; ok && niri.Profile != nil {
				m.profile = *niri.Profile
			}
		}
	}
	m.view.SoftWrap = true
	m.resize()
	return m
}
func (m Model) Init() tea.Cmd {
	if m.startup != nil {
		return tea.Batch(m.startup, m.spin.Tick)
	}
	return tea.Batch(m.inspectStatus(), m.spin.Tick)
}
func (m Model) inspectStatus() tea.Cmd {
	if m.s == nil {
		return nil
	}
	return func() tea.Msg { r, e := m.s.Status(context.Background()); return statusMsg{r, e} }
}
func (m *Model) resize() {
	m.view.SetWidth(max(12, m.width-4))
	header := 10
	if m.expandedLogo() {
		header += 4
	}
	if m.screen == "setup-finish" {
		header += 6
	}
	m.view.SetHeight(max(3, m.height-header))
}
func (m Model) accent(s string) string {
	if m.mono {
		return s
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#65c5a5")).Bold(true).Render(s)
}
func (m Model) muted(s string) string {
	if m.mono {
		return s
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#929c99")).Render(s)
}
func (m Model) inspectSelection() tea.Cmd {
	registerShared := m.setup != nil && m.setup.Report != nil && m.setup.Report.Shared == "current" && !m.setup.Incomplete
	names := []string{}
	for _, n := range components.Names {
		if m.selected[n] {
			names = append(names, n)
		}
	}
	return func() tea.Msg {
		pref := Preferences{Components: names}
		if m.profile == "g14" || m.profile == "desktop" {
			pref.Machine = m.profile
		}
		if e := m.s.Manager.WithLock(context.Background(), false, func() error { return m.s.Manager.SaveJSON(filepath.Join(m.s.Manager.State, "preferences.json"), pref) }); e != nil {
			return inspectedMsg{err: e}
		}
		reviews := []*app.Inspection{}
		if registerShared {
			if e := m.s.RecordRequirements(context.Background(), "shared", app.SharedRequirements()); e != nil {
				return inspectedMsg{err: e}
			}
		}
		for _, n := range names {
			p := ""
			if n == "niri" {
				p = m.profile
			}
			r, e := m.s.Inspect(context.Background(), n, p, m.restore)
			if e != nil {
				return inspectedMsg{reviews, e}
			}
			reviews = append(reviews, r)
		}
		return inspectedMsg{reviews, nil}
	}
}
func (m *Model) prepareDiffs() {
	m.files = nil
	m.diffs = map[string]string{}
	var summary strings.Builder
	for _, r := range m.reviews {
		fmt.Fprintf(&summary, "%s: %d file(s) changed\n", r.Plan.Component, len(r.Plan.Changes))
		for _, e := range r.Errors {
			summary.WriteString("  error: " + e + "\n")
		}
		for _, w := range r.Warnings {
			summary.WriteString("  warning: " + w + "\n")
		}
		if len(r.Packages) > 0 {
			summary.WriteString("  packages required: " + strings.Join(r.Packages, ", ") + "\n")
		}
		for p, c := range r.Plan.Changes {
			d, a, b := app.Unified(p, c.Before, c.After)
			key := fmt.Sprintf("%s  +%d -%d", p, a, b)
			m.files = append(m.files, key)
			m.diffs[key] = m.colorDiff(d)
		}
	}
	sort.Strings(m.files)
	m.fileIndex = 0
	m.summary = summary.String()
	m.showDiff()
}
func (m Model) colorDiff(d string) string {
	if m.mono {
		return d
	}
	lines := strings.Split(d, "\n")
	for i, line := range lines {
		color := ""
		if strings.HasPrefix(line, "+") {
			color = "#65c5a5"
		} else if strings.HasPrefix(line, "-") {
			color = "#ef8c86"
		} else if strings.HasPrefix(line, "@@") {
			color = "#8db9df"
		}
		if color != "" {
			lines[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
func (m *Model) showDiff() {
	d := "No file changes. You can check prerequisites with c."
	if len(m.files) > 0 {
		d = m.diffs[m.files[m.fileIndex]]
	}
	m.view.SetContent(d)
	m.view.GotoTop()
}
func (m *Model) start() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.screen = "execute"
	m.started = time.Now()
	m.stopping = false
	m.results = nil
	m.events = make(chan app.Event, 16)
	reviews := append([]*app.Inspection{}, m.reviews...)
	s := m.s
	events := m.events
	execute := func() tea.Msg {
		results := []app.Result{}
		for _, r := range reviews {
			if ctx.Err() != nil {
				results = append(results, app.Result{Component: r.Plan.Component, State: "skipped", Detail: "safe stop requested"})
				continue
			}
			if len(r.Errors) > 0 {
				results = append(results, app.Result{Component: r.Plan.Component, State: "failed", Detail: strings.Join(r.Errors, "; ")})
				continue
			}
			_, alreadyManaged := r.Plan.Ledger[r.Plan.Component]
			if len(r.Plan.Changes) == 0 && (alreadyManaged || r.Restore) {
				results = append(results, app.Result{Component: r.Plan.Component, State: "skipped", Detail: "current; no file changes"})
				continue
			}
			events <- app.Event{Component: r.Plan.Component, State: "running", Detail: "validating and deploying"}
			if e := s.Execute(ctx, r, false); e != nil {
				state := "failed"
				if r.Plan.Component == "niri" && strings.Contains(e.Error(), "log out") {
					state = "deferred"
				}
				results = append(results, app.Result{Component: r.Plan.Component, State: state, Detail: e.Error()})
			} else {
				required := r.Packages
				if r.Restore {
					required = nil
				}
				if e := s.RecordRequirements(ctx, r.Plan.Component, required); e != nil {
					results = append(results, app.Result{Component: r.Plan.Component, State: "failed", Detail: "Configuration changed but package requirements could not be recorded: " + e.Error()})
					continue
				}
				detail := "Configuration applied. "
				if r.Restore {
					detail = "Original files restored. "
				}
				detail += "Inspect status before retrying; application data retained."
				results = append(results, app.Result{Component: r.Plan.Component, State: "completed", Detail: detail, Backup: filepath.Join(s.Manager.State, "installed.json")})
			}
		}
		close(events)
		return finishedMsg{results}
	}
	return tea.Batch(execute, m.waitEvent(), func() tea.Msg { time.Sleep(time.Second); return tickMsg(time.Now()) })
}
func (m Model) waitEvent() tea.Cmd {
	events := m.events
	return func() tea.Msg {
		e, ok := <-events
		if !ok {
			return nil
		}
		return eventMsg(e)
	}
}
func (m Model) handoff(action string) tea.Cmd {
	exe, e := os.Executable()
	if e != nil {
		return func() tea.Msg { return handoffMsg{e} }
	}
	args := []string{action}
	switch action {
	case "init":
		if m.profile == "g14" || m.profile == "desktop" {
			args = append(args, "--machine", m.profile)
		}
	case "install-packages":
		for _, r := range m.reviews {
			if !r.Restore {
				args = append(args, r.Plan.Component)
			}
		}
	}
	c := exec.Command(exe, args...)
	c.Env = append(os.Environ(), "DOTAWESOME_ROOT="+m.s.Manager.Repo)
	return tea.ExecProcess(c, func(e error) tea.Msg { return handoffMsg{e} })
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if next, cmd, handled := m.updateSetup(msg); handled {
		return next, cmd
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		m.resize()
		return m, nil
	case statusMsg:
		m.loading = false
		m.status = v.results
		if v.err != nil {
			m.err = v.err
		}
		return m, nil
	case differencesMsg:
		if m.screen != "differences" {
			return m, nil
		}
		m.loading = false
		m.err = v.err
		m.view.SetContent(v.text)
		m.view.GotoTop()
		return m, nil
	case inspectedMsg:
		if m.screen != "preflight" {
			return m, nil
		}
		m.loading = false
		m.reviews = v.reviews
		m.err = v.err
		if v.err != nil {
			m.screen = "preflight"
		} else {
			m.screen = "review"
			m.prepareDiffs()
		}
		return m, nil
	case finishedMsg:
		if m.cancel != nil {
			m.cancel()
		}
		if m.setup != nil {
			m.finishSetup(v.results)
			return m, nil
		}
		m.screen = "results"
		m.results = v.results
		m.showResults()
		m.loading = false
		return m, nil
	case eventMsg:
		if m.screen != "execute" {
			return m, nil
		}
		m.results = append(m.results, app.Result{Component: v.Component, State: v.State, Detail: v.Detail})
		return m, m.waitEvent()
	case handoffMsg:
		m.screen = "results"
		state := "completed"
		detail := "Return home and review configuration again before continuing."
		if v.err != nil {
			state = "failed"
			detail = v.err.Error()
		}
		m.results = []app.Result{{Component: m.externalAction, State: state, Detail: detail}}
		m.showResults()
		return m, nil
	case tickMsg:
		if m.screen == "execute" {
			return m, func() tea.Msg { time.Sleep(time.Second); return tickMsg(time.Now()) }
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		k := v.String()
		if m.screen == "execute" {
			if k == "esc" || k == "ctrl+c" {
				if m.cancel != nil {
					m.cancel()
				}
				m.stopping = true
			}
			return m, nil
		}
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		if k == "?" {
			m.help = !m.help
			return m, nil
		}
		if m.help && k != "esc" && k != "q" {
			return m, nil
		}
		if k == "esc" {
			if m.help {
				m.help = false
				return m, nil
			}
			wasHome := m.screen == "overview" || m.screen == "results"
			m.back()
			if wasHome {
				m.loading = true
				return m, m.inspectStatus()
			}
			return m, nil
		}
		if m.screen == "external-confirm" {
			if k == "enter" {
				action := m.externalAction
				m.screen = "handoff"
				return m, m.handoff(action)
			}
			return m, nil
		}
		if k == "q" && (m.screen == "overview" || m.screen == "components" || m.screen == "component" || m.screen == "more" || m.screen == "help" || m.screen == "select" || strings.HasPrefix(m.screen, "setup")) {
			return m, tea.Quit
		}
		switch m.screen {
		case "overview":
			switch k {
			case "up", "shift+tab", "left":
				m.nav = (m.nav + 3) % 4
			case "down", "tab", "right":
				m.nav = (m.nav + 1) % 4
			case "enter":
				switch m.nav {
				case 0:
					cmd := m.beginSetup("setup", false)
					return m, cmd
				case 1:
					m.setup = nil
					m.screen = "components"
					m.cursor = 0
				case 2:
					cmd := m.beginSetup("sync", false)
					return m, cmd
				case 3:
					m.setup = nil
					m.screen = "more"
					m.cursor = 0
				}
			}
		case "components":
			switch k {
			case "up", "shift+tab":
				m.cursor = (m.cursor + 2) % 3
			case "down", "tab":
				m.cursor = (m.cursor + 1) % 3
			case "enter":
				m.component = components.Names[m.cursor]
				m.screen = "component"
				m.cursor = 0
			}
		case "component":
			options := m.componentActions()
			switch k {
			case "up", "shift+tab":
				m.cursor = (m.cursor + len(options) - 1) % len(options)
			case "down", "tab":
				m.cursor = (m.cursor + 1) % len(options)
			case "enter":
				switch options[m.cursor].action {
				case "review":
					m.selected = map[string]bool{m.component: true}
					m.selectionOrigin = "component"
					m.restore = false
					m.loading = true
					m.screen = "preflight"
					m.err = nil
					return m, m.inspectSelection()
				case "diff":
					m.screen = "differences"
					m.loading = true
					m.err = nil
					return m, m.inspectDifferences()
				case "technical":
					m.screen = "technical"
					m.view.SetContent(m.technicalText())
					m.view.GotoTop()
				case "back":
					m.back()
				}
			}
		case "more":
			switch k {
			case "up", "shift+tab":
				m.cursor = (m.cursor + len(moreOptions) - 1) % len(moreOptions)
			case "down", "tab":
				m.cursor = (m.cursor + 1) % len(moreOptions)
			case "enter":
				action := moreOptions[m.cursor].action
				switch action {
				case "restore":
					m.restore = true
					m.selectionOrigin = "more"
					m.screen = "select"
					m.cursor = 0
					m.err = nil
				case "refresh":
					m.screen = "overview"
					m.nav = 0
					m.loading = true
					return m, m.inspectStatus()
				case "help":
					m.screen = "help"
				case "back":
					m.back()
				case "init":
					cmd := m.beginSetup("shared", false)
					return m, cmd
				default:
					m.openExternal(action)
				}
			}
		case "files":
			switch k {
			case "up", "shift+tab":
				m.fileIndex = (m.fileIndex + len(m.files) - 1) % len(m.files)
			case "down", "tab":
				m.fileIndex = (m.fileIndex + 1) % len(m.files)
			case "enter":
				m.screen = "review"
				m.showDiff()
				return m, nil
			}
			m.showFileList()
		case "technical", "differences", "checks":
			if m.screen == "checks" && k == "p" {
				m.openExternal("install-packages")
				return m, nil
			}
			var cmd tea.Cmd
			m.view, cmd = m.view.Update(msg)
			return m, cmd
		case "select":
			switch k {
			case "up":
				m.cursor = (m.cursor + 2) % 3
			case "down", "tab":
				m.cursor = (m.cursor + 1) % 3
			case "space", " ":
				n := components.Names[m.cursor]
				m.selected[n] = !m.selected[n]
			case "p":
				profiles := []string{"generic", "g14", "desktop"}
				m.profileIndex = (m.profileIndex + 1) % 3
				m.profile = profiles[m.profileIndex]
			case "enter":
				count := 0
				for _, v := range m.selected {
					if v {
						count++
					}
				}
				if count == 0 && m.setup != nil {
					m.finishSetup([]app.Result{{Component: "apps", State: "skipped", Detail: "No app configuration selected."}})
					return m, nil
				}
				if count > 0 {
					m.loading = true
					m.screen = "preflight"
					m.err = nil
					return m, m.inspectSelection()
				}
			}
		case "preflight":
			if k == "enter" {
				m.loading = true
				m.err = nil
				return m, m.inspectSelection()
			}
		case "review":
			switch k {
			case "left", "h":
				if len(m.files) > 0 {
					m.fileIndex = (m.fileIndex + len(m.files) - 1) % len(m.files)
					m.showDiff()
				}
			case "right", "l", "tab":
				if len(m.files) > 0 {
					m.fileIndex = (m.fileIndex + 1) % len(m.files)
					m.showDiff()
				}
			case "f":
				if len(m.files) > 0 {
					m.screen = "files"
					m.showFileList()
				}
			case "c":
				m.screen = "checks"
				m.view.SetContent(m.summary)
				m.view.GotoTop()
			case "p":
				m.openExternal("install-packages")
			case "enter":
				if m.checkErrors() > 0 {
					m.screen = "checks"
					m.view.SetContent(m.summary)
					m.view.GotoTop()
				} else {
					m.screen = "confirm"
				}
			default:
				var cmd tea.Cmd
				m.view, cmd = m.view.Update(msg)
				return m, cmd
			}
		case "confirm":
			if k == "enter" {
				return m, m.start()
			}
		case "results":
			if k == "r" {
				m.screen = "preflight"
				m.loading = true
				m.err = nil
				return m, m.inspectSelection()
			}
			if k == "enter" {
				m.screen = "overview"
				m.loading = true
				return m, m.inspectStatus()
			}
			var cmd tea.Cmd
			m.view, cmd = m.view.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}
func (m *Model) showResults() {
	var b strings.Builder
	for _, r := range m.results {
		fmt.Fprintf(&b, "%s: %s\n  %s\n", r.Component, r.State, r.Detail)
		if r.Backup != "" {
			fmt.Fprintf(&b, "  Backup record: %s\n", r.Backup)
		}
	}
	m.view.SetContent(b.String())
	m.view.GotoTop()
}
func (m Model) View() tea.View {
	var b strings.Builder
	title, subtitle := "DOTAWESOME", "Workstation configuration · "+m.profileLabel()
	if m.expandedLogo() {
		title = projectLogo
		subtitle = "DOTAWESOME · " + subtitle
	}
	b.WriteString(m.accent(title) + "\n")
	b.WriteString(m.muted(subtitle) + "\n\n")
	if m.help {
		b.WriteString(m.contextHelp() + "\n\n? or Escape closes help.")
		v := tea.NewView(m.wrap(b.String()))
		v.AltScreen = true
		return v
	}
	if m.err != nil && m.screen != "overview" {
		b.WriteString("Could not complete the check: " + m.err.Error() + "\n\n")
	}
	body, ok := m.setupBody()
	if !ok {
		body, ok = m.guidedBody()
	}
	if ok {
		footer := m.wrap(m.muted(m.footer()))
		budget := max(3, m.height-lipgloss.Height(m.wrap(b.String()))-lipgloss.Height(footer)-1)
		b.WriteString(m.fitBody(body, budget))
		b.WriteString("\n" + footer)
		v := tea.NewView(m.wrap(b.String()))
		v.AltScreen = true
		return v
	}
	switch m.screen {
	case "select":
		label := "Choose what to configure"
		if m.restore {
			label = "Restore original component files"
		}
		b.WriteString(m.accent(label) + "\nChoose the configuration to review. Nothing is applied yet.\n\n")
		for i, n := range components.Names {
			mark := " "
			if m.selected[n] {
				mark = "x"
			}
			cursor := "  "
			if i == m.cursor {
				cursor = "> "
			}
			fmt.Fprintf(&b, "%s[%s] %s — %s\n", cursor, mark, componentName(n), componentPurpose(n))
		}

	case "preflight":
		if m.err != nil {
			b.WriteString("Preflight blocked. Preserve local edits; recovery is available under More options.\nEnter: inspect again   Escape: back\n")
			break
		}
		b.WriteString(m.spin.View() + " Preflight: inspecting dependencies and ownership…\nEnter retries a fresh inspection.\n")
	case "review":
		b.WriteString(m.accent("Review changes") + "\nReview each file before confirming. Nothing has been applied.\n")
		if len(m.files) > 0 {
			fmt.Fprintf(&b, "File %d/%d: %s\n", m.fileIndex+1, len(m.files), m.displayFile(m.files[m.fileIndex]))
		} else {
			b.WriteString("No file changes.\n")
		}
		if count := m.checkErrors(); count > 0 {
			fmt.Fprintf(&b, "%d prerequisite issue(s) need attention. Press c for explanations.\n", count)
		}
		b.WriteString(m.view.View() + "\n")
	case "confirm":
		b.WriteString(m.accent("Confirm configuration changes") + "\n\n" + m.summary + "\nEnter approves these reviewed file changes. Escape returns.\n")
	case "handoff":
		b.WriteString("Terminal command running…\n")
	case "execute":
		fmt.Fprintf(&b, "%s Executing  %s elapsed\n", m.spin.View(), time.Since(m.started).Round(time.Second))
		if m.stopping {
			b.WriteString("Safe stop requested; waiting for rollback/reconciliation.\n")
		}
		for _, r := range m.results {
			fmt.Fprintf(&b, "  %s: %s — %s\n", r.Component, r.State, r.Detail)
		}
		b.WriteString("\nEscape: request safe stop\n")
	case "results":
		b.WriteString(m.accent("Results remain here until you continue") + "\n\n")
		b.WriteString(m.view.View() + "\n")
	}
	b.WriteString("\n" + m.muted(m.footer()))
	v := tea.NewView(m.wrap(b.String()))
	v.AltScreen = true
	return v
}
func Run(s *app.Service) error { return runModel(New(s)) }
func runModel(m Model) error {
	_, e := tea.NewProgram(m, tea.WithFilter(func(m tea.Model, msg tea.Msg) tea.Msg {
		_, interrupt := msg.(tea.InterruptMsg)
		_, quit := msg.(tea.QuitMsg)
		if interrupt || quit {
			if model, ok := m.(Model); ok && model.screen == "execute" {
				return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
			}
		}
		return msg
	})).Run()
	return e
}

func RunFlow(s *app.Service, intent string, pulled bool) error {
	m := New(s)
	m.startup = m.beginSetup(intent, pulled)
	return runModel(m)
}
