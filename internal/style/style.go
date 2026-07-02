// Package style provides semantic color/symbol formatting for cenv's CLI
// output. Every colored status message pairs its color with a symbol
// (✓/✗/⚠/→) so meaning survives when color is disabled — piped output,
// --no-color, NO_COLOR, or a non-color terminal. Whether color is actually
// emitted is controlled by fatih/color's package-level color.NoColor
// switch, set once at startup by cmd/cenv/main.go; this package never
// checks TTY/env state itself.
package style

import "github.com/fatih/color"

// Success formats a success message: green text, "✓ " prefix.
func Success(format string, args ...any) string {
	return color.GreenString("✓ "+format, args...)
}

// Error formats an error message: red text, "✗ " prefix.
func Error(format string, args ...any) string {
	return color.RedString("✗ "+format, args...)
}

// Warning formats a warning message: yellow text, "⚠ " prefix.
func Warning(format string, args ...any) string {
	return color.YellowString("⚠ "+format, args...)
}

// Info formats an informational message: blue text, "→ " prefix.
func Info(format string, args ...any) string {
	return color.BlueString("→ "+format, args...)
}

// Secondary de-emphasizes text: gray, no symbol prefix. For metadata and
// de-emphasized values (e.g. "no" in an auth-status column), not statuses.
func Secondary(text string) string {
	return color.HiBlackString("%s", text)
}

// Green highlights text as affirmative without a status-line symbol
// prefix: for table cells where a symbol on every row would be redundant
// with the column header (e.g. "yes" in an auth-status column).
func Green(text string) string {
	return color.GreenString("%s", text)
}
