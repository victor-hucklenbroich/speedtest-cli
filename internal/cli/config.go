package cli

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"speedtest/internal/constants"
)

type fileConfig struct {
	URL     string  `yaml:"url"`
	Timeout float64 `yaml:"timeout"`
	Style   string  `yaml:"style"`
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

func writeFileConfig(path string, fc fileConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf(`# speedtest configuration
# --url, --timeout and --style write their values here, so they persist across runs.
# You can also edit these directly.
url: %s
timeout: %g
style: %s
`, fc.URL, fc.Timeout, fc.Style)
	return os.WriteFile(path, []byte(content), 0o644)
}

// SaveStyle makes name the style of later runs.
func SaveStyle(name string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	file, err := readFileConfig(path)
	if err != nil {
		return err
	}
	file.Style = name
	return writeFileConfig(path, file)
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
		file.URL, changed = constants.DefaultServerURL, true
	}
	if file.Timeout <= 0 {
		file.Timeout, changed = constants.DefaultTimeoutMinutes, true
	}
	if file.Style == "" {
		file.Style, changed = constants.DefaultStyle, true
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
	cfg.Style = file.Style
	if path != "" {
		cfg.StylesDir = filepath.Join(filepath.Dir(path), "styles")
	}
	return nil
}
