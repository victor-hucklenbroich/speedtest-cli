package builtin

import (
	"fmt"
	"strings"

	"speedtest/style"
)

// Digits is the default style: a block-digit readout over a sparkline.
var Digits = style.Style{Name: "digits", Live: live, Bar: bar}

func init() { style.Register(Digits) }

const sparkMaxWidth = 48

func live(f style.Frame) []string {
	var lines []string
	for i, row := range bigNumber(f.Mbps) {
		line := f.Accent(row)
		if i == 2 {
			line += f.Dim("  Mbps")
		}
		lines = append(lines, line)
	}
	w := f.Width
	if w > sparkMaxWidth {
		w = sparkMaxWidth
	}
	if len(f.History) > 0 && w > 0 {
		lines = append(lines, f.Accent(sparkline(f.History, w)))
	}
	return lines
}

var digitFont = map[rune][]string{
	'0': {"███", "█ █", "█ █", "█ █", "███"},
	'1': {"  █", "  █", "  █", "  █", "  █"},
	'2': {"███", "  █", "███", "█  ", "███"},
	'3': {"███", "  █", "███", "  █", "███"},
	'4': {"█ █", "█ █", "███", "  █", "  █"},
	'5': {"███", "█  ", "███", "  █", "███"},
	'6': {"███", "█  ", "███", "█ █", "███"},
	'7': {"███", "  █", "  █", "  █", "  █"},
	'8': {"███", "█ █", "███", "█ █", "███"},
	'9': {"███", "█ █", "███", "  █", "███"},
	'.': {" ", " ", " ", " ", "█"},
	' ': {"   ", "   ", "   ", "   ", "   "},
}

func bigNumber(v float64) []string {
	s := fmt.Sprintf("%5.1f", v)
	if v >= 1000 {
		s = fmt.Sprintf("%5.0f", v)
	}
	rows := make([]string, 5)
	for i := range rows {
		var parts []string
		for _, r := range s {
			if g, ok := digitFont[r]; ok {
				parts = append(parts, g[i])
			}
		}
		rows[i] = strings.Join(parts, " ")
	}
	return rows
}

var sparkChars = []rune("▁▂▃▄▅▆▇█")

func sparkline(vals []float64, width int) string {
	if len(vals) > width {
		vals = vals[len(vals)-width:]
	}
	max := 0.0
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	var b strings.Builder
	for _, v := range vals {
		idx := 0
		if max > 0 {
			idx = int(v / max * float64(len(sparkChars)-1))
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		b.WriteRune(sparkChars[idx])
	}
	return b.String()
}

func bar(f style.Frame) string {
	frac := f.Progress
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	fill := int(frac*float64(f.Width) + 0.5)
	return f.Accent(strings.Repeat("█", fill) + strings.Repeat("░", f.Width-fill))
}
