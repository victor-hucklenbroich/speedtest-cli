// Command gendocs regenerates the usage section of README.md from
// cli.Usage(), keeping --help and the README in sync. Run it from the
// repository root, normally via go generate ./...
package main

import (
	"bytes"
	"fmt"
	"os"

	"speedtest/internal/cli"
)

const (
	readmePath  = "README.md"
	beginMarker = "<!-- BEGIN USAGE -->"
	endMarker   = "<!-- END USAGE -->"
)

func main() {
	data, err := os.ReadFile(readmePath)
	if err != nil {
		fail(err)
	}
	begin := bytes.Index(data, []byte(beginMarker))
	end := bytes.Index(data, []byte(endMarker))
	if begin < 0 || end < 0 || end < begin {
		fail(fmt.Errorf("%s must contain %q followed by %q", readmePath, beginMarker, endMarker))
	}

	var out bytes.Buffer
	out.Write(data[:begin+len(beginMarker)])
	fmt.Fprintf(&out, "\n\n```\n%s```\n\n", cli.Usage())
	out.Write(data[end:])

	if bytes.Equal(out.Bytes(), data) {
		return
	}
	if err := os.WriteFile(readmePath, out.Bytes(), 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "gendocs:", err)
	os.Exit(1)
}
