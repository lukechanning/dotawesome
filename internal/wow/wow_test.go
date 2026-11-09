package wow_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(dir, "..", ".."))
}

func run(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(filepath.Join(repoRoot(t), "bin", "wow"), args...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestRefusesStorageOutsideHive(t *testing.T) {
	out, err := run(t, []string{"WOW_ROOT=/tmp/unsafe-wow"}, "layout")
	if err == nil {
		t.Fatalf("layout unexpectedly accepted an unsafe root: %s", out)
	}
	if !strings.Contains(out, "must be beneath /mnt/hive") {
		t.Fatalf("unexpected error: %s", out)
	}
}

func TestLayoutCreatesEveryPersistentDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "WoW")
	out, err := run(t, []string{"WOW_ROOT=" + root, "WOW_TEST_MODE=1"}, "layout")
	if err != nil {
		t.Fatalf("layout failed: %v\n%s", err, out)
	}
	for _, rel := range []string{
		"client/downloads", "client/base", "client/hd", "client/retail-cache", "client/patches",
		"server/database", "server/backups", "server/logs", "server/config",
		"source", "build", "docker", "tmp",
	} {
		info, err := os.Stat(filepath.Join(root, rel))
		if err != nil || !info.IsDir() {
			t.Errorf("missing directory %s", rel)
		}
	}
}

func TestHelpListsAdministrationCommands(t *testing.T) {
	out, err := run(t, nil, "help")
	if err != nil {
		t.Fatalf("help failed: %v\n%s", err, out)
	}
	for _, command := range []string{
		"start", "stop", "status", "play", "backup", "restore",
		"client-fetch", "client-build", "client-rollback",
	} {
		if !strings.Contains(out, command) {
			t.Errorf("help does not list %q", command)
		}
	}
}

func TestComposeKeepsPersistentDataOnHive(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "components", "wow", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	compose := string(b)
	for _, required := range []string{
		"/mnt/hive/WoW/server/database:/var/lib/mysql",
		"/mnt/hive/WoW/server/config:/azerothcore/env/dist/etc",
		"/mnt/hive/WoW/server/logs:/azerothcore/env/dist/logs",
		"${WOW_TAILSCALE_IP}:3724:3724",
		"${WOW_TAILSCALE_IP}:8085:8085",
		"--disable-log-bin",
	} {
		if !strings.Contains(compose, required) {
			t.Errorf("compose is missing storage/network invariant %q", required)
		}
	}
	for _, forbidden := range []string{"ac-database:", "ac-client-data:"} {
		if strings.Contains(compose, "volumes:\n  "+forbidden) {
			t.Errorf("compose declares anonymous/named persistent volume %q", forbidden)
		}
	}
	if strings.Contains(compose, "3306:3306") {
		t.Error("MySQL must remain internal to the Compose network")
	}
}

func TestRootlessBindWritersUseHostMappedContainerRoot(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "components", "wow", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// In rootless Docker, container UID 0 maps to the invoking host user. The
	// upstream acore UID 1000 maps to a subordinate UID and cannot write our
	// Luke-owned /mnt/hive bind mounts.
	if got := strings.Count(string(b), "user: \"0:0\""); got < 4 {
		t.Fatalf("only %d bind-writing services use the rootless host mapping; want at least 4", got)
	}
}

func TestRepositoryContainsNoRealmCredentials(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "components", "wow", "realm.env.example"))
	if err != nil {
		t.Fatal(err)
	}
	env := string(b)
	for _, placeholder := range []string{
		"WOW_TAILSCALE_IP=CHANGE_ME",
		"WOW_DB_ROOT_PASSWORD=CHANGE_ME",
		"OPENROUTER_API_KEY=CHANGE_ME",
	} {
		if !strings.Contains(env, placeholder) {
			t.Errorf("example environment is missing %q", placeholder)
		}
	}
}

func TestClientArchiveMustBeSuppliedByOperator(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "components", "wow", "wow-client-fetch"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(b)
	if !strings.Contains(script, "WOW_CLIENT_ARCHIVE") {
		t.Error("client fetch must require an operator-supplied archive path")
	}
	if strings.Contains(script, "ChromieCraft") {
		t.Error("client fetch must not name or promote a third-party client distribution")
	}
}

func TestComponentDocumentsAssetAndSecretBoundary(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "components", "wow", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	for _, required := range []string{
		"does not contain or distribute Blizzard software or game assets",
		"lawfully obtained compatible client",
		"realm.env",
		"database backups",
	} {
		if !strings.Contains(doc, required) {
			t.Errorf("component documentation is missing %q", required)
		}
	}
}

func TestUserServicesAreIndependentFromGaming(t *testing.T) {
	for _, name := range []string{"radium-wow-docker.service", "radium-wow.service"} {
		b, err := os.ReadFile(filepath.Join(repoRoot(t), "home", "dot_config", "systemd", "user", name))
		if err != nil {
			t.Fatal(err)
		}
		unit := string(b)
		if strings.Contains(unit, "PartOf=radium-gaming.service") {
			t.Errorf("%s must remain independent from radium-gaming", name)
		}
	}
}

func TestFishCommandsDelegateToSingleAdminCLI(t *testing.T) {
	for _, name := range []string{
		"wow-start", "wow-stop", "wow-status", "wow-play", "wow-backup", "wow-restore",
		"wow-client-fetch", "wow-client-build", "wow-client-rollback",
	} {
		path := filepath.Join(repoRoot(t), "home", "dot_config", "fish", "functions", name+".fish")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !strings.Contains(string(b), "bin/wow "+strings.TrimPrefix(name, "wow-")) {
			t.Errorf("%s does not delegate to bin/wow", name)
		}
	}
}
