package cli

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	DefaultServerURL = "https://speedtest-worker.speedtest-cli.workers.dev"

	ServerURLEnv = "SPEEDTEST_URL"
)

var Version = "0.1.0-indev"

type Config struct {
	ServerURL   string
	ShowVersion bool
	Plain       bool
}

func flagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("speedtest", flag.ExitOnError)
	fs.StringVar(&cfg.ServerURL, "url", "", "speedtest server base URL")
	fs.BoolVar(&cfg.Plain, "plain", false, "plain line output instead of the animated TUI (automatic when stdout is not a terminal)")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "print version and exit")
	fs.Usage = func() { fmt.Fprint(fs.Output(), Usage()) }
	return fs
}

func Usage() string {
	var b strings.Builder
	b.WriteString("Usage: speedtest [flags]\n\nFlags:\n")
	fs := flagSet(&Config{})
	fs.SetOutput(&b)
	fs.PrintDefaults()
	fmt.Fprintf(&b, "\nThe server URL is resolved in this order:\n")
	fmt.Fprintf(&b, "  1. --url flag\n")
	fmt.Fprintf(&b, "  2. %s environment variable\n", ServerURLEnv)
	fmt.Fprintf(&b, "  3. built-in default: %s\n", DefaultServerURL)
	return b.String()
}

func ParseConfig(args []string) (Config, error) {
	var cfg Config
	flagSet(&cfg).Parse(args)

	if cfg.ShowVersion {
		return cfg, nil
	}

	raw := cfg.ServerURL
	if raw == "" {
		raw = os.Getenv(ServerURLEnv)
	}
	if raw == "" {
		raw = DefaultServerURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return Config{}, fmt.Errorf("invalid server URL %q (expected http(s)://host)", raw)
	}
	cfg.ServerURL = strings.TrimRight(raw, "/")
	return cfg, nil
}
