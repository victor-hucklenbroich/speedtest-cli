package builtin

import (
	"fmt"
	"math"
	"strings"

	"speedtest/style"
)

var Gauges = style.Style{Name: "gauges", Live: gaugeLive, Bar: gaugeBar}

func init() { style.Register(Gauges) }

const (
	gaugeMaxWidth = 36
	gaugeMinWidth = 28

	gaugeStart = 225.0
	gaugeSweep = 270.0
)

var gaugeTicks = []float64{0, 1, 5, 10, 20, 30, 50, 75, 100}

func gaugeLive(f style.Frame) []string {
	cols := f.Width
	if cols > gaugeMaxWidth {
		cols = gaugeMaxWidth
	}
	if cols < gaugeMinWidth {
		num, unit := gaugeReading(f.Mbps)
		return []string{f.Accent(num) + f.Dim(" "+unit)}
	}

	ticks := gaugeScale(f)
	val := gaugeFrac(ticks, f.Mbps)

	r := float64(cols - 1)
	thick := math.Max(3, math.Round(r/7))
	mid := r - thick/2
	labelR := r - thick - 3
	needle := labelR - 8
	cx, cy := float64(cols), r+1

	type label struct {
		text     string
		col, row int
	}
	labels := make([]label, len(ticks))
	for i, t := range ticks {
		s := gaugeLabel(t)
		a := gaugeAngle(float64(i) / float64(len(ticks)-1))
		hw, hh := float64(len(s)), 2.0
		d := labelR - hw*math.Abs(math.Cos(a)) - hh*math.Abs(math.Sin(a))
		labels[i] = label{s, int(math.Round((cx + d*math.Cos(a) - hw) / 2)), int(math.Round((cy - d*math.Sin(a) - hh) / 4))}
	}

	top := int((cy+needle*math.Sqrt2/2)/4) + 1
	left, right := (cols-11)/2-1, (cols+11)/2+1
	for moved := true; moved; {
		moved = false
		for _, l := range labels {
			if l.row == top && l.col < right && l.col+len(l.text) > left {
				top, moved = l.row+1, true
			}
		}
	}
	rows := top + 1
	if bottom := int(math.Ceil((cy + mid*math.Sqrt2/2 + thick/2) / 4)); bottom > rows {
		rows = bottom
	}
	c := newGaugeCanvas(cols, rows)

	tip := gaugeAngle(val)
	ux, uy := math.Cos(tip), math.Sin(tip)
	for y := 0; y < c.rows*4; y++ {
		for x := 0; x < c.cols*2; x++ {
			dx, dy := float64(x)+0.5-cx, cy-float64(y)-0.5
			d := math.Hypot(dx, dy)
			along, across := dx*ux+dy*uy, math.Abs(dx*uy-dy*ux)
			switch {
			case d <= 2.5:
				c.dot(x, y, gaugePlain)
			case along >= 0 && along <= needle && across <= 1.5*(1-along/needle)+0.5:
				c.dot(x, y, gaugePlain)
			default:
				pos := math.Mod(gaugeStart-math.Atan2(dy, dx)*180/math.Pi+360, 360)
				if pos > gaugeSweep {
					if pos > (gaugeSweep+360)/2 {
						pos = 0
					} else {
						pos = gaugeSweep
					}
					a := gaugeAngle(pos / gaugeSweep)
					d = math.Hypot(dx-mid*math.Cos(a), dy-mid*math.Sin(a))
				} else {
					d -= mid
				}
				if math.Abs(d) > thick/2 {
					continue
				}
				if pos < val*gaugeSweep || val > 0 && pos == 0 {
					c.dot(x, y, gaugeAccent)
				} else {
					c.dot(x, y, gaugeDim)
				}
			}
		}
	}

	for i, l := range labels {
		ink := gaugeDim
		if ticks[i] <= f.Mbps {
			ink = gaugePlain
		}
		c.text(l.col, l.row, l.text, ink)
	}

	num, unit := gaugeReading(f.Mbps)
	arrow := "↓"
	if f.Phase == "upload" {
		arrow = "↑"
	}
	col := (cols - len(num) - len(unit) - 3) / 2
	c.text(col, top, arrow, gaugeAccent)
	c.text(col+2, top, num, gaugePlain)
	c.text(col+3+len(num), top, unit, gaugeDim)
	return c.lines(f)
}

func gaugeScale(f style.Frame) []float64 {
	peak := f.Mbps
	for _, v := range f.History {
		if v > peak {
			peak = v
		}
	}
	mult := 1.0
	for i := 0; i < 6 && gaugeTicks[len(gaugeTicks)-1]*mult < peak; i++ {
		mult *= 10
	}
	ticks := make([]float64, len(gaugeTicks))
	for i, t := range gaugeTicks {
		ticks[i] = t * mult
	}
	return ticks
}

func gaugeFrac(ticks []float64, v float64) float64 {
	if !(v > ticks[0]) {
		return 0
	}
	n := len(ticks) - 1
	for i := 0; i < n; i++ {
		if v <= ticks[i+1] {
			return (float64(i) + (v-ticks[i])/(ticks[i+1]-ticks[i])) / float64(n)
		}
	}
	return 1
}

func gaugeAngle(frac float64) float64 {
	return (gaugeStart - frac*gaugeSweep) * math.Pi / 180
}

func gaugeLabel(t float64) string {
	if t >= 1000 {
		return fmt.Sprintf("%gk", t/1000)
	}
	return fmt.Sprintf("%g", t)
}

func gaugeReading(v float64) (num, unit string) {
	unit = "Mbps"
	if v >= 999.5 {
		v, unit = v/1000, "Gbps"
	}
	switch {
	case v < 9.995:
		return fmt.Sprintf("%.2f", v), unit
	case v < 99.95:
		return fmt.Sprintf("%.1f", v), unit
	}
	return fmt.Sprintf("%.0f", v), unit
}

const (
	gaugeBlank = iota
	gaugeDim
	gaugeAccent
	gaugePlain
)

type gaugeCell struct {
	dots byte
	text rune
	ink  int
}

type gaugeCanvas struct {
	cols, rows int
	cells      [][]gaugeCell
}

func newGaugeCanvas(cols, rows int) *gaugeCanvas {
	cells := make([][]gaugeCell, rows)
	for i := range cells {
		cells[i] = make([]gaugeCell, cols)
	}
	return &gaugeCanvas{cols: cols, rows: rows, cells: cells}
}

var gaugeDots = [4][2]byte{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

func (c *gaugeCanvas) dot(x, y, ink int) {
	if x < 0 || y < 0 || x >= c.cols*2 || y >= c.rows*4 {
		return
	}
	cell := &c.cells[y/4][x/2]
	cell.dots |= gaugeDots[y%4][x%2]
	if ink > cell.ink {
		cell.ink = ink
	}
}

func (c *gaugeCanvas) text(col, row int, s string, ink int) {
	if row < 0 || row >= c.rows {
		return
	}
	for _, r := range s {
		if col >= 0 && col < c.cols {
			c.cells[row][col] = gaugeCell{text: r, ink: ink}
		}
		col++
	}
}

func (c *gaugeCanvas) lines(f style.Frame) []string {
	paint := map[int]func(string) string{gaugeDim: f.Dim, gaugeAccent: f.Accent}
	out := make([]string, 0, c.rows)
	for _, row := range c.cells {
		var b strings.Builder
		for i := 0; i < len(row); {
			var run strings.Builder
			j := i
			for ; j < len(row) && row[j].ink == row[i].ink; j++ {
				switch {
				case row[j].text != 0:
					run.WriteRune(row[j].text)
				case row[j].dots != 0:
					run.WriteRune(rune(0x2800 + int(row[j].dots)))
				default:
					run.WriteRune(' ')
				}
			}
			if p, ok := paint[row[i].ink]; ok {
				b.WriteString(p(run.String()))
			} else if j < len(row) || row[i].ink != gaugeBlank {
				b.WriteString(run.String())
			}
			i = j
		}
		out = append(out, b.String())
	}
	return out
}

func gaugeBar(f style.Frame) string {
	frac := f.Progress
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	fill := int(frac*float64(f.Width) + 0.5)
	return f.Accent(strings.Repeat("━", fill)) + f.Dim(strings.Repeat("━", f.Width-fill))
}
