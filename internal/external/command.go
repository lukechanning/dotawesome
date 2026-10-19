// Package external centralizes subprocess execution and terminal ownership.
package external

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func Run(ctx context.Context, args ...string) (string, error) { return Env(ctx, nil, args...) }
func Env(ctx context.Context, env []string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Env = append(os.Environ(), env...)
	b, e := c.CombinedOutput()
	if e != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(args, " "), e, strings.TrimSpace(string(b)))
	}
	return strings.TrimSpace(string(b)), nil
}
func Terminal(ctx context.Context, args ...string) error {
	c := exec.CommandContext(ctx, args[0], args[1:]...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
