package work

import (
	"context"
	"dotawesome/internal/components"
	"encoding/json"
	"strings"
	"testing"
)

func TestArrangeUsesExactIDsAndPreservesExistingWidth(t *testing.T) {
	calls := []string{}
	run := func(ctx context.Context, a ...string) (string, error) {
		calls = append(calls, strings.Join(a, " "))
		if a[2] == "--json" {
			if a[3] == "workspaces" {
				return `[{"name":"One","id":1}]`, nil
			}
			b, _ := json.Marshal([]map[string]any{{"id": 2, "workspace_id": 1, "app_id": "Other"}, {"id": 3, "workspace_id": 1, "app_id": "Alacritty"}})
			return string(b), nil
		}
		return "", nil
	}
	fail := Arrange(context.Background(), run, []components.App{{Name: "terminal", Command: []string{"alacritty"}, IDs: []string{"Alacritty"}, Workspace: "One", Width: "100%"}}, 0)
	if len(fail) != 0 {
		t.Fatal(fail)
	}
	all := strings.Join(calls, "\n")
	if strings.Contains(all, "spawn") || strings.Contains(all, "set-column-width") || !strings.Contains(all, "focus-window --id 3") {
		t.Fatal(all)
	}
}
