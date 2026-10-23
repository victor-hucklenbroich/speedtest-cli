package stylepack

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"speedtest/style"
	"speedtest/style/builtin"
)

func write(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testFrame() style.Frame {
	plain := func(s string) string { return s }
	return style.Frame{
		Phase:    "download",
		Mbps:     123.4,
		History:  []float64{10, 50, 30},
		Progress: 0.5,
		Width:    24,
		Accent:   plain,
		Dim:      plain,
	}
}

// The built-in style is the template users copy, so it must keep working
// unchanged as an interpreted user style.
func TestBuiltinLoadsAsUserStyle(t *testing.T) {
	src, err := os.ReadFile("../../style/builtin/digits.go")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write(t, dir, "copy.go", strings.Replace(string(src), `Name: "digits"`, `Name: "digits-copy"`, 1))

	if errs := Load(dir); len(errs) > 0 {
		t.Fatal(errs)
	}
	got, ok := style.Lookup("digits-copy")
	if !ok {
		t.Fatal("digits-copy was not registered")
	}
	f := testFrame()
	if want, have := builtin.Digits.Live(f), got.Live(f); !reflect.DeepEqual(want, have) {
		t.Errorf("Live differs from compiled style:\nwant %q\nhave %q", want, have)
	}
	if want, have := builtin.Digits.Bar(f), got.Bar(f); want != have {
		t.Errorf("Bar differs from compiled style: want %q, have %q", want, have)
	}
}

func TestBadFilesAreReportedAndSkipped(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a_syntax.go", "package x\nfunc {")
	write(t, dir, "b_unnamed.go", `package x
import "speedtest/style"
func init() { style.Register(style.Style{}) }`)
	write(t, dir, "c_ok.go", `package x
import "speedtest/style"
func init() {
	style.Register(style.Style{Name: "ok", Live: func(f style.Frame) []string { return []string{"ok"} }})
}`)
	write(t, dir, "notes.txt", "not a style")

	if errs := Load(dir); len(errs) != 2 {
		t.Fatalf("want 2 errors, got %d: %v", len(errs), errs)
	}
	if _, ok := style.Lookup("ok"); !ok {
		t.Error("valid style next to broken ones was not registered")
	}
}

// The TUI recovers from panics in a style's render funcs; that only works if
// interpreted panics surface as ordinary Go panics.
func TestRenderPanicIsRecoverable(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "boom.go", `package boom
import "speedtest/style"
func init() {
	style.Register(style.Style{Name: "boom", Live: func(f style.Frame) []string { panic("boom") }})
}`)
	if errs := Load(dir); len(errs) > 0 {
		t.Fatal(errs)
	}
	s, _ := style.Lookup("boom")
	defer func() {
		if recover() == nil {
			t.Error("want a recoverable panic from the interpreted Live")
		}
	}()
	s.Live(testFrame())
}

func TestMissingDirIsCreated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "styles")
	if errs := Load(dir); len(errs) > 0 {
		t.Fatal(errs)
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		t.Errorf("styles dir not created: %v", err)
	}
}
