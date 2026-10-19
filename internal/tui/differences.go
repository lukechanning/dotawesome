package tui

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"dotawesome/internal/app"
	"dotawesome/internal/components"
	"dotawesome/internal/deploy"
)

type differencesMsg struct {
	text string
	err  error
}

// Inspecting edits deliberately avoids Plan, whose drift guard rejects deployment.
func (m Model) inspectDifferences() tea.Cmd {
	return func() tea.Msg {
		if m.s == nil {
			return differencesMsg{err: fmt.Errorf("configuration service is unavailable")}
		}
		manager := m.s.Manager
		records, e := manager.Records()
		if e != nil {
			return differencesMsg{err: e}
		}
		record := records[m.component]
		desired := deploy.Files{}
		localEdits := m.componentStatus(m.component).State == "locally-modified"
		if localEdits {
			for p, entry := range record.Files {
				desired[p] = entry.Installed
			}
		} else {
			profile := ""
			if m.component == "niri" {
				profile = m.profile
			}
			desired, e = components.Render(manager, m.component, profile, m.s.Helper)
			if e != nil {
				return differencesMsg{err: e}
			}
			for p, entry := range record.Files {
				if _, ok := desired[p]; !ok {
					desired[p] = entry.Original
				}
			}
		}
		paths := []string{}
		for p := range desired {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		var b strings.Builder
		if localEdits {
			b.WriteString("Saved version → your local files\n\n")
		} else {
			b.WriteString("Your files → repository configuration\n\n")
		}
		changed := 0
		for _, p := range paths {
			current, e := deploy.Inspect(p)
			if e != nil {
				return differencesMsg{err: e}
			}
			before, after := current, desired[p]
			if localEdits {
				before, after = after, before
			}
			if reflect.DeepEqual(before, after) {
				continue
			}
			changed++
			diff, _, _ := app.Unified(p, before, after)
			b.WriteString(m.colorDiff(diff) + "\n")
		}
		if changed == 0 {
			b.WriteString("No file differences found.\n")
		}
		return differencesMsg{text: b.String()}
	}
}
