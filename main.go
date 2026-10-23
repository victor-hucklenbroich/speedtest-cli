package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"speedtest/internal/cli"
	"speedtest/internal/constants"
	"speedtest/internal/measure"
	"speedtest/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

//go:generate go run ./internal/gendocs

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
		fmt.Printf("speedtest %s (%s/%s)\n", constants.Version, runtime.GOOS, runtime.GOARCH)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mcfg := measure.Config{
		BaseURL: cfg.ServerURL,
		Client:  &http.Client{Timeout: cfg.Timeout},
	}
	if cfg.Ping {
		mcfg.LatencySamples = constants.LatencySamples
	}
	if cfg.Down {
		mcfg.DownloadSizes = ladder(constants.DownloadLadder, cfg.SizeBytes, cfg.SizeAndUp)
	}
	if cfg.Up {
		mcfg.UploadSizes = ladder(constants.UploadLadder, cfg.SizeBytes, cfg.SizeAndUp)
	}

	events := make(chan measure.Event, constants.EventBuffer)
	go func() {
		measure.Run(ctx, mcfg, events)
		close(events)
	}()

	if cfg.Plain || !(isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())) {
		runPlain(events, cfg.ServerURL)
		return
	}

	server := ""
	if cfg.ServerURL != constants.DefaultServerURL {
		server = cfg.ServerURL
	}
	if _, err := tea.NewProgram(tui.New(events, cancel, server, cfg.Ping)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
		os.Exit(1)
	}
}
