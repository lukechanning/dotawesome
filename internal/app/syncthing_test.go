package app

import (
	"bytes"
	"context"
	"dotawesome/internal/deploy"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentsReconciliation(t *testing.T) {
	home := t.TempDir()
	for _, folder := range []map[string]any{{"id": "other", "path": filepath.Join(home, "Documents")}, {"id": documentsID, "path": "/elsewhere"}} {
		if validateDocuments([]map[string]any{folder}, home) == nil {
			t.Fatal("accepted conflicting mapping")
		}
	}
	if err := validateDocuments(nil, home); err != nil {
		t.Fatal(err)
	}
}
func TestPeerValidation(t *testing.T) {
	for _, id := range []string{"invalid", ""} {
		if validDeviceID(id) {
			t.Fatal(id)
		}
	}
	if validTailIP("192.168.1.2") || !validTailIP("100.64.0.2") {
		t.Fatal("address validation")
	}
}
func TestSyncEvidence(t *testing.T) {
	for _, tc := range []struct {
		state        string
		connected    bool
		need, remote float64
		want         string
	}{
		{"idle", false, 0, 100, "peer offline"}, {"idle", true, 0, 99, "syncing"}, {"idle", true, 0, 100, "up to date"}, {"scanning", true, 0, 100, "scanning"}, {"idle", true, 2, 100, "syncing"},
	} {
		if got := syncState(tc.state, tc.connected, tc.need, tc.remote); got != tc.want {
			t.Fatalf("%+v: %s", tc, got)
		}
	}
}

func TestAuthenticatedGranularAPI(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("X-API-Key") != "private" || r.Method != "PATCH" || r.URL.Path != "/rest/config/options" {
			t.Error("wrong request")
		}
		io.WriteString(w, `{}`)
	}))
	defer server.Close()
	a := syncAPI{base: server.URL, key: "private", client: server.Client()}
	if e := a.request(context.Background(), "PATCH", "config/options", map[string]bool{"natEnabled": false}, nil); e != nil || !called {
		t.Fatal(e)
	}
}

func testDevice(seed byte) string {
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	blocks := []string{}
	for b := 0; b < 4; b++ {
		data := strings.Repeat(string(seed), 13)
		sum, factor := 0, 1
		for i := 12; i >= 0; i-- {
			n := strings.IndexByte(alphabet, data[i]) * factor
			sum += n/32 + n%32
			factor = 3 - factor
		}
		blocks = append(blocks, data+string(alphabet[(32-sum%32)%32]))
	}
	raw := strings.Join(blocks, "")
	groups := []string{}
	for i := 0; i < len(raw); i += 7 {
		groups = append(groups, raw[i:i+7])
	}
	return strings.Join(groups, "-")
}

func TestSetupFreshRepeatDeferredAdoptionAndFailures(t *testing.T) {
	for _, mode := range []string{"fresh", "repeat", "deferred", "extra-peer", "decline-adoption", "conflict", "api-failure", "partial-failure", "self", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			config := filepath.Join(home, "config.xml")
			own, peer := testDevice('A'), testDevice('B')
			if !validDeviceID(peer) {
				t.Fatal("bad fixture")
			}
			folders := []map[string]any{}
			devices := []map[string]any{}
			writes := 0
			options := map[string]any{}
			folderWrites := 0
			if mode == "extra-peer" {
				folders = append(folders, map[string]any{"id": documentsID, "path": filepath.Join(home, "Documents"), "devices": []any{map[string]any{"deviceID": own}, map[string]any{"deviceID": testDevice('C')}}})
			}
			if mode == "conflict" {
				folders = append(folders, map[string]any{"id": "old", "path": filepath.Join(home, "Documents")})
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-API-Key") != "secret" {
					t.Error("missing authentication")
				}
				if mode == "api-failure" {
					http.Error(w, "no", 503)
					return
				}
				if r.Method != "GET" {
					writes++
				}
				var result any = map[string]any{}
				switch r.URL.Path {
				case "/rest/system/status":
					result = map[string]any{"myID": own}
				case "/rest/config/folders":
					if r.Method == "GET" {
						result = folders
					} else {
						var f map[string]any
						json.NewDecoder(r.Body).Decode(&f)
						folders = []map[string]any{f}
						folderWrites++
					}
				case "/rest/config/devices":
					if r.Method == "GET" {
						result = devices
					} else {
						var d map[string]any
						json.NewDecoder(r.Body).Decode(&d)
						devices = []map[string]any{d}
					}
				case "/rest/config/options":
					json.NewDecoder(r.Body).Decode(&options)
				case "/rest/config/defaults/device":
					if mode == "partial-failure" {
						http.Error(w, "fail", 500)
						return
					}
				}
				json.NewEncoder(w).Encode(result)
			}))
			defer server.Close()
			xmlConfig := `<configuration><gui><address>` + strings.TrimPrefix(server.URL, "http://") + `</address><apikey>secret</apikey></gui></configuration>`
			if mode != "fresh" {
				os.WriteFile(config, []byte(xmlConfig), 0600)
			}
			mgr := deploy.New(t.TempDir(), home)
			mgr.State = filepath.Join(home, "state")
			s := New(mgr, nil)
			s.Lookup = func(string) bool { return true }
			s.Manager.Run = func(_ context.Context, args ...string) (string, error) {
				switch args[0] {
				case "tailscale":
					return "100.64.0.1", nil
				case "syncthing":
					if args[1] == "paths" {
						return "Configuration file:\n    " + config, nil
					}
					if args[1] == "generate" {
						return "", os.WriteFile(config, []byte(xmlConfig), 0600)
					}
				}
				return "", nil
			}
			s.Terminal = func(context.Context, ...string) error { return nil }
			ask := func(prompt string) bool {
				if strings.Contains(prompt, "Explicitly adopt") && mode == "decline-adoption" {
					return false
				}
				return !strings.Contains(prompt, "lingering")
			}
			input := func(prompt string) string {
				if strings.Contains(prompt, "IPv4") {
					return "100.64.0.2"
				}
				switch mode {
				case "deferred":
					return ""
				case "self":
					return own
				case "invalid":
					return "invalid"
				}
				return peer
			}
			var output bytes.Buffer
			e := s.SyncthingSetup(context.Background(), &output, ask, input)
			if mode == "fresh" || mode == "repeat" || mode == "deferred" {
				if e != nil {
					t.Fatal(e, output.String())
				}
			} else if e == nil {
				t.Fatal("expected refusal")
			}
			if mode == "decline-adoption" || mode == "conflict" || mode == "api-failure" {
				if writes != 0 {
					t.Fatal("mutated before reconciliation")
				}
				return
			}
			if options["natEnabled"] != false || options["relaysEnabled"] != false || options["globalAnnounceEnabled"] != false || options["localAnnounceEnabled"] != false {
				t.Fatal(options)
			}
			addresses, _ := options["listenAddresses"].([]any)
			if len(addresses) != 1 || addresses[0] != "tcp://100.64.0.1:22000" {
				t.Fatal(options)
			}
			if mode == "deferred" {
				if folderWrites != 0 || !strings.Contains(output.String(), "Pairing pending") {
					t.Fatal(output.String())
				}
				return
			}
			if mode == "extra-peer" {
				if len(devices) != 0 {
					t.Fatal("changed devices before sharing reconciliation")
				}
				return
			}
			if mode == "partial-failure" || mode == "self" || mode == "invalid" {
				if folderWrites != 0 {
					t.Fatal("unexpected folder activation")
				}
				return
			}
			if !strings.Contains(output.String(), "peer offline") {
				t.Fatal("activation did not show the remaining connectivity step:", output.String())
			}
			f := folders[0]
			if f["id"] != documentsID || f["type"] != "sendreceive" || f["fsWatcherEnabled"] != true {
				t.Fatal(f)
			}
			d := devices[0]
			if d["introducer"] != false || d["autoAcceptFolders"] != false {
				t.Fatal(d)
			}
			if mode == "repeat" {
				if e = s.SyncthingSetup(context.Background(), &output, ask, input); e != nil {
					t.Fatal(e)
				}
				if len(folders) != 1 || len(devices) != 1 {
					t.Fatal("duplicated configuration")
				}
			}
		})
	}
}

func TestPeerMustShareBackBeforeUpToDate(t *testing.T) {
	for _, tc := range []struct{ remote, want string }{{"notSharing", "pairing pending"}, {"unknown", "pairing pending"}, {"paused", "paused"}, {"valid", ""}} {
		if got := remoteSyncState(tc.remote); got != tc.want {
			t.Fatal(tc, got)
		}
	}
}

func TestTailscaleManualDeferralAndAuthenticationFailure(t *testing.T) {
	for _, manual := range []bool{true, false} {
		t.Run(fmt.Sprint(manual), func(t *testing.T) {
			s := New(deploy.New(t.TempDir(), t.TempDir()), nil)
			s.Lookup = func(string) bool { return true }
			s.Manager.Run = func(context.Context, ...string) (string, error) { return "", errors.New("logged out") }
			calls := 0
			s.Terminal = func(_ context.Context, args ...string) error {
				calls++
				if args[0] == "tailscale" {
					return errors.New("permission denied")
				}
				return nil
			}
			var output bytes.Buffer
			e := s.SyncthingSetup(context.Background(), &output, func(string) bool { return !manual }, func(string) string { return "" })
			if e == nil {
				t.Fatal("accepted disconnected machine")
			}
			if manual {
				if calls != 0 || !strings.Contains(e.Error(), "sudo tailscale up") || !strings.Contains(e.Error(), "retained") {
					t.Fatal(calls, e)
				}
			} else {
				if calls != 2 || !strings.Contains(e.Error(), "authentication failed after tailscaled started") || !strings.Contains(e.Error(), "sudo tailscale up") {
					t.Fatal(calls, e)
				}
			}
		})
	}
}
