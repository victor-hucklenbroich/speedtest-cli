// Package builtin holds the styles compiled into speedtest, one style per file
package builtin

import (
	"embed"
	"strings"
)

//go:embed *.go
var files embed.FS

// Sources returns the source of every built-in style by name
func Sources() map[string][]byte {
	entries, _ := files.ReadDir(".")
	srcs := make(map[string][]byte, len(entries))
	for _, e := range entries {
		if e.Name() == "doc.go" || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		srcs[e.Name()], _ = files.ReadFile(e.Name())
	}
	return srcs
}
