package components

import (
	"encoding/json"
	"strings"
)

// JSON is also valid YAML, so the menu needs no additional parser dependency.
func jumpMenu(template []byte, helper string, apps []App) ([]byte, error) {
	var menu map[string]any
	if err := json.Unmarshal(template, &menu); err != nil {
		return nil, err
	}
	shellQuote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	focus := func(ids []string) string {
		cmd := shellQuote(helper) + " focus"
		for _, id := range ids {
			cmd += " " + shellQuote(id)
		}
		return cmd
	}
	entries := []map[string]string{{"key": "s", "desc": "Slack", "cmd": focus([]string{"slack", "Slack", "com.slack.Slack"})}}
	for _, key := range []string{"b", "t"} {
		for _, app := range apps {
			if app.JumpKey == key {
				entries = append(entries, map[string]string{"key": key, "desc": app.Name, "cmd": focus(app.IDs)})
			}
		}
	}
	entries = append(entries, map[string]string{"key": "/", "desc": "Search open windows", "cmd": "noctalia msg panel-open launcher '/win '"})
	menu["menu"] = entries
	b, err := json.MarshalIndent(menu, "", "  ")
	return append(b, '\n'), err
}
