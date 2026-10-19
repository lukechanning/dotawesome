package app

import (
	"bytes"
	"dotawesome/internal/deploy"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Unified emits one full-file hunk, retaining unchanged lines for context.
func Unified(path string, before, after *deploy.Value) (string, int, int) {
	aBytes, bBytes := before.Bytes(), after.Bytes()
	if bytes.IndexByte(aBytes, 0) >= 0 || bytes.IndexByte(bBytes, 0) >= 0 || !utf8.Valid(aBytes) || !utf8.Valid(bBytes) {
		return fmt.Sprintf("Binary file %s changes (%d → %d bytes)\n", path, len(aBytes), len(bBytes)), 0, 0
	}
	split := func(v *deploy.Value) []string {
		s := string(v.Bytes())
		if s == "" {
			return nil
		}
		return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	}
	a, b := split(before), split(after)
	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s (planned)\n@@ -1,%d +1,%d @@\n", path, path, len(a), len(b))
	added, removed := 0, 0
	if len(a)*len(b) > 4000000 {
		for _, s := range a {
			out.WriteString("-" + s + "\n")
			removed++
		}
		for _, s := range b {
			out.WriteString("+" + s + "\n")
			added++
		}
		return out.String(), added, removed
	}
	dp := make([][]int, len(a)+1)
	for i := range dp {
		dp[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		if i < len(a) && j < len(b) && a[i] == b[j] {
			out.WriteString(" " + a[i] + "\n")
			i++
			j++
		} else if i < len(a) && (j == len(b) || dp[i+1][j] >= dp[i][j+1]) {
			out.WriteString("-" + a[i] + "\n")
			removed++
			i++
		} else {
			out.WriteString("+" + b[j] + "\n")
			added++
			j++
		}
	}
	if before != nil && after != nil && before.Mode != nil && after.Mode != nil && *before.Mode != *after.Mode {
		fmt.Fprintf(&out, "mode %04o -> %04o\n", *before.Mode, *after.Mode)
	}
	return out.String(), added, removed
}
