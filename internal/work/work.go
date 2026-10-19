// Package work arranges exact profile-owned windows through bounded Niri IPC.
package work

import (
	"context"
	"dotawesome/internal/components"
	"dotawesome/internal/deploy"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"
	"time"
)

type workspace struct {
	Name string
	ID   int
}
type window struct {
	ID        int
	Workspace int    `json:"workspace_id"`
	App       string `json:"app_id"`
}

func Arrange(ctx context.Context, run deploy.Runner, apps []components.App, timeout time.Duration) []string {
	ipc := run
	run = func(ctx context.Context, args ...string) (string, error) {
		bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return ipc(bounded, args...)
	}
	out, e := run(ctx, "niri", "msg", "--json", "workspaces")
	if e != nil {
		return []string{e.Error()}
	}
	var ws []workspace
	if e = json.Unmarshal([]byte(out), &ws); e != nil {
		return []string{e.Error()}
	}
	spaces := map[string]int{}
	for _, w := range ws {
		spaces[w.Name] = w.ID
	}
	fail := []string{}
	first := 0
	action := func(a ...string) error {
		_, e := run(ctx, append([]string{"niri", "msg", "action"}, a...)...)
		return e
	}
	for _, a := range apps {
		if len(a.Command) == 0 {
			continue
		}
		err := func() error {
			target, ok := spaces[a.Workspace]
			if !ok {
				return fmt.Errorf("workspace %s is missing", a.Workspace)
			}
			matching := func() ([]window, error) {
				out, e := run(ctx, "niri", "msg", "--json", "windows")
				if e != nil {
					return nil, e
				}
				var windows []window
				if e = json.Unmarshal([]byte(out), &windows); e != nil {
					return nil, e
				}
				matches := []window{}
				for _, w := range windows {
					for _, id := range a.IDs {
						if w.App == id {
							matches = append(matches, w)
							break
						}
					}
				}
				sort.Slice(matches, func(i, j int) bool {
					if (matches[i].Workspace == target) != (matches[j].Workspace == target) {
						return matches[i].Workspace == target
					}
					return matches[i].ID < matches[j].ID
				})
				return matches, nil
			}
			matches, e := matching()
			if e != nil {
				return e
			}
			newWindow := len(matches) == 0
			if newWindow {
				if e = action(append([]string{"spawn", "--"}, a.Command...)...); e != nil {
					return e
				}
				deadline := time.Now().Add(timeout)
				for {
					matches, e = matching()
					if e != nil {
						return e
					}
					if len(matches) > 0 || time.Now().After(deadline) {
						break
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(150 * time.Millisecond):
					}
				}
			}
			if len(matches) == 0 {
				return fmt.Errorf("no matching window before startup timeout; inspect niri msg --json windows")
			}
			id := strconv.Itoa(matches[0].ID)
			if e = action("focus-window", "--id", id); e != nil {
				return e
			}
			if e = action("move-window-to-workspace", "--window-id", id, "--focus", "false", a.Workspace); e != nil {
				return e
			}
			if newWindow {
				if e = action("focus-window", "--id", id); e != nil {
					return e
				}
				if e = action("set-column-width", a.Width); e != nil {
					return e
				}
			}
			if a.Workspace == "One" && first == 0 {
				first = matches[0].ID
			}
			return nil
		}()
		if err != nil {
			fail = append(fail, a.Name+": "+err.Error())
		}
	}
	if e = action("focus-workspace", "One"); e != nil {
		fail = append(fail, e.Error())
	}
	if first != 0 {
		if e = action("focus-window", "--id", strconv.Itoa(first)); e != nil {
			fail = append(fail, e.Error())
		}
	}
	return fail
}
func Main(ctx context.Context, run deploy.Runner) error {
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		return fmt.Errorf("XDG_RUNTIME_DIR is required inside the Niri session")
	}
	fd, e := syscall.Open(filepath.Join(runtime, "dotawesome-niri-work.lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return e
	}
	defer syscall.Close(fd)
	if e = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); e == syscall.EWOULDBLOCK {
		return nil
	} else if e != nil {
		return e
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	var profile components.Profile
	if e = deploy.ReadJSON(filepath.Join(filepath.Dir(exe), "profile.json"), &profile); e != nil {
		return e
	}
	fail := Arrange(ctx, run, profile.Apps, 12*time.Second)
	if len(fail) > 0 {
		message := ""
		for _, f := range fail {
			message += f + "\n"
		}
		run(ctx, "noctalia", "msg", "notification-show", "Work layout: launch failures", message)
		return fmt.Errorf("%s", message)
	}
	return nil
}
