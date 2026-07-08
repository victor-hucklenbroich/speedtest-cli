package cli

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	URL     string  `yaml:"url"`
	Timeout float64 `yaml:"timeout"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "speedtest", "config.yml"), nil
}

func readFileConfig(path string) (fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, nil
	}
	var fc fileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fileConfig{}, fmt.Errorf("config %s: %w", path, err)
	}
	return fc, nil
}

func writeFileConfig(path string, fc fileConfig) {
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	content := fmt.Sprintf(`# speedtest configuration
# --url and --timeout write their values here, so they persist across runs.
# You can also edit these directly.
url: %s
timeout: %g
`, fc.URL, fc.Timeout)
	os.WriteFile(path, []byte(content), 0o644)
}

func resolveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		path = ""
	}
	var file fileConfig
	if path != "" {
		if file, err = readFileConfig(path); err != nil {
			return err
		}
	}

	changed := false
	if cfg.ServerURL != "" && cfg.ServerURL != file.URL {
		file.URL, changed = cfg.ServerURL, true
	}
	if cfg.timeoutMins > 0 && cfg.timeoutMins != file.Timeout {
		file.Timeout, changed = cfg.timeoutMins, true
	}
	if file.URL == "" {
		file.URL, changed = DefaultServerURL, true
	}
	if file.Timeout <= 0 {
		file.Timeout, changed = DefaultTimeoutMinutes, true
	}

	u, err := url.Parse(file.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("invalid server URL %q (expected http(s)://host)", file.URL)
	}
	if trimmed := strings.TrimRight(file.URL, "/"); trimmed != file.URL {
		file.URL, changed = trimmed, true
	}

	if changed && path != "" {
		writeFileConfig(path, file)
	}

	cfg.ServerURL = file.URL
	cfg.Timeout = time.Duration(file.Timeout * float64(time.Minute))
	return nil
}
