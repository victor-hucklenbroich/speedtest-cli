// Package style is the API TUI styles are written against.
//
// A style is a single Go file that builds a Style and passes it to Register
// from init. Built-in styles are compiled in (see style/builtin). User styles
// are .go files in the styles directory next to config.yml; they are run by
// an embedded Go interpreter at startup, so no rebuild is needed. Both kinds
// use the same file format: style/builtin/digits.go doubles as a template.
package style

// Frame is what a style draws from. The TUI builds a fresh one for every
// element on every animation frame.
type Frame struct {
	// Phase is "download" or "upload".
	Phase string
	// Mbps is the current speed, eased between samples for smooth animation.
	Mbps float64
	// History holds recent throughput samples in Mbps, oldest first.
	History []float64
	// Progress is the fraction of the running transfer done, from 0 to 1.
	Progress float64
	// Width is the number of columns the element may use.
	Width int

	// Accent renders text in the phase color, Dim renders it muted.
	Accent func(string) string
	Dim    func(string) string
}

// Style renders the animated parts of the TUI.
type Style struct {
	Name string
	// Live draws the readout shown while a phase runs. Required.
	Live func(f Frame) []string
	// Bar draws the progress bar of the running transfer. Optional: nil
	// falls back to the default style's bar.
	Bar func(f Frame) string
}

var registry []Style

// Register makes s selectable by name. Registering an existing name replaces
// that style, so a user style can override a built-in one.
func Register(s Style) {
	if s.Name == "" || s.Live == nil {
		panic("style: Register needs a Name and a Live func")
	}
	for i := range registry {
		if registry[i].Name == s.Name {
			registry[i] = s
			return
		}
	}
	registry = append(registry, s)
}

// Lookup returns the style registered under name.
func Lookup(name string) (Style, bool) {
	for _, s := range registry {
		if s.Name == name {
			return s, true
		}
	}
	return Style{}, false
}
