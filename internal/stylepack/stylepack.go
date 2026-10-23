// Package stylepack loads user styles
package stylepack

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	"speedtest/internal/constants"
	"speedtest/style"
	"speedtest/style/builtin"
)

// symbols exposes package style to interpreted files as "speedtest/style".
var symbols = interp.Exports{
	"speedtest/style/style": {
		"Frame":    reflect.ValueOf((*style.Frame)(nil)),
		"Style":    reflect.ValueOf((*style.Style)(nil)),
		"Register": reflect.ValueOf(style.Register),
		"Lookup":   reflect.ValueOf(style.Lookup),
	},
}

func Load(dir string) []error {
	deposit(dir, builtin.Sources())
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return []error{err}
	}
	var errs []error
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if src, err := os.ReadFile(path); err == nil && pristine(src) {
			// An unedited built-in copy: the compiled style is used instead.
			continue
		}
		if err := loadFile(path); err != nil {
			msg := strings.ReplaceAll(err.Error(), path, e.Name())
			if !strings.HasPrefix(msg, e.Name()) {
				msg = e.Name() + ": " + msg
			}
			errs = append(errs, errors.New("style "+msg))
		}
	}
	return errs
}

// deposit copies the built-in styles into dir as editable templates. Copies
// still matching their checksum are refreshed when the built-in changes; edited
// ones are left alone, and deleted ones are restored.
func deposit(dir string, sources map[string][]byte) {
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	for name, src := range sources {
		path := filepath.Join(dir, name)
		want := withChecksum(src)
		if have, err := os.ReadFile(path); err == nil && (bytes.Equal(have, want) || !pristine(have)) {
			continue
		}
		os.WriteFile(path, want, 0o644)
	}
}

func withChecksum(src []byte) []byte {
	return fmt.Appendf(nil, `// A built-in speedtest style. Edit it to change that style, or copy it under a
// new Name to add your own. speedtest keeps this file up to date until it is
// edited; delete it to get the current version back.
%s%x

%s`, constants.StyleChecksumPrefix, sha256.Sum256(src), src)
}

// pristine reports whether data is a built-in copy nobody has edited.
func pristine(data []byte) bool {
	_, rest, ok := bytes.Cut(data, []byte(constants.StyleChecksumPrefix))
	if !ok {
		return false
	}
	sum, src, ok := bytes.Cut(rest, []byte("\n\n"))
	return ok && string(sum) == fmt.Sprintf("%x", sha256.Sum256(src))
}

func loadFile(path string) error {
	i := interp.New(interp.Options{})
	if err := i.Use(stdlib.Symbols); err != nil {
		return err
	}
	if err := i.Use(symbols); err != nil {
		return err
	}
	_, err := i.EvalPath(path)
	return err
}
