package style

import (
	"strings"
	"testing"

	"github.com/fatih/color"
)

func withColor(t *testing.T, enabled bool, fn func()) {
	t.Helper()
	orig := color.NoColor
	color.NoColor = !enabled
	t.Cleanup(func() { color.NoColor = orig })
	fn()
}

func TestSymbolHelpers(t *testing.T) {
	cases := []struct {
		name string
		fn   func() string
		want string // plain text, including symbol
		code string // ANSI SGR code fatih/color should use
	}{
		{"Success", func() string { return Success("Created %q", "foo") }, `✓ Created "foo"`, "32"},
		{"Error", func() string { return Error("environment %q not found", "foo") }, `✗ environment "foo" not found`, "31"},
		{"Warning", func() string { return Warning("disk usage high") }, `⚠ disk usage high`, "33"},
		{"Info", func() string { return Info("Using %q (%s)", "foo", "/tmp/foo") }, `→ Using "foo" (/tmp/foo)`, "34"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/color disabled", func(t *testing.T) {
			withColor(t, false, func() {
				if got := tc.fn(); got != tc.want {
					t.Errorf("%s() = %q, want %q", tc.name, got, tc.want)
				}
			})
		})
		t.Run(tc.name+"/color enabled", func(t *testing.T) {
			withColor(t, true, func() {
				got := tc.fn()
				if !strings.Contains(got, tc.want) {
					t.Errorf("%s() = %q, want it to contain plain text %q", tc.name, got, tc.want)
				}
				if !strings.Contains(got, "\x1b["+tc.code) {
					t.Errorf("%s() = %q, want it to contain ANSI code %q", tc.name, got, tc.code)
				}
			})
		})
	}
}

func TestPlainColorHelpers(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) string
		code string
	}{
		{"Secondary", Secondary, "90"},
		{"Green", Green, "32"},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/color disabled", func(t *testing.T) {
			withColor(t, false, func() {
				if got := tc.fn("no"); got != "no" {
					t.Errorf("%s(%q) = %q, want %q", tc.name, "no", got, "no")
				}
			})
		})
		t.Run(tc.name+"/color enabled", func(t *testing.T) {
			withColor(t, true, func() {
				got := tc.fn("no")
				if !strings.Contains(got, "no") {
					t.Errorf("%s(%q) = %q, want it to contain %q", tc.name, "no", got, "no")
				}
				if !strings.Contains(got, "\x1b["+tc.code) {
					t.Errorf("%s(%q) = %q, want it to contain ANSI code %q", tc.name, "no", got, tc.code)
				}
			})
		})
	}
}
