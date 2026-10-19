// Package cli keeps explicit commands and the interactive application on shared services.
package cli

import (
	"bufio"
	"context"
	"dotawesome/internal/app"
	"dotawesome/internal/components"
	"dotawesome/internal/deploy"
	"dotawesome/internal/tui"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Run(ctx context.Context, s *app.Service, args []string, in io.Reader, out, stderr io.Writer) int {
	fail := func(e error) int {
		fmt.Fprintln(stderr, "dotawesome:", e)
		if errors.Is(e, context.Canceled) {
			return 130
		}
		return 1
	}
	scan := bufio.NewScanner(in)
	ask := func(prompt string) bool {
		fmt.Fprint(out, prompt+" [yes/no]: ")
		return scan.Scan() && strings.EqualFold(strings.TrimSpace(scan.Text()), "yes")
	}
	if len(args) == 0 || args[0] == "setup" {
		if len(args) > 1 {
			return fail(errors.New("setup does not accept arguments"))
		}
		returnCode := tui.Run(s)
		if returnCode != nil {
			return fail(returnCode)
		}
		return 0
	}
	if args[0] == "sync" {
		pulled := len(args) == 2 && args[1] == "--after-pull"
		if len(args) > 1 && !pulled {
			fmt.Fprintln(stderr, "Usage: bin/dotawesome sync")
			return 2
		}
		if e := tui.RunFlow(s, "sync", pulled); e != nil {
			return fail(e)
		}
		return 0
	}
	command := args[0]
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	var profile, machine, action, receipt string
	var live, remove, installPackages, removePackages, skipHooks, staged bool
	switch command {
	case "check", "plan", "apply":
		fs.StringVar(&profile, "profile", "", "Niri profile: generic, g14, desktop, or TOML path")
	}
	switch command {
	case "plan", "apply":
		fs.BoolVar(&live, "live", false, "appearance-only Niri update")
	}
	if command == "plan" {
		fs.BoolVar(&remove, "remove", false, "preview restoration")
	}
	if command == "apply" {
		fs.BoolVar(&installPackages, "install-packages", false, "approve Arch full system upgrade")
	}
	if command == "remove" {
		fs.BoolVar(&removePackages, "remove-packages", false, "remove eligible recorded package additions")
	}
	if command == "init" || command == "setup-step" {
		fs.StringVar(&machine, "machine", "", "g14 or desktop")
		fs.BoolVar(&skipHooks, "skip-hooks", false, "preserve existing hooks")
	}
	if command == "setup-step" {
		fs.StringVar(&action, "action", "", "setup action")
		fs.StringVar(&receipt, "receipt", "", "private handoff result")
	}
	if command == "check-secrets" {
		fs.BoolVar(&staged, "staged", false, "scan the Git index")
	}
	fs.Usage = func() {
		fmt.Fprintln(out, "Usage: bin/dotawesome [setup|sync|list|status|check|plan|apply|remove|init|enroll-age|check-secrets] [component] [flags]")
		fs.PrintDefaults()
	}
	if command == "help" || command == "--help" || command == "-h" {
		fs.Usage()
		return 0
	}
	valid := map[string]bool{"list": true, "status": true, "check": true, "plan": true, "apply": true, "remove": true, "init": true, "enroll-age": true, "check-secrets": true, "install-packages": true, "login": true, "recover": true, "setup-step": true}
	if !valid[command] {
		fs.Usage()
		return 2
	}
	flags, positions := []string{}, []string{}
	for i := 1; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if a == "--profile" || a == "-profile" || a == "--machine" || a == "-machine" || a == "--action" || a == "--receipt" {
				if i+1 < len(args) {
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			positions = append(positions, a)
		}
	}
	if e := fs.Parse(flags); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if command == "setup-step" {
		if action == "" || receipt == "" || len(positions) > 0 {
			fmt.Fprintln(stderr, "setup-step requires --action and --receipt")
			return 2
		}
		result, e := s.PerformSetupStep(ctx, action, machine, skipHooks, ask)
		if e != nil {
			result.Step = action
			result.Outcome = "failed"
			result.Detail = e.Error()
		}
		if writeErr := app.WriteSetupReceipt(receipt, result); writeErr != nil {
			return fail(writeErr)
		}
		if e != nil {
			return fail(e)
		}
		return 0
	}
	if command == "list" {
		for _, n := range components.Names {
			fmt.Fprintln(out, n)
		}
		return 0
	}
	if command == "check-secrets" {
		if e := s.ScanSecrets(ctx, staged); e != nil {
			return fail(e)
		}
		return 0
	}
	if command == "enroll-age" {
		if !ask("Enroll a dedicated age identity and encrypted recovery copy?") {
			return fail(errors.New("enrollment declined"))
		}
		if e := s.Enroll(ctx); e != nil {
			return fail(e)
		}
		return 0
	}
	if command == "init" {
		if e := s.Initialize(ctx, machine, skipHooks, ask); e != nil {
			return fail(e)
		}
		return 0
	}
	if command == "recover" {
		if !ask("Recover the interrupted transaction with its recorded originals?") {
			return fail(errors.New("recovery declined"))
		}
		if e := s.Manager.WithLock(ctx, true, func() error { return nil }); e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, "Recovery completed; inspect a fresh plan before deployment.")
		return 0
	}
	if command == "login" {
		if !ask("Run machine-local AWS login for cof-aws?") {
			return fail(errors.New("login declined"))
		}
		if e := s.Terminal(ctx, "aws", "login", "--profile", "cof-aws"); e != nil {
			return fail(e)
		}
		return 0
	}
	if command == "install-packages" {
		req := map[string][]string{}
		for _, n := range positions {
			p := ""
			if n == "niri" {
				p = "generic"
			}
			r, e := s.Inspect(ctx, n, p, false)
			if e != nil {
				return fail(e)
			}
			req[n] = r.Packages
		}
		if len(req) == 0 {
			return fail(errors.New("select components first"))
		}
		if !ask("Run sudo pacman -Syu --needed for selected components (full system upgrade)?") {
			return fail(errors.New("package upgrade declined"))
		}
		if e := s.InstallPackages(ctx, req); e != nil {
			return fail(e)
		}
		return 0
	}
	name := ""
	if len(positions) > 0 {
		name = positions[0]
	}
	if len(positions) > 1 {
		fmt.Fprintln(stderr, "too many component arguments")
		return 2
	}
	if name != "" && name != "niri" && name != "nvim" && name != "tmux" {
		fmt.Fprintln(stderr, "unknown component:", name)
		return 2
	}
	if command == "status" {
		results, e := s.Status(ctx)
		if e != nil {
			return fail(e)
		}
		for _, r := range results {
			if name == "" || r.Component == name {
				fmt.Fprintf(out, "%s: %s\n", r.Component, r.State)
				if r.Detail != "" {
					fmt.Fprintln(out, r.Detail)
				}
				if r.Backup != "" {
					fmt.Fprintln(out, "  backup:", r.Backup)
				}
			}
		}
		return 0
	}
	if name == "" {
		fs.Usage()
		return 2
	}
	if live && name != "niri" {
		return fail(errors.New("--live is only valid for niri"))
	}
	if live && installPackages {
		return fail(errors.New("live update cannot install packages; install outside Niri"))
	}
	if remove && (profile != "" || live) {
		return fail(errors.New("plan --remove does not accept --profile or --live"))
	}
	if (command == "apply" || command == "remove") && !live && deploy.Exists(filepath.Join(s.Manager.State, "transaction.json")) {
		if !ask("Recover the interrupted transaction before fresh inspection?") {
			return fail(errors.New("recovery declined"))
		}
		if e := s.Manager.WithLock(ctx, true, func() error { return nil }); e != nil {
			return fail(e)
		}
	}
	restore := command == "remove" || remove
	r, e := s.Inspect(ctx, name, profile, restore)
	if e != nil {
		return fail(e)
	}
	fmt.Fprintf(out, "%s: %s\n", name, command)
	for _, e := range r.Errors {
		fmt.Fprintln(out, "  error:", e)
	}
	for _, w := range r.Warnings {
		fmt.Fprintln(out, "  warning:", w)
	}
	if command == "check" {
		if len(r.Errors) > 0 {
			return fail(errors.New("preflight failed"))
		}
		if e = s.Validate(ctx, r); e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, "  configuration: valid")
		return 0
	}
	keys := []string{}
	for p := range r.Plan.Changes {
		keys = append(keys, p)
	}
	sort.Strings(keys)
	for _, p := range keys {
		change := r.Plan.Changes[p]
		d, _, _ := app.Unified(p, change.Before, change.After)
		fmt.Fprint(out, d)
	}
	fmt.Fprintf(out, "%d file(s) changed.\n", len(keys))
	if restore {
		candidates, e := s.PackagePlan(ctx, name)
		if e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, "Package cleanup:", strings.Join(candidates, ", "))
	} else {
		fmt.Fprintln(out, "Packages required:", strings.Join(r.Packages, ", "))
	}
	if command == "plan" {
		if live {
			if e = s.ValidateLive(r); e != nil {
				return fail(e)
			}
		}
		fmt.Fprintln(out, "No changes made. Native validation runs during check/apply.")
		return 0
	}
	if !ask("Apply these reviewed changes with backups?") {
		return fail(errors.New("operation declined; no configuration deployed"))
	}
	if installPackages {
		if !ask("Run a full system upgrade with sudo pacman -Syu --needed?") {
			return fail(errors.New("package upgrade declined"))
		}
		if e = s.InstallPackages(ctx, map[string][]string{name: r.Packages}); e != nil {
			return fail(e)
		}
		r, e = s.Inspect(ctx, name, profile, false)
		if e != nil {
			return fail(e)
		}
		if !ask("Confirm the freshly inspected configuration after package update?") {
			return fail(errors.New("deployment declined"))
		}
	}
	if e = s.Execute(ctx, r, live); e != nil {
		return fail(e)
	}
	required := r.Packages
	if restore {
		required = nil
	}
	if e = s.RecordRequirements(ctx, name, required); e != nil {
		return fail(e)
	}
	if removePackages {
		if !ask("Remove eligible recorded package additions with sudo pacman -Ru?") {
			return fail(errors.New("package cleanup declined; configuration already restored"))
		}
		removed, e := s.RemovePackages(ctx, name)
		if e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, "Removed packages:", strings.Join(removed, ", "))
	}
	fmt.Fprintf(out, "%s: completed. Backup record: %s\n", name, filepath.Join(s.Manager.State, "installed.json"))
	return 0
}

// Root locates the checkout for both launcher builds and direct go run invocations.
func Root() string {
	if r := os.Getenv("DOTAWESOME_ROOT"); r != "" {
		r, _ = filepath.Abs(r)
		return r
	}
	cwd, _ := os.Getwd()
	for p := cwd; ; p = filepath.Dir(p) {
		if _, e := os.Stat(filepath.Join(p, ".chezmoiroot")); e == nil {
			return p
		}
		if p == filepath.Dir(p) {
			return cwd
		}
	}
}
