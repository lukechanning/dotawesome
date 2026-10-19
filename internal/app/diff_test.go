package app

import (
	"dotawesome/internal/deploy"
	"strings"
	"testing"
)

func TestUnifiedDiffShowsChangedLines(t *testing.T) {
	d, a, r := Unified("file", deploy.File([]byte("one\ntwo\n"), 0644), deploy.File([]byte("one\nthree\n"), 0644))
	if a != 1 || r != 1 || !strings.Contains(d, "-two\n+three") || !strings.Contains(d, "@@") {
		t.Fatal(d, a, r)
	}
}
func TestBinaryDiffDoesNotRenderExecutable(t *testing.T) {
	d, _, _ := Unified("work", nil, deploy.File([]byte{0, 1, 2, 27}, 0755))
	if !strings.Contains(d, "Binary") || strings.ContainsRune(d, 0) {
		t.Fatal(d)
	}
}
