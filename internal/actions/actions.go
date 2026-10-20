// Package actions defines achievable actions shared by help, routing and navigation.
package actions

type Action struct{ Label, Description, Command string }

var All = []Action{
	{"Syncthing Setup", "Install, configure and pair Documents syncing over Tailscale. Complete this guide on both machines.", "syncthing setup"},
	{"Syncthing Status", "Check connectivity and Documents sync progress.", "syncthing status"},
	{"Config Setup", "Set up shared Git, Fish, Alacritty and AWS configuration.", "config setup"},
	{"Config Sync", "Pull repository updates, review and apply managed configuration.", "config sync"},
	{"App Setup", "Choose applications, check their prerequisites and review configuration.", "app setup"},
	{"Troubleshoot or Restore", "Inspect differences, recover updates or restore originals.", "troubleshoot"},
}

func Resolve(command string) string {
	if command == "sync" {
		return "config sync"
	}
	for _, a := range All {
		if a.Command == command {
			return a.Command
		}
	}
	return ""
}

func Index(command string) int {
	for i, a := range All {
		if a.Command == command {
			return i
		}
	}
	return 0
}
