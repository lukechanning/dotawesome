package main

import (
	"context"
	"dotawesome/internal/external"
	"dotawesome/internal/work"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()
	var e error
	if len(os.Args) > 1 {
		if os.Args[1] != "focus" {
			fmt.Fprintln(os.Stderr, "usage: work [focus APP_ID...]")
			os.Exit(2)
		}
		e = work.Focus(ctx, external.Run, os.Args[2:])
		if e != nil {
			external.Run(ctx, "noctalia", "msg", "notification-show", "Jump to app", e.Error())
		}
	} else {
		e = work.Main(ctx, external.Run)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, "work layout:", e)
		os.Exit(1)
	}
}
