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
		BaseURL:        cfg.ServerURL,
		Client:         &http.Client{Timeout: 60 * time.Second},
		LatencySamples: 20,
		DownloadSizes: []int{
			1 << 20,
			10 << 20,
			25 << 20,
			50 << 20,
			90 << 20,
		},
		UploadSizes: []int{
			1 << 20,
			5 << 20,
			10 << 20,
			25 << 20,
			50 << 20,
		},
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

	if _, err := tea.NewProgram(tui.New(events, cancel, cfg.ServerURL, cli.Version)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
		os.Exit(1)
	}
}
