package work

import (
	"context"
	"dotawesome/internal/deploy"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Focus selects the most recently used exact app match without changing layout.
func Focus(ctx context.Context, run deploy.Runner, ids []string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if len(ids) == 0 {
		return fmt.Errorf("focus requires at least one app ID")
	}
	out, err := run(ctx, "niri", "msg", "--json", "windows")
	if err != nil {
		return err
	}
	type timestamp struct {
		Secs  uint64
		Nanos uint32
	}
	var windows []struct {
		ID        int
		App       string    `json:"app_id"`
		Timestamp timestamp `json:"focus_timestamp"`
	}
	if err := json.Unmarshal([]byte(out), &windows); err != nil {
		return err
	}
	best := -1
	for i, w := range windows {
		if !slices.Contains(ids, w.App) {
			continue
		}
		if best < 0 || w.Timestamp.Secs > windows[best].Timestamp.Secs ||
			w.Timestamp.Secs == windows[best].Timestamp.Secs && w.Timestamp.Nanos > windows[best].Timestamp.Nanos ||
			w.Timestamp == windows[best].Timestamp && w.ID < windows[best].ID {
			best = i
		}
	}
	if best < 0 {
		return fmt.Errorf("no open window for %s; use Super+G then / to search", strings.Join(ids, ", "))
	}
	_, err = run(ctx, "niri", "msg", "action", "focus-window", "--id", strconv.Itoa(windows[best].ID))
	return err
}
