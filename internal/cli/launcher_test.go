package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLauncherNeverRunsStaleBinaryAfterBuildFailure(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "bin"), 0755)
	os.MkdirAll(filepath.Join(root, ".build/bin"), 0755)
	b, e := os.ReadFile("../../bin/dotawesome")
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(root, "bin/dotawesome"), b, 0755)
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture"), 0644)
	os.WriteFile(filepath.Join(root, ".build/bin/dotawesome"), []byte("#!/bin/sh\necho STALE\n"), 0755)
	fake := filepath.Join(root, "tools")
	os.Mkdir(fake, 0755)
	os.WriteFile(filepath.Join(fake, "go"), []byte("#!/bin/sh\necho 'compile failed' >&2\nexit 1\n"), 0755)
	c := exec.Command(filepath.Join(root, "bin/dotawesome"), "list")
	c.Env = append(os.Environ(), "PATH="+fake+":"+os.Getenv("PATH"))
	out, e := c.CombinedOutput()
	if e == nil || strings.Contains(string(out), "STALE") || !strings.Contains(string(out), "compile failed") {
		t.Fatal(e, string(out))
	}
}
