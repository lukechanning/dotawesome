package cli

import (
	"bytes"
	"context"
	"fmt"
	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestTerminalLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("PTY scenarios")
	}
	repo, _ := filepath.Abs("../..")
	binary := filepath.Join(t.TempDir(), "dotawesome")
	build := exec.Command("go", "build", "-o", binary, "./cmd/dotawesome")
	build.Dir = repo
	if b, e := build.CombinedOutput(); e != nil {
		t.Fatal(e, string(b))
	}
	for _, scenario := range []string{"startup-resize-exit", "persistent-completion", "external-handoff", "safe-cancellation", "machine-readiness", "dirty-sync"} {
		t.Run(scenario, func(t *testing.T) {
			home := t.TempDir()
			tools := filepath.Join(home, "tools")
			os.Mkdir(tools, 0755)
			fake := "#!/bin/sh\nexit 0\n"
			if scenario == "safe-cancellation" {
				fake = "#!/bin/sh\ncase \"$*\" in *start-server*) sleep 1;; esac\nexit 0\n"
			}
			os.WriteFile(filepath.Join(tools, "tmux"), []byte(fake), 0755)
			if scenario == "dirty-sync" {
				os.WriteFile(filepath.Join(tools, "git"), []byte("#!/bin/sh\ncase \"$*\" in *status*) echo \" M home/config\";; *) echo main;; esac\n"), 0755)
			}
			master, slave, e := pty.Open()
			if e != nil {
				t.Fatal(e)
			}
			defer master.Close()
			defer slave.Close()
			pty.Setsize(master, &pty.Winsize{Cols: 100, Rows: 35})
			before, e := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			c := exec.CommandContext(ctx, binary)
			c.Dir = repo
			c.Env = append(os.Environ(), "DOTAWESOME_ROOT="+repo, "HOME="+home, "XDG_CONFIG_HOME="+filepath.Join(home, ".config"), "XDG_STATE_HOME="+filepath.Join(home, "state"), "XDG_DATA_HOME="+filepath.Join(home, "data"), "XDG_CACHE_HOME="+filepath.Join(home, "cache"), "TERM=xterm-256color", "NO_COLOR=1", "PATH="+tools+":"+os.Getenv("PATH"))
			c.Stdin = slave
			c.Stdout = slave
			c.Stderr = slave
			c.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 0}
			if e = c.Start(); e != nil {
				t.Fatal(e)
			}
			defer func() {
				if c.ProcessState == nil {
					c.Process.Kill()
					c.Wait()
				}
			}()
			chunks := make(chan []byte, 256)
			go func() {
				defer close(chunks)
				b := make([]byte, 16384)
				for {
					n, e := master.Read(b)
					if n > 0 {
						copyB := append([]byte{}, b[:n]...)
						chunks <- copyB
					}
					if e != nil {
						return
					}
				}
			}()
			var output bytes.Buffer
			send := func(s string) {
				t.Helper()
				if _, e := master.Write([]byte(s)); e != nil {
					t.Fatal(e)
				}
			}
			wait := func(want string) {
				t.Helper()
				deadline := time.NewTimer(8 * time.Second)
				defer deadline.Stop()
				for !strings.Contains(output.String(), want) {
					select {
					case b := <-chunks:
						output.Write(b)
						if bytes.Contains(b, []byte("\x1b[6n")) {
							send("\x1b[1;1R")
						}
					case <-deadline.C:
						t.Fatalf("timeout waiting for %q:\n%s", want, output.String())
					}
				}
			}
			wait("DOTAWESOME")
			switch scenario {
			case "startup-resize-exit":
				pty.Setsize(master, &pty.Winsize{Cols: 40, Rows: 18})
				c.Process.Signal(syscall.SIGWINCH)
				send("?")
				wait("Arrows navigate")
				send("q")
			case "external-handoff":
				send("\t\t\t\r")
				wait("More options")
				send("\t\t\r")
				wait("Enter to continue, or Escape to go back.")
				send("\r")
				wait("Enroll a dedicated age identity")
				send("no\r")
				wait("Results remain here")
				send("\x1b")
				time.Sleep(100 * time.Millisecond)
				send("q")
			case "machine-readiness":
				send("\r")
				wait("Tools")
				wait("Choose G14 or desktop")
				if _, e := os.Lstat(filepath.Join(home, ".config")); !os.IsNotExist(e) {
					t.Fatal("readiness check changed home")
				}
				send("\x1b")
				time.Sleep(100 * time.Millisecond)
				send("q")
			case "dirty-sync":
				send("\t\t\r")
				wait("Inspect setup issues")
				send("\r")
				wait("local changes")
				send("\x1b")
				time.Sleep(100 * time.Millisecond)
				send("\x1b")
				time.Sleep(100 * time.Millisecond)
				send("q")
			default:
				send("\t\r")
				wait("Component configuration")
				send("\x1b[B\x1b[B\r")
				wait("tmux —")
				send("\r")
				wait("Review changes")
				send("\r")
				wait("Confirm")
				send("\r")
				if scenario == "safe-cancellation" {
					wait("Executing")
					send("\x1b")
					wait("Safe stop requested")
				}
				wait("Results remain here")
				send("\r")
				time.Sleep(100 * time.Millisecond)
				send("q")
			}
			if e = c.Wait(); e != nil {
				t.Fatal(e, output.String())
			}
			after, e := unix.IoctlGetTermios(int(slave.Fd()), unix.TCGETS)
			if e != nil {
				t.Fatal(e)
			}
			if before.Lflag != after.Lflag || before.Iflag != after.Iflag || before.Oflag != after.Oflag {
				t.Fatal(fmt.Sprintf("terminal not restored: before=%+v after=%+v", before, after))
			}
		})
	}
}
