package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"speedtest/internal/cli"
	"speedtest/internal/measure"
	"speedtest/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

//go:generate go run ./internal/gendocs

var (
	downloadLadder = []int{
		10 << 20,
		25 << 20,
		50 << 20,
		90 << 20,
	}
	uploadLadder = []int{
		5 << 20,
		10 << 20,
		25 << 20,
		50 << 20,
	}
)

func ladder(defaults []int, size int, andUp bool) []int {
	if size == 0 {
		return defaults
	}
	if !andUp {
		return []int{size}
	}
	out := []int{size}
	for _, s := range defaults {
		if s > size {
			out = append(out, s)
		}
	}
	return out
}

func main() {
	cfg, err := cli.ParseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
		os.Exit(2)
	}
	if cfg.ShowVersion {
		fmt.Printf("speedtest %s (%s/%s)\n", cli.Version, runtime.GOOS, runtime.GOARCH)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mcfg := measure.Config{
		BaseURL: cfg.ServerURL,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
	if cfg.Ping {
		mcfg.LatencySamples = 20
	}
	if cfg.Down {
		mcfg.DownloadSizes = ladder(downloadLadder, cfg.SizeBytes, cfg.SizeAndUp)
	}
	if cfg.Up {
		mcfg.UploadSizes = ladder(uploadLadder, cfg.SizeBytes, cfg.SizeAndUp)
	}

	events := make(chan measure.Event, 64)
	go func() {
		measure.Run(ctx, mcfg, events)
		close(events)
	}()

	if cfg.Plain || !(isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())) {
		runPlain(events, cfg.ServerURL)
		return
	}

	server := ""
	if cfg.ServerURL != cli.DefaultServerURL {
		server = cfg.ServerURL
	}
	if _, err := tea.NewProgram(tui.New(events, cancel, server, cfg.Ping)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
		os.Exit(1)
	}
}
