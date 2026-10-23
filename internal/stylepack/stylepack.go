// Package stylepack loads user styles
package stylepack

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	"speedtest/style"
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
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0o755)
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
