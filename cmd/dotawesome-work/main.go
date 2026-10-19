package main

import (
	"context"
	"dotawesome/internal/external"
	"dotawesome/internal/work"
	"fmt"
	"os"
)

func main() {
	if e := work.Main(context.Background(), external.Run); e != nil {
		fmt.Fprintln(os.Stderr, "work layout:", e)
		os.Exit(1)
	}
}
