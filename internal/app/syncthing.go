package app

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const documentsID = "dotawesome-documents"

func validTailIP(s string) bool {
	ip := net.ParseIP(s)
	_, network, _ := net.ParseCIDR("100.64.0.0/10")
	return ip != nil && ip.To4() != nil && network.Contains(ip)
}

// Syncthing device IDs contain four independently checked Luhn32 blocks.
func validDeviceID(s string) bool {
	s = strings.ReplaceAll(strings.ToUpper(s), "-", "")
	if len(s) != 56 {
		return false
	}
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	for block := 0; block < 4; block++ {
		sum := 0
		factor := 1
		for i := block*14 + 12; i >= block*14; i-- {
			n := strings.IndexByte(alphabet, s[i])
			if n < 0 {
				return false
			}
			n *= factor
			sum += n/32 + n%32
			factor = 3 - factor
		}
		if strings.IndexByte(alphabet, s[block*14+13]) != (32-sum%32)%32 {
			return false
		}
	}
	return true
}
func validateDocuments(folders []map[string]any, home string) error {
	path := filepath.Join(home, "Documents")
	for _, f := range folders {
		id, _ := f["id"].(string)
		p, _ := f["path"].(string)
		if strings.HasPrefix(p, "~/") {
			p = filepath.Join(home, p[2:])
		}
		if (id == documentsID && filepath.Clean(p) != path) || (filepath.Clean(p) == path && id != documentsID) {
			return errors.New("Documents already has a different folder ID or mapping; reconcile it in the loopback administration interface before retrying")
		}
	}
	return nil
}
func syncState(state string, connected bool, need, completion float64) string {
	if state == "error" {
		return "folder errors"
	}
	if strings.Contains(state, "scan") {
		return "scanning"
	}
	if !connected {
		return "peer offline"
	}
	if state != "idle" || need != 0 || completion != 100 {
		return "syncing"
	}
	return "up to date"
}

func remoteSyncState(remote string) string {
	switch remote {
	case "valid":
		return ""
	case "paused":
		return "paused"
	default:
		return "pairing pending"
	}
}

type syncAPI struct {
	base, key string
	client    *http.Client
}

func (a *syncAPI) request(ctx context.Context, method, path string, body, result any) error {
	var b bytes.Buffer
	if body != nil {
		if e := json.NewEncoder(&b).Encode(body); e != nil {
			return e
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, method, a.base+"/rest/"+path, &b)
	if e != nil {
		return e
	}
	req.Header.Set("X-API-Key", a.key)
	req.Header.Set("Content-Type", "application/json")
	resp, e := a.client.Do(req)
	if e != nil {
		return errors.New("Syncthing API unavailable; check journalctl --user -u syncthing and rerun bin/dotawesome syncthing setup")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Syncthing API %s %s returned %d", method, path, resp.StatusCode)
	}
	if result != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(result)
	}
	return nil
}
func (s *Service) localSyncAPI(ctx context.Context) (*syncAPI, error) {
	paths, e := s.Manager.Run(ctx, "syncthing", "paths")
	if e != nil {
		return nil, e
	}
	config := ""
	lines := strings.Split(paths, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Configuration file:") {
			config = strings.TrimSpace(strings.TrimPrefix(line, "Configuration file:"))
			if config == "" && i+1 < len(lines) {
				config = strings.TrimSpace(lines[i+1])
			}
			break
		}
	}
	if config == "" {
		return nil, errors.New("cannot discover Syncthing configuration with syncthing paths")
	}
	b, e := os.ReadFile(config)
	if e != nil {
		return nil, e
	}
	var c struct {
		GUI struct {
			Address string `xml:"address"`
			Key     string `xml:"apikey"`
			TLS     bool   `xml:"tls,attr"`
		} `xml:"gui"`
	}
	if e = xml.Unmarshal(b, &c); e != nil {
		return nil, e
	}
	host, _, e := net.SplitHostPort(c.GUI.Address)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || c.GUI.TLS || c.GUI.Key == "" {
		return nil, errors.New("adoption requires an authenticated HTTP GUI on loopback; preserve existing settings and reconcile locally first")
	}
	return &syncAPI{base: "http://" + c.GUI.Address, key: c.GUI.Key, client: &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }}}, nil
}
func (s *Service) tailIP(ctx context.Context) (string, error) {
	out, e := s.Manager.Run(ctx, "tailscale", "ip", "-4")
	ip := strings.TrimSpace(out)
	if e != nil || !validTailIP(ip) {
		return "", errors.New("Tailscale is not connected. To connect manually, check systemctl status tailscaled.service, start it if needed with sudo systemctl enable --now tailscaled.service, then run sudo tailscale up and tailscale status. Rerun bin/dotawesome syncthing setup once connected; completed preparation is retained.")
	}
	return ip, nil
}
func (s *Service) SyncthingStatus(ctx context.Context, out io.Writer) error {
	for _, tool := range []string{"syncthing", "tailscale"} {
		if !s.Lookup(tool) {
			fmt.Fprintln(out, "missing tools:", tool)
			return nil
		}
	}
	if _, e := s.tailIP(ctx); e != nil {
		fmt.Fprintln(out, e)
		return nil
	}
	if _, e := s.Manager.Run(ctx, "systemctl", "--user", "is-active", "syncthing.service"); e != nil {
		fmt.Fprintln(out, "service stopped; run bin/dotawesome syncthing setup")
		return nil
	}
	a, e := s.localSyncAPI(ctx)
	if e != nil {
		return e
	}
	fmt.Fprintln(out, "Administration:", a.base, "\nDiagnostics: journalctl --user -u syncthing")
	var folders []map[string]any
	if e = a.request(ctx, "GET", "config/folders", nil, &folders); e != nil {
		return e
	}
	if e = validateDocuments(folders, s.Manager.Home); e != nil {
		return e
	}
	for _, f := range folders {
		if f["id"] != documentsID {
			continue
		}
		if f["paused"] == true {
			fmt.Fprintln(out, "paused")
			return nil
		}
		devices, _ := f["devices"].([]any)
		peer := ""
		var sys struct{ MyID string }
		if e = a.request(ctx, "GET", "system/status", nil, &sys); e != nil {
			return e
		}
		for _, d := range devices {
			m, _ := d.(map[string]any)
			id, _ := m["deviceID"].(string)
			if id != sys.MyID {
				peer = id
			}
		}
		if peer == "" {
			fmt.Fprintln(out, "pairing pending; run bin/dotawesome syncthing setup on both machines")
			return nil
		}
		var configured []map[string]any
		if e = a.request(ctx, "GET", "config/devices", nil, &configured); e != nil {
			return e
		}
		for _, d := range configured {
			if d["deviceID"] == peer && d["paused"] == true {
				fmt.Fprintln(out, "paused (peer)")
				return nil
			}
		}

		var connections struct {
			Connections map[string]struct{ Connected bool }
		}
		if e = a.request(ctx, "GET", "system/connections", nil, &connections); e != nil {
			return e
		}
		var status struct {
			State          string
			NeedTotalItems float64
			Error          string
			PullErrors     int
		}
		if e = a.request(ctx, "GET", "db/status?folder="+documentsID, nil, &status); e != nil {
			return e
		}
		if status.Error != "" || status.PullErrors > 0 {
			fmt.Fprintln(out, "folder errors:", status.Error)
			return nil
		}
		var completion struct {
			Completion             float64
			NeedItems, NeedDeletes float64
			RemoteState            string
		}
		if connections.Connections[peer].Connected {
			if e = a.request(ctx, "GET", "db/completion?folder="+documentsID+"&device="+url.QueryEscape(peer), nil, &completion); e != nil {
				return e
			}
		}
		if connections.Connections[peer].Connected {
			if state := remoteSyncState(completion.RemoteState); state != "" {
				fmt.Fprintln(out, state, "— complete or resume Documents sharing on the other machine")
				return nil
			}
		}

		fmt.Fprintf(out, "%s (local remaining %.0f; peer %.1f%%)\n", syncState(status.State, connections.Connections[peer].Connected, status.NeedTotalItems+completion.NeedItems+completion.NeedDeletes, completion.Completion), status.NeedTotalItems, completion.Completion)
		fmt.Fprintln(out, "If offline: complete pairing on both machines and check tailnet/firewall permission for TCP 22000. No policies were changed.")
		return nil
	}
	fmt.Fprintln(out, "pairing pending: run bin/dotawesome syncthing setup")
	return nil
}

// SyncthingSetup prepares local service first; peer information may be deferred.
func (s *Service) SyncthingSetup(ctx context.Context, out io.Writer, ask func(string) bool, input func(string) string) error {
	fmt.Fprintln(out, "Documents setup: complete this guide on BOTH machines. Completed preparation is reused.")
	missing := false
	for _, t := range []string{"syncthing", "tailscale"} {
		if !s.Lookup(t) {
			missing = true
		}
	}
	if missing {
		if !ask("Install Syncthing and Tailscale with sudo pacman -Syu --needed (full system upgrade)?") {
			return errors.New("missing tools; rerun setup to continue")
		}
		if e := s.InstallPackages(ctx, map[string][]string{"syncthing": {"syncthing", "tailscale"}}); e != nil {
			return e
		}
	}
	ip, e := s.tailIP(ctx)
	if e != nil {
		if !ask("Start tailscaled and authenticate Tailscale interactively, preserving existing membership/settings?") {
			return e
		}
		if e = s.Terminal(ctx, "sudo", "systemctl", "enable", "--now", "tailscaled.service"); e != nil {
			return fmt.Errorf("tailscaled service startup failed: %w. Inspect systemctl status tailscaled.service and journalctl -u tailscaled.service -n 50 --no-pager; start it manually, then rerun bin/dotawesome syncthing setup", e)
		}
		if e = s.Terminal(ctx, "tailscale", "up"); e != nil {
			return fmt.Errorf("Tailscale authentication failed after tailscaled started: %w. Run sudo tailscale up manually, check tailscale status, then rerun bin/dotawesome syncthing setup", e)
		}
		ip, e = s.tailIP(ctx)
		if e != nil {
			return e
		}
	}
	paths, e := s.Manager.Run(ctx, "syncthing", "paths")
	if e != nil {
		return e
	}
	existing := false
	for _, line := range strings.Split(paths, "\n") {
		p := strings.TrimSpace(line)
		if strings.HasSuffix(p, "config.xml") {
			p = strings.TrimSpace(strings.TrimPrefix(p, "Configuration file:"))
			if _, err := os.Stat(p); err == nil {
				existing = true
			}
		}
	}
	if existing && !ask("Explicitly adopt this existing Syncthing installation? Transport settings will change globally; unrelated devices, folders and ignores are preserved.") {
		return errors.New("adoption declined")
	}
	if !ask("Prepare local identity and enable the packaged Syncthing user service at login?") {
		return errors.New("local preparation pending")
	}
	if !existing {
		if _, e = s.Manager.Run(ctx, "syncthing", "generate"); e != nil {
			return e
		}
	}
	if e = s.Terminal(ctx, "systemctl", "--user", "enable", "--now", "syncthing.service"); e != nil {
		return e
	}
	var a *syncAPI
	for attempt := 0; attempt < 10; attempt++ {
		a, e = s.localSyncAPI(ctx)
		if e == nil {
			var ping map[string]any
			e = a.request(ctx, "GET", "system/ping", nil, &ping)
		}
		if e == nil {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	if e != nil {
		return e
	}
	var sys struct{ MyID string }
	if e = a.request(ctx, "GET", "system/status", nil, &sys); e != nil {
		return e
	}
	fmt.Fprintf(out, "This device: %s\nTailscale IPv4: %s\nAdministration: %s\n", sys.MyID, ip, a.base)
	var folders []map[string]any
	if e = a.request(ctx, "GET", "config/folders", nil, &folders); e != nil {
		return e
	}
	if e = validateDocuments(folders, s.Manager.Home); e != nil {
		return e
	}
	if !ask("Bind Syncthing to this Tailscale IPv4 and disable discovery, relays and NAT traversal? This applies globally to this installation.") {
		return errors.New("transport configuration pending; rerun setup")
	}

	options := map[string]any{"listenAddresses": []string{"tcp://" + ip + ":22000"}, "globalAnnounceEnabled": false, "localAnnounceEnabled": false, "relaysEnabled": false, "natEnabled": false}
	if e = a.request(ctx, "PATCH", "config/options", options, nil); e != nil {
		return e
	}
	if !existing {
		for _, f := range folders {
			if f["id"] == "default" {
				ds, _ := f["devices"].([]any)
				unshared := true
				for _, v := range ds {
					d, _ := v.(map[string]any)
					if d["deviceID"] != sys.MyID {
						unshared = false
					}
				}
				if unshared {
					if e = a.request(ctx, "DELETE", "config/folders/default", nil, nil); e != nil {
						return e
					}
				}
			}
		}
	}

	if e = s.RecordRequirements(ctx, "syncthing", []string{"syncthing", "tailscale"}); e != nil {
		return e
	}

	peer := strings.ToUpper(strings.TrimSpace(input("Other machine device ID (blank defers pairing): ")))
	if peer == "" {
		fmt.Fprintln(out, "Local preparation retained. Pairing pending; rerun this guide with the other machine's ID and Tailscale IPv4.")
		return nil
	}
	if !validDeviceID(peer) || strings.ReplaceAll(peer, "-", "") == strings.ReplaceAll(sys.MyID, "-", "") {
		return errors.New("invalid or self peer device ID; local preparation retained")
	}
	rawPeer := strings.ReplaceAll(peer, "-", "")
	groups := []string{}
	for i := 0; i < len(rawPeer); i += 7 {
		groups = append(groups, rawPeer[i:i+7])
	}
	peer = strings.Join(groups, "-")

	peerIP := strings.TrimSpace(input("Other machine Tailscale IPv4: "))
	if !validTailIP(peerIP) || peerIP == ip {
		return errors.New("invalid or self Tailscale IPv4")
	}
	fmt.Fprintf(out, "Review: %s ↔ %s at tcp://%s:22000\nFolder ID: %s; two-way; filesystem watching; staggered versions, maximum 90 days.\nInitial folders MERGE. Later deletions propagate. Simultaneous edits can create conflict copies. Incoming changes archive previous versions; this is not a complete backup.\n", filepath.Join(s.Manager.Home, "Documents"), peer, peerIP, documentsID)
	if !ask("Enable this Documents sharing and Tailscale-only Syncthing transport?") {
		return errors.New("activation deferred; rerun setup")
	}
	// Inspect again immediately before granular mutations.
	if e = a.request(ctx, "GET", "config/folders", nil, &folders); e != nil {
		return e
	}
	if e = validateDocuments(folders, s.Manager.Home); e != nil {
		return e
	}
	var folder map[string]any
	for _, f := range folders {
		if f["id"] == documentsID {
			folder = f
		}
	}
	if folder == nil {
		if e = a.request(ctx, "GET", "config/defaults/folder", nil, &folder); e != nil {
			return e
		}
	}
	// Do not silently drop additional existing shares.
	if ds, ok := folder["devices"].([]any); ok {
		for _, v := range ds {
			d, _ := v.(map[string]any)
			if id := d["deviceID"]; id != nil && id != peer && id != sys.MyID {
				return errors.New("Documents has additional peers; reconcile sharing before retrying")
			}
		}
	}
	var device map[string]any
	var devices []map[string]any
	if e = a.request(ctx, "GET", "config/devices", nil, &devices); e != nil {
		return e
	}
	for _, d := range devices {
		if d["deviceID"] == peer {
			device = d
		}
	}
	if device == nil {
		if e = a.request(ctx, "GET", "config/defaults/device", nil, &device); e != nil {
			return e
		}
	}
	device["deviceID"] = peer
	device["addresses"] = []string{"tcp://" + peerIP + ":22000"}
	device["introducer"] = false
	device["autoAcceptFolders"] = false
	if e = a.request(ctx, "POST", "config/devices", device, nil); e != nil {
		return e
	}
	folder["id"] = documentsID
	folder["label"] = "Documents"
	folder["path"] = filepath.Join(s.Manager.Home, "Documents")
	folder["type"] = "sendreceive"
	folder["fsWatcherEnabled"] = true
	folder["devices"] = []map[string]string{{"deviceID": sys.MyID}, {"deviceID": peer}}
	folder["versioning"] = map[string]any{"type": "staggered", "params": map[string]string{"maxAge": "7776000"}, "cleanupIntervalS": 3600}
	folder["paused"] = false
	if e = os.MkdirAll(filepath.Join(s.Manager.Home, "Documents"), 0700); e != nil {
		return e
	}
	if e = a.request(ctx, "POST", "config/folders", folder, nil); e != nil {
		return e
	}
	var restart struct{ RequiresRestart bool }
	if e = a.request(ctx, "GET", "config/restart-required", nil, &restart); e != nil {
		return e
	}
	if restart.RequiresRestart {
		if e = a.request(ctx, "POST", "system/restart", nil, nil); e != nil {
			return e
		}
	}
	if ask("Enable lingering for this desktop user, allowing syncing before login and after logout? Power management is unchanged.") {
		if e = s.Terminal(ctx, "loginctl", "enable-linger"); e != nil {
			return e
		}
	}
	fmt.Fprintln(out, "Sharing configured. Complete the same guide on the other machine. Check progress with bin/dotawesome syncthing status.\nVerify reconnection after Tailscale becomes available; inspect journalctl --user -u syncthing if TCP 22000 does not recover.")
	return s.SyncthingStatus(ctx, out)
}
