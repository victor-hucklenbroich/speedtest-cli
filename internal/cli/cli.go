package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultServerURL = "https://speedtest-worker.speedtest-cli.workers.dev"

	DefaultTimeoutMinutes = 1

	MaxSize = 1 << 40
)

var Version = "0.1.1"

type Config struct {
	ServerURL   string
	ShowVersion bool
	Plain       bool
	Ping        bool
	Down        bool
	Up          bool
	SizeBytes   int
	SizeAndUp   bool
	Timeout     time.Duration
	sizeRaw     string
	timeoutMins float64
}

func flagSet(cfg *Config) *flag.FlagSet {
	fs := flag.NewFlagSet("speedtest", flag.ExitOnError)
	fs.StringVar(&cfg.ServerURL, "url", "", "speedtest server base URL (saved to the config file)")
	fs.BoolVar(&cfg.Ping, "ping", false, "measure ping")
	fs.BoolVar(&cfg.Down, "down", false, "measure download")
	fs.BoolVar(&cfg.Up, "up", false, "measure upload")
	fs.StringVar(&cfg.sizeRaw, "size", "", "transfer size, e.g. 25MB or 500KB (bare number = MB, max 1TB); append + to escalate from there")
	fs.Float64Var(&cfg.timeoutMins, "timeout", 0, "per-transfer timeout in minutes, default 1 (saved to the config file)")
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
	fmt.Fprintf(&b, "\n--timeout 5 caps each transfer at 5 minutes; raise it for very large --size values.\n")
	fmt.Fprintf(&b, "\n--url and --timeout are written to the config file, so they apply now and persist for\n")
	fmt.Fprintf(&b, "later runs. Without a flag the saved value is used (built-in defaults on first run).\n")
	fmt.Fprintf(&b, "\nConfig file:\n")
	fmt.Fprintf(&b, "  macOS    ~/Library/Application Support/speedtest/config.yml\n")
	fmt.Fprintf(&b, "  Linux    ~/.config/speedtest/config.yml\n")
	fmt.Fprintf(&b, "  Windows  %%AppData%%\\speedtest\\config.yml\n")
	return b.String()
}

func ParseConfig(args []string) (Config, error) {
	var cfg Config
	fs := flagSet(&cfg)
	fs.Parse(args)

	if fs.NArg() > 0 {
		return Config{}, fmt.Errorf("unexpected argument %q (for supported flags see 'speedtest --help')", fs.Arg(0))
	}

	if cfg.ShowVersion {
		return cfg, nil
	}

	if cfg.timeoutMins < 0 {
		return Config{}, fmt.Errorf("invalid --timeout %g (minutes must be positive)", cfg.timeoutMins)
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

	if err := resolveConfig(&cfg); err != nil {
		return Config{}, err
	}
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
		{"TB", 1 << 40},
		{"T", 1 << 40},
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
		return 0, false, fmt.Errorf("invalid --size %q (examples: 25MB, 500KB, 1.5GB, 25MB+)", s)
	}
	if v*mult > MaxSize {
		return 0, false, fmt.Errorf("--size %q exceeds the 1 TB limit", s)
	}
	b := int(v * mult)
	if b < 1 {
		return 0, false, fmt.Errorf("invalid --size %q (too small)", s)
	}
	return b, andUp, nil
}
