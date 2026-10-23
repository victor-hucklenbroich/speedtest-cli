// Command gendocs regenerates the usage section of README.md from cli.Usage()
package main

import (
	"bytes"
	"fmt"
	"os"

	"speedtest/internal/cli"
	"speedtest/internal/constants"
)

func main() {
	data, err := os.ReadFile(constants.ReadmePath)
	if err != nil {
		fail(err)
	}
	begin := bytes.Index(data, []byte(constants.UsageBeginMarker))
	end := bytes.Index(data, []byte(constants.UsageEndMarker))
	if begin < 0 || end < 0 || end < begin {
		fail(fmt.Errorf("%s must contain %q followed by %q", constants.ReadmePath, constants.UsageBeginMarker, constants.UsageEndMarker))
	}

	var out bytes.Buffer
	out.Write(data[:begin+len(constants.UsageBeginMarker)])
	fmt.Fprintf(&out, "\n\n```\n%s```\n\n", cli.Usage())
	out.Write(data[end:])

	if bytes.Equal(out.Bytes(), data) {
		return
	}
	if err := os.WriteFile(constants.ReadmePath, out.Bytes(), 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "gendocs:", err)
	os.Exit(1)
}
