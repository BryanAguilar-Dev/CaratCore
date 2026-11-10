// Package term colours text for the terminal with ANSI escape codes, and only
// when it makes sense: output to a terminal, NO_COLOR empty (no-color.org)
// and TERM not "dumb". Its zero value never colours, so tests and redirected
// output stay plain text.
package term

import "os"

// Style decides whether text gets colour codes.
type Style struct {
	on bool
}

// For returns the Style for writing to f.
func For(f *os.File) Style {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return Style{}
	}
	info, err := f.Stat()
	if err != nil {
		return Style{}
	}
	return Style{on: info.Mode()&os.ModeCharDevice != 0}
}

func (s Style) wrap(code, text string) string {
	if !s.on {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// Bold is for step titles.
func (s Style) Bold(text string) string { return s.wrap("1", text) }

// Dim is for secondary text, such as the command about to run.
func (s Style) Dim(text string) string { return s.wrap("2", text) }

// Green is for success.
func (s Style) Green(text string) string { return s.wrap("32", text) }

// Red is for failure.
func (s Style) Red(text string) string { return s.wrap("31", text) }

// Yellow is for things the user must act on.
func (s Style) Yellow(text string) string { return s.wrap("33", text) }
