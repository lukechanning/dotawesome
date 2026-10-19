package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
	"dotawesome/internal/app"
	"dotawesome/internal/components"
)

type menuOption struct{ label, description, action string }

var moreOptions = []menuOption{
	{"Restore original files", "Return selected components to the files saved before Dotawesome first configured them.", "restore"},
	{"Shared configuration", "Set up shared Git, shell and terminal files with chezmoi, or recover this machine's encryption key.", "init"},
	{"Enroll recovery identity", "Create and verify the first encryption identity. Keep its recovery passphrase somewhere safe.", "enroll-age"},
	{"AWS login", "Sign in to the local cof-aws account. Login credentials stay on this machine.", "login"},
	{"Recover interrupted update", "Roll back an unfinished file update using its saved originals. Later edits are protected.", "recover"},
	{"Refresh configuration check", "Check the saved configuration again without applying changes.", "refresh"},
	{"Help", "Learn how configuration review, backups and restoration work.", "help"},
	{"Inspect app configuration", "Check individual apps, inspect local edits and review their files.", "components"},
	{"Back", "Return home.", "back"},
}

func componentName(name string) string {
	switch name {
	case "niri":
		return "Niri + Noctalia"
	case "nvim":
		return "Neovim"
	case "tmux":
		return "tmux"
	}
	return name
}
func componentPurpose(name string) string {
	switch name {
	case "niri":
		return "Desktop, panel and workspace layout"
	case "nvim":
		return "Editor settings and pinned plugins"
	case "tmux":
		return "Terminal sessions and shortcuts"
	}
	return "Workstation configuration"
}
func (m Model) componentStatus(name string) app.Result {
	for _, r := range m.status {
		if r.Component == name {
			return r
		}
	}
	return app.Result{Component: name, State: "unknown"}
}
func configurationStatus(r app.Result) (string, string) {
	if strings.Contains(r.Detail, "source unavailable:") {
		return "Unable to check", "The repository source could not be read. Open technical details before reviewing changes."
	}
	switch r.State {
	case "inactive":
		return "Not managed yet", "Dotawesome has not changed these configuration files. The app may already be installed."
	case "current":
		return "Configuration matches", "Managed files match the repository. No configuration update is needed."
	case "source-changed":
		return "Repository changes available", "The repository has different configuration. You can review the changes before applying them."
	case "locally-modified":
		return "Local edits found", "Your files differ from the last version Dotawesome deployed. Updates are paused to protect your edits."
	default:
		return "Unable to check", "The configuration check has not produced a usable result. Refresh the check or open technical details."
	}
}
func (m Model) needsRecovery() bool {
	for _, r := range m.status {
		if r.State == "recovery pending" {
			return true
		}
	}
	return false
}
func (m Model) homeSummary() string {
	if m.err != nil {
		return "The configuration check could not finish. Open component details to investigate."
	}
	if m.needsRecovery() {
		return "An update was interrupted. Recover it before reviewing new changes."
	}
	if m.loading {
		return m.spin.View() + " Inspecting saved configuration…"
	}
	managed, changed, edits, unknown := 0, 0, 0, 0
	for _, r := range m.status {
		if r.Component == "" {
			continue
		}
		label, _ := configurationStatus(r)
		switch label {
		case "Configuration matches":
			managed++
		case "Not managed yet":
		case "Local edits found":
			managed++
			edits++
		case "Repository changes available":
			managed++
			changed++
		default:
			unknown++
		}
		if strings.Contains(r.Detail, "Interrupted package operation") {
			return "A package operation needs attention. See component details before continuing."
		}
	}
	if unknown > 0 {
		return "Some configuration could not be checked. See component details."
	}
	if edits > 0 {
		return "Local edits need attention. Review them in component details."
	}
	if changed > 0 {
		return "Repository changes are available to review."
	}
	if managed == 0 {
		return "Choose what you want Dotawesome to configure."
	}
	return "Your managed app configuration matches the repository."
}
func (m Model) menu(options []menuOption, index int) string {
	var b strings.Builder
	for i, o := range options {
		line := "  " + o.label
		if i == index {
			line = m.accent("> " + o.label)
		}
		b.WriteString(line + "\n")
	}
	if index >= 0 && index < len(options) && options[index].description != "" {
		b.WriteString("\n" + options[index].description + "\n")
	}
	return b.String()
}
func (m Model) guidedBody() (string, bool) {
	switch m.screen {
	case "overview":
		options := []menuOption{{"Set up this machine", "First time here, or want to change which apps are configured? Start here. Completed steps are reused.", ""}, {"Sync this machine", "Get the latest repository updates, review changes and apply them to this machine.", ""}, {"Troubleshoot or restore", "Inspect local edits, check individual apps or restore previous configuration.", ""}}
		return "What would you like to do?\nYou’ll review changes before anything is applied.\n\n" + m.homeSummary() + "\n\n" + m.menu(options, m.nav), true
	case "components":
		options := []menuOption{}
		for _, name := range components.Names {
			status, explanation := configurationStatus(m.componentStatus(name))
			options = append(options, menuOption{label: componentName(name) + " — " + status, description: explanation})
		}
		if m.loading {
			return "Component configuration\n\n" + m.spin.View() + " Inspecting saved configuration…\n", true
		}
		return "Component configuration\nThese labels describe managed files, not whether apps are running.\n\n" + m.menu(options, m.cursor), true
	case "component":
		r := m.componentStatus(m.component)
		label, explanation := configurationStatus(r)
		options := m.componentActions()
		return componentName(m.component) + " — " + label + "\n" + componentPurpose(m.component) + "\n\n" + explanation + "\n\n" + m.menu(options, m.cursor), true
	case "more":
		return "Troubleshoot or restore\nCheck a problem or restore previous configuration.\n\n" + m.menu(moreOptions, m.cursor), true
	case "checks":
		return "Prerequisite checks\nResolve the errors before applying configuration. p opens package setup.\n\n" + m.view.View() + "\n", true
	case "files":
		return "Changed files\nChoose a file to inspect.\n\n" + m.view.View() + "\n", true
	case "technical":
		return componentName(m.component) + " — technical details\n\n" + m.view.View() + "\n", true
	case "differences":
		if m.loading {
			return "Inspect differences\n\n" + m.spin.View() + " Reading configuration files…\n", true
		}
		return "Inspect differences\nThis preview does not change your files.\n\n" + m.view.View() + "\n", true
	case "external-confirm":
		label, description := m.externalDescription()
		return label + "\n\n" + description + "\n\nThe command will use this terminal, then return here.\nEnter to continue, or Escape to go back.\n", true
	case "help":
		return "How Dotawesome works\n\nChoose what to configure, check prerequisites, then review the changes.\nNothing is deployed until you confirm. Originals are saved for restoration.\nLocal edits block updates so your work is protected.\n\nComponent details explain configuration status and let you inspect files.\nTroubleshoot or restore contains app checks, restoration and advanced setup.\n\nEscape returns home. Explicit commands are documented in docs/usage.md.\n", true
	}
	return "", false
}
func (m Model) componentActions() []menuOption {
	review := menuOption{"Review configuration", "Check prerequisites and preview repository changes before applying anything.", "review"}
	inspect := menuOption{"Inspect differences", "Compare your files with the saved or repository configuration without changing them.", "diff"}
	options := []menuOption{review, inspect}
	if m.componentStatus(m.component).State == "locally-modified" {
		options = []menuOption{inspect, review}
	}
	return append(options, menuOption{"Technical details", "Show file paths, package records and diagnostic messages.", "technical"}, menuOption{"Back", "Return to the component list.", "back"})
}
func (m Model) externalDescription() (string, string) {
	for _, o := range moreOptions {
		if o.action == m.externalAction {
			return o.label, o.description
		}
	}
	if m.externalAction == "install-packages" {
		return "Install required packages", "This runs sudo pacman -Syu --needed, including a full system upgrade.\nPackage additions are recorded for optional cleanup. Recheck configuration afterwards."
	}
	return "Continue terminal operation", "Review the command's prompts before proceeding."
}
func (m Model) profileLabel() string {
	switch m.profile {
	case "g14":
		return "G14"
	case "desktop":
		return "Desktop"
	case "generic":
		return "Standard profile"
	default:
		return "Custom profile · " + filepath.Base(m.profile)
	}
}
func (m Model) footer() string {
	if strings.HasPrefix(m.screen, "setup") {
		if m.screen == "setup-issues" {
			return "↑↓ Scroll   Esc Back"
		}
		if m.screen == "setup-finish" {
			return "↑↓ Choose   PgUp/PgDn Results   Enter Continue"
		}
		return "↑↓ Choose   Enter Continue   Esc Back   ? Help"
	}
	switch m.screen {
	case "overview":
		return "↑↓ Choose   Enter Open   ? Help   q Quit"
	case "components", "component", "more":
		return "↑↓ Choose   Enter Open   Esc Back   ? Help"
	case "differences", "technical":
		return "↑↓ Scroll   Esc Back   ? Help"
	case "checks":
		return "↑↓ Scroll   p Packages   Esc Back"
	case "files":
		return "↑↓ Choose   Enter Inspect   Esc Back"
	case "execute":
		return "Esc Request safe stop"
	case "external-confirm":
		return "Enter Continue   Esc Back"
	case "select":
		return "Space Select   p Profile   Enter Continue   Esc Back"
	case "review":
		return "←→ File   ↑↓ Scroll   f Files   c Checks   Enter Continue   ? Help"
	case "confirm":
		return "Enter Confirm   Esc Back"
	case "results":
		return "Enter Home   r Review again   ↑↓ Scroll"
	default:
		return "Esc Back   ? Help"
	}
}
func (m Model) contextHelp() string {
	if strings.HasPrefix(m.screen, "setup") {
		return "Setup checks the machine before offering the next incomplete step.\nExisting configuration and identities are reused. Terminal actions require confirmation.\nSet up this machine lets you change your app selections. Sync gets repository updates before reviewing this machine’s configuration."
	}
	switch m.screen {
	case "overview", "components", "component", "more":
		return "Arrows navigate; Tab selects the next item. Enter opens it.\nEscape goes back; q quits. Checking configuration does not deploy files."
	case "select":
		return "Space toggles a component. p changes the Niri machine profile.\nEnter checks prerequisites, then opens a preview of the changes."
	case "review", "differences":
		return "Arrows and Page Up/Down scroll. In review, left/right changes files.\nf lists changed files; c explains prerequisite checks; p opens package setup.\nThe preview is read-only. Deployment requires a separate confirmation."
	case "checks":
		return "↑↓ Scroll   p Packages   Esc Back"
	case "files":
		return "↑↓ Choose   Enter Inspect   Esc Back"
	case "execute":
		return "Escape or Ctrl-C requests a safe stop. Wait for writes and rollback to finish."
	default:
		return "Escape goes back. Terminal commands return to the application when finished."
	}
}
func (m Model) technicalText() string {
	r := m.componentStatus(m.component)
	label, explanation := configurationStatus(r)
	text := fmt.Sprintf("%s\n%s\n\nProfile: %s\n", label, explanation, m.profile)
	if r.Detail != "" {
		text += "\n" + r.Detail + "\n"
	}
	if r.Backup != "" {
		text += "\nRendered snapshot: " + r.Backup + "\n"
	}
	if m.s != nil {
		if records, e := m.s.Manager.Records(); e == nil {
			paths := []string{}
			for path := range records[m.component].Files {
				paths = append(paths, path)
			}
			sort.Strings(paths)
			if len(paths) > 0 {
				text += "\nManaged files:\n" + strings.Join(paths, "\n") + "\n"
			}
		}
	}
	if m.err != nil {
		text += "\nCheck error: " + m.err.Error() + "\n"
	}
	return text
}

func (m *Model) openExternal(action string) {
	m.confirmOrigin = m.screen
	m.externalAction = action
	m.screen = "external-confirm"
}
func (m *Model) back() {
	switch m.screen {
	case "component":
		m.screen = "components"
		for i, n := range components.Names {
			if n == m.component {
				m.cursor = i
			}
		}
	case "setup":
		m.setup = nil
		m.screen = "overview"
		m.nav = 0
	case "setup-check":
		m.setup = nil
		m.screen = "overview"
		m.nav = 0
		m.loading = false
	case "setup-machine", "setup-hooks", "setup-paused", "setup-issues":
		m.screen = "setup"
		m.cursor = 0
		m.loading = false
	case "setup-finish":
		m.setup = nil
		m.screen = "overview"
		m.nav = 0
	case "setup-accounts":
		m.screen = "setup-finish"
		m.cursor = 0
	case "files", "checks":
		m.screen = "review"
		m.showDiff()
	case "technical", "differences":
		m.screen = "component"
		m.cursor = 0
		m.loading = false
		m.err = nil
	case "external-confirm":
		m.screen = m.confirmOrigin
		if m.screen == "" {
			m.screen = "overview"
		}
	case "confirm":
		m.screen = "review"
	case "preflight", "review":
		if m.selectionOrigin == "component" {
			m.screen = "component"
			m.cursor = 0
		} else {
			m.screen = "select"
		}
		m.loading = false
		m.err = nil
	case "select":
		m.screen = m.selectionOrigin
		if m.screen == "" {
			m.screen = "overview"
		}
		m.cursor = 0
	case "help":
		m.screen = "more"
		m.cursor = 6
	default:
		m.screen = "overview"
		m.nav = 0
		m.cursor = 0
		m.err = nil
	}
}

func (m Model) checkErrors() int {
	count := 0
	for _, r := range m.reviews {
		count += len(r.Errors)
	}
	return count
}
func (m Model) displayFile(path string) string {
	if m.s != nil {
		return strings.Replace(path, m.s.Manager.Home+"/", "~/", 1)
	}
	return path
}
func (m *Model) showFileList() {
	options := []menuOption{}
	for _, path := range m.files {
		options = append(options, menuOption{label: m.displayFile(path)})
	}
	m.view.SetContent(m.menu(options, m.fileIndex))
	m.view.SetYOffset(max(0, m.fileIndex-m.view.Height()/2))
}

func (m Model) wrap(s string) string { return lipgloss.NewStyle().Width(max(12, m.width)).Render(s) }
func (m Model) fitBody(body string, budget int) string {
	lines := strings.Split(m.wrap(strings.TrimRight(body, "\n")), "\n")
	if len(lines) <= budget {
		return strings.Join(lines, "\n") + "\n"
	}
	start := 0
	if m.screen == "overview" || m.screen == "components" || m.screen == "component" || m.screen == "more" || strings.HasPrefix(m.screen, "setup") {
		for i, line := range lines {
			if strings.Contains(line, "> ") {
				if i >= budget-3 {
					start = max(0, i-budget/2)
				}
				break
			}
		}
	}
	if start+budget > len(lines) {
		start = max(0, len(lines)-budget)
	}
	end := min(len(lines), start+budget)
	visible := append([]string{}, lines[start:end]...)
	if start > 0 {
		visible[0] = m.muted("↑ More above")
	}
	if end < len(lines) {
		visible[len(visible)-1] = m.muted("↓ More below")
	}
	return strings.Join(visible, "\n") + "\n"
}
