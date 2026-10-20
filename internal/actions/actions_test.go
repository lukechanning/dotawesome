package actions

import "testing"

func TestDirectAndLegacyRouting(t *testing.T) {
	for _, tc := range []struct{ in, want string }{{"sync", "config sync"}, {"config sync", "config sync"}, {"syncthing setup", "syncthing setup"}, {"app setup", "app setup"}} {
		if got := Resolve(tc.in); got != tc.want {
			t.Errorf("%s: %s", tc.in, got)
		}
	}
	if len(All) != 6 {
		t.Fatal(len(All))
	}
}
