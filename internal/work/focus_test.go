package work

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestFocusJumpsToMostRecentExactAppAcrossWorkspaces(t *testing.T) {
	var actions [][]string
	run := func(_ context.Context, args ...string) (string, error) {
		if reflect.DeepEqual(args, []string{"niri", "msg", "--json", "windows"}) {
			return `[
			 {"id":1,"app_id":"slack-helper","focus_timestamp":{"secs":100,"nanos":0}},
			 {"id":2,"app_id":"Slack","workspace_id":9,"focus_timestamp":{"secs":2,"nanos":1}},
			 {"id":3,"app_id":"slack","workspace_id":8,"focus_timestamp":{"secs":2,"nanos":2}}
			]`, nil
		}
		actions = append(actions, args)
		return "", nil
	}
	if err := Focus(context.Background(), run, []string{"slack", "Slack"}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"niri", "msg", "action", "focus-window", "--id", "3"}}
	if !reflect.DeepEqual(actions, want) {
		t.Fatalf("actions = %v, want %v", actions, want)
	}
}

func TestFocusMissingAppDoesNotLaunchOrMoveAnything(t *testing.T) {
	calls := 0
	run := func(_ context.Context, _ ...string) (string, error) {
		calls++
		return `[]`, nil
	}
	if err := Focus(context.Background(), run, []string{"slack"}); err == nil {
		t.Fatal("missing window accepted")
	}
	if calls != 1 {
		t.Fatalf("made %d calls", calls)
	}
}

func TestFocusPropagatesIPCFailures(t *testing.T) {
	failure := errors.New("IPC unavailable")
	for _, failList := range []bool{true, false} {
		run := func(_ context.Context, args ...string) (string, error) {
			if args[2] == "--json" && !failList {
				return `[{"id":1,"app_id":"slack"}]`, nil
			}
			return "", failure
		}
		if err := Focus(context.Background(), run, []string{"slack"}); !errors.Is(err, failure) {
			t.Fatal(err)
		}
	}
}
