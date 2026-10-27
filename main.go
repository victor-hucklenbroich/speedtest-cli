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
	"speedtest/internal/stylepack"
	"speedtest/internal/tui"
	"speedtest/style"
	"speedtest/style/builtin"

	tea "github.com/charmbracelet/bubbletea"
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
	if cfg.SetStyle != "" {
		if err := setStyle(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
			os.Exit(1)
		}
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

	tuiStyle := loadStyle(cfg)

	events := make(chan measure.Event, constants.EventBuffer)
	go func() {
		measure.Run(ctx, mcfg, events)
		close(events)
	}()

	server := ""
	if cfg.ServerURL != constants.DefaultServerURL {
		server = cfg.ServerURL
	}
	if _, err := tea.NewProgram(tui.New(events, cancel, server, cfg.Ping, tuiStyle)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
		os.Exit(1)
	}
}

func setStyle(cfg cli.Config) error {
	loadStyles(cfg.StylesDir)
	if _, ok := style.Lookup(cfg.SetStyle); !ok {
		return fmt.Errorf("unknown style %q", cfg.SetStyle)
	}
	if err := cli.SaveStyle(cfg.SetStyle); err != nil {
		return err
	}
	fmt.Printf("✓ speedtest style set to \"%s\"\n", cfg.SetStyle)
	return nil
}

func loadStyle(cfg cli.Config) style.Style {
	loadStyles(cfg.StylesDir)
	if s, ok := style.Lookup(cfg.Style); ok {
		return s
	}
	fmt.Fprintf(os.Stderr, "speedtest: unknown style %q, using %q\n", cfg.Style, builtin.Digits.Name)
	return builtin.Digits
}

func loadStyles(dir string) {
	if dir == "" {
		return
	}
	for _, err := range stylepack.Load(dir) {
		fmt.Fprintf(os.Stderr, "speedtest: %v\n", err)
	}
}
