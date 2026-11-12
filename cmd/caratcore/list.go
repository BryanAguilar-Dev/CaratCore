package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/BryanAguilar-Dev/CaratCore/internal/catalog"
	"github.com/BryanAguilar-Dev/CaratCore/internal/service"
)

// listRow is one service in the output of `caratcore list`.
type listRow struct {
	id      string
	status  service.Status
	backups []string
}

// runList handles `caratcore list`. It only reads files.
func runList(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "usage: caratcore list")
		return 2
	}
	if runtime.GOOS != "linux" {
		fmt.Fprintln(os.Stderr, "unsupported platform: services run on Linux only")
		return 1
	}
	services, err := catalog.All()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rows := make([]listRow, 0, len(services))
	for _, s := range services {
		host, err := service.Detect(ctx, s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		st, err := service.StatusOf(s, host)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		rows = append(rows, listRow{s.ID, st, host.Backups})
	}
	printList(os.Stdout, rows)
	return 0
}

func printList(w io.Writer, rows []listRow) {
	fmt.Fprintf(w, "Services in %s, read from their files (CaratCore %s).\n", service.BaseDir, version)
	fmt.Fprintln(w, "It does not ask Docker whether they are running: use `sudo docker ps` for that.")
	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SERVICE\tSTATUS\tDETAILS")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", r.id, r.status.Name, r.status.Detail)
	}
	tw.Flush()

	var copies []string
	for _, r := range rows {
		copies = append(copies, r.backups...)
	}
	if len(copies) > 0 {
		fmt.Fprintln(w, "\nCopies made by `caratcore update` (delete them with sudo rm -r when you no longer need them):")
		for _, c := range copies {
			fmt.Fprintln(w, "  "+c)
		}
	}
}
