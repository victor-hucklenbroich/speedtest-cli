package cli

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultServerURL = "https://speedtest-worker.speedtest-cli.workers.dev"

	ServerURLEnv = "SPEEDTEST_URL"

	MaxSize = 1 << 30
)

var Version = "0.1.0-indev"

type Config struct {
	ServerURL   string
	ShowVersion bool
	Plain       bool
	Ping        bool
	Down        bool
	Up          bool
	SizeBytes   int
	SizeAndUp   bool
	sizeRaw     string
}

func flagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("speedtest", flag.ExitOnError)
	fs.StringVar(&cfg.ServerURL, "url", "", "speedtest server base URL")
	fs.BoolVar(&cfg.Ping, "ping", false, "measure ping")
	fs.BoolVar(&cfg.Down, "down", false, "measure download")
	fs.BoolVar(&cfg.Up, "up", false, "measure upload")
	fs.StringVar(&cfg.sizeRaw, "size", "", "transfer size, e.g. 25MB or 500KB (bare number = MB, max 1GB); append + to escalate from there")
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
	fmt.Fprintf(&b, "\nPhase flags combine: --ping --down runs ping and download\n")
	fmt.Fprintf(&b, "\n--size 25MB runs a single 25 MB transfer\n")
	fmt.Fprintf(&b, "--size 25MB+ walks the normal escalation ladder but starts it at 25 MB.\n")
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

	if !cfg.Ping && !cfg.Down && !cfg.Up {
		cfg.Ping, cfg.Down, cfg.Up = true, true, true
	}

	if cfg.sizeRaw != "" {
		if !cfg.Down && !cfg.Up {
			return Config{}, fmt.Errorf("--size has no effect with --ping only")
		}
		var err error
		cfg.SizeBytes, cfg.SizeAndUp, err = parseSize(cfg.sizeRaw)
		if err != nil {
			return Config{}, err
		}
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

func parseSize(s string) (int, bool, error) {
	andUp := strings.HasSuffix(s, "+")
	t := strings.ToUpper(strings.TrimSpace(strings.TrimSuffix(s, "+")))
	mult := float64(1 << 20)
	for _, u := range []struct {
		suffix string
		mult   float64
	}{
		{"GB", 1 << 30},
		{"G", 1 << 30},
		{"MB", 1 << 20},
		{"M", 1 << 20},
		{"KB", 1 << 10},
		{"K", 1 << 10},
		{"B", 1},
	} {
		if strings.HasSuffix(t, u.suffix) {
			t = strings.TrimSuffix(t, u.suffix)
			mult = u.mult
			break
		}
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil || v <= 0 {
		return 0, false, fmt.Errorf("invalid --size %q (examples: 25MB, 500KB, 1.5MB, 25MB+)", s)
	}
	b := int(v * mult)
	if b < 1 {
		return 0, false, fmt.Errorf("invalid --size %q (too small)", s)
	}
	if b > MaxSize {
		return 0, false, fmt.Errorf("--size %q exceeds the 1 GB limit", s)
	}
	return b, andUp, nil
}
