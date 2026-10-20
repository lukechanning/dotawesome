package app

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Opt-in: launches two real Syncthing instances with disposable homes and folders.
// It uses loopback transport to test transfer semantics; Tailscale is hardware acceptance.
func TestSyncthingTwoInstanceTransfer(t *testing.T) {
	if os.Getenv("DOTAWESOME_SYNCTHING_INTEGRATION") != "1" {
		t.Skip("set DOTAWESOME_SYNCTHING_INTEGRATION=1 with syncthing installed")
	}
	binary, e := exec.LookPath("syncthing")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	port := func() string {
		l, e := net.Listen("tcp4", "127.0.0.1:0")
		if e != nil {
			t.Fatal(e)
		}
		addr := l.Addr().String()
		l.Close()
		return addr
	}
	type instance struct {
		api              *syncAPI
		id, path, listen string
	}
	nodes := []instance{}
	for i := 0; i < 2; i++ {
		home := t.TempDir()
		cmd := exec.CommandContext(ctx, binary, "generate", "--home", home)
		if b, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("generate: %v: %s", e, b)
		}
		// An explicit --home makes config.xml a supported, isolated location.
		b, e := os.ReadFile(filepath.Join(home, "config.xml"))
		if e != nil {
			t.Fatal(e)
		}
		var cfg struct {
			GUI struct {
				Key string `xml:"apikey"`
			} `xml:"gui"`
		}
		if e = xml.Unmarshal(b, &cfg); e != nil {
			t.Fatal(e)
		}
		gui := port()
		log, e := os.Create(filepath.Join(home, "process.log"))
		if e != nil {
			t.Fatal(e)
		}
		process := exec.CommandContext(ctx, binary, "serve", "--home", home, "--gui-address", gui, "--no-browser", "--no-upgrade")
		process.Stdout = log
		process.Stderr = log
		if e = process.Start(); e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { process.Process.Kill(); process.Wait(); log.Close() })
		a := &syncAPI{base: "http://" + gui, key: cfg.GUI.Key, client: &http.Client{Timeout: time.Second}}
		var status struct{ MyID string }
		ready := false
		for j := 0; j < 60; j++ {
			if e = a.request(ctx, "GET", "system/status", nil, &status); e == nil {
				ready = true
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !ready {
			t.Fatal("instance did not start", e)
		}
		nodes = append(nodes, instance{api: a, id: status.MyID, path: filepath.Join(home, "Documents"), listen: port()})
	}
	for i, n := range nodes {
		peer := nodes[1-i]
		options := map[string]any{"listenAddresses": []string{"tcp://" + n.listen}, "globalAnnounceEnabled": false, "localAnnounceEnabled": false, "relaysEnabled": false, "natEnabled": false, "reconnectionIntervalS": 1}
		if e = n.api.request(ctx, "PATCH", "config/options", options, nil); e != nil {
			t.Fatal(e)
		}
		var device map[string]any
		if e = n.api.request(ctx, "GET", "config/defaults/device", nil, &device); e != nil {
			t.Fatal(e)
		}
		device["deviceID"] = peer.id
		device["addresses"] = []string{"tcp://" + peer.listen}
		device["introducer"] = false
		device["autoAcceptFolders"] = false
		if e = n.api.request(ctx, "POST", "config/devices", device, nil); e != nil {
			t.Fatal(e)
		}
		os.MkdirAll(n.path, 0700)
		var folder map[string]any
		if e = n.api.request(ctx, "GET", "config/defaults/folder", nil, &folder); e != nil {
			t.Fatal(e)
		}
		folder["id"] = documentsID
		folder["path"] = n.path
		folder["type"] = "sendreceive"
		folder["fsWatcherEnabled"] = true
		folder["devices"] = []map[string]string{{"deviceID": n.id}, {"deviceID": peer.id}}
		folder["versioning"] = map[string]any{"type": "staggered", "params": map[string]string{"maxAge": "7776000"}, "cleanupIntervalS": 3600}
		if e = n.api.request(ctx, "POST", "config/folders", folder, nil); e != nil {
			t.Fatal(e)
		}
	}
	await := func(label string, predicate func() bool) {
		t.Helper()
		for j := 0; j < 300; j++ {
			if predicate() {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}
		t.Fatal("timeout:", label)
	}
	matches := func(path, want string) bool { b, e := os.ReadFile(path); return e == nil && string(b) == want }
	scan := func(i int) {
		if e := nodes[i].api.request(ctx, "POST", "db/scan?folder="+documentsID, nil, nil); e != nil {
			t.Fatal(e)
		}
	}
	for i := 0; i < 2; i++ {
		name := fmt.Sprintf("from-%d.txt", i)
		os.WriteFile(filepath.Join(nodes[i].path, name), []byte("first"), 0600)
		scan(i)
		await("bidirectional transfer", func() bool { return matches(filepath.Join(nodes[1-i].path, name), "first") })
	}
	os.WriteFile(filepath.Join(nodes[0].path, "from-0.txt"), []byte("second"), 0600)
	scan(0)
	await("replacement", func() bool { return matches(filepath.Join(nodes[1].path, "from-0.txt"), "second") })
	os.Remove(filepath.Join(nodes[0].path, "from-0.txt"))
	scan(0)
	await("deletion", func() bool { _, e := os.Stat(filepath.Join(nodes[1].path, "from-0.txt")); return os.IsNotExist(e) })
	await("version retained", func() bool {
		found := false
		filepath.Walk(filepath.Join(nodes[1].path, ".stversions"), func(path string, info os.FileInfo, e error) error {
			if e == nil && !info.IsDir() && strings.Contains(info.Name(), "from-0") {
				b, _ := os.ReadFile(path)
				if string(b) == "first" || string(b) == "second" {
					found = true
				}
			}
			return nil
		})
		return found
	})
}
