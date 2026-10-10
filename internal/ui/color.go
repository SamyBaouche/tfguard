// Package ui provides terminal styling, banners, and animated progress for the CLI.
package ui

import (
	"io"
	"os"
)

// ANSI color / style codes.
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	cyan    = "\033[36m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	red     = "\033[31m"
	magenta = "\033[35m"
	white   = "\033[37m"
	blue    = "\033[34m"
)

// Style wraps an io.Writer with optional ANSI coloring.
type Style struct {
	enabled bool
}

// NewStyle enables color when out is a TTY and NO_COLOR is unset.
func NewStyle(out io.Writer) Style {
	if os.Getenv("NO_COLOR") != "" {
		return Style{enabled: false}
	}
	f, ok := out.(*os.File)
	if !ok {
		return Style{enabled: false}
	}
	info, err := f.Stat()
	if err != nil {
		return Style{enabled: false}
	}
	return Style{enabled: (info.Mode() & os.ModeCharDevice) != 0}
}

// Enabled reports whether ANSI styles are active.
func (s Style) Enabled() bool { return s.enabled }

func (s Style) paint(code, text string) string {
	if !s.enabled {
		return text
	}
	return code + text + reset
}

func (s Style) Bold(text string) string    { return s.paint(bold, text) }
func (s Style) Dim(text string) string     { return s.paint(dim, text) }
func (s Style) Cyan(text string) string    { return s.paint(cyan, text) }
func (s Style) Green(text string) string   { return s.paint(green, text) }
func (s Style) Yellow(text string) string  { return s.paint(yellow, text) }
func (s Style) Red(text string) string     { return s.paint(red, text) }
func (s Style) Magenta(text string) string { return s.paint(magenta, text) }
func (s Style) White(text string) string   { return s.paint(white, text) }
func (s Style) Blue(text string) string    { return s.paint(blue, text) }

// Risk colors a risk / severity label.
func (s Style) Risk(level string) string {
	switch level {
	case "SAFE", "LOW":
		return s.Green(level)
	case "CAUTION", "MEDIUM":
		return s.Yellow(level)
	case "DANGER", "HIGH":
		return s.Red(level)
	case "CRITICAL":
		return s.paint(bold+magenta, level)
	default:
		return level
	}
}