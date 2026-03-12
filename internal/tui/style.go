// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
// Package tui implements the terminal user interface for d9s using raw ANSI codes.
// It uses only Go standard library: os, syscall, fmt, strings, time, unicode.
package tui

import (
	"fmt"
	"strings"
	"time"
)

// Color constants using ANSI 256-color codes.
const (
	// Reset
	Reset = "\x1b[0m"

	// Text colors
	colorBlack   = "\x1b[30m"
	colorRed     = "\x1b[31m"
	colorGreen   = "\x1b[32m"
	colorYellow  = "\x1b[33m"
	colorBlue    = "\x1b[34m"
	colorMagenta = "\x1b[35m"
	colorCyan    = "\x1b[36m"
	colorWhite   = "\x1b[37m"

	// Bright colors
	colorBrightBlack   = "\x1b[90m"
	colorBrightRed     = "\x1b[91m"
	colorBrightGreen   = "\x1b[92m"
	colorBrightYellow  = "\x1b[93m"
	colorBrightBlue    = "\x1b[94m"
	colorBrightMagenta = "\x1b[95m"
	colorBrightCyan    = "\x1b[96m"
	colorBrightWhite   = "\x1b[97m"

	// Background colors
	bgBlack   = "\x1b[40m"
	bgRed     = "\x1b[41m"
	bgGreen   = "\x1b[42m"
	bgYellow  = "\x1b[43m"
	bgBlue    = "\x1b[44m"
	bgMagenta = "\x1b[45m"
	bgCyan    = "\x1b[46m"
	bgWhite   = "\x1b[47m"

	bgBrightBlack = "\x1b[100m"
	bgBrightBlue  = "\x1b[104m"

	// 256-color (foreground)
	fg256 = "\x1b[38;5;%dm"
	bg256 = "\x1b[48;5;%dm"

	// Text styles
	bold      = "\x1b[1m"
	dim       = "\x1b[2m"
	italic    = "\x1b[3m"
	underline = "\x1b[4m"
	reverse   = "\x1b[7m"
)

// Theme holds the UI color palette.
type Theme struct {
	// Brand colors
	Primary   string // primary accent (bright cyan)
	Secondary string // secondary accent (magenta)
	Success   string // green
	Warning   string // yellow
	Error     string // red
	Danger    string // red (alias for ease of use)
	Info      string // cyan (information)
	Muted     string // dim white

	// Backgrounds
	BgHeader  string
	BgSelected string
	BgPanel   string

	// Text
	TextHeader string
	TextNormal string
	TextDim    string
	TextBold   string

	// Container states
	StateRunning string
	StateExited  string
	StatePaused  string
}

// DefaultTheme returns the default dark theme.
func DefaultTheme() Theme {
	// Using 256 colors for richer palette
	return Theme{
		Primary:   fmt.Sprintf(fg256, 39),   // bright sky blue
		Secondary: fmt.Sprintf(fg256, 135),  // medium orchid
		Success:   fmt.Sprintf(fg256, 76),   // chartreuse green
		Warning:   fmt.Sprintf(fg256, 220),  // gold
		Error:     fmt.Sprintf(fg256, 196),  // bright red
		Danger:    fmt.Sprintf(fg256, 196),  // same as Error
		Info:      fmt.Sprintf(fg256, 45),   // bright turquoise
		Muted:     colorBrightBlack,

		BgHeader:   fmt.Sprintf(bg256, 17),   // dark navy
		BgSelected: fmt.Sprintf(bg256, 24),   // steel blue
		BgPanel:    "",                         // transparent

		TextHeader: bold + colorBrightWhite,
		TextNormal: colorWhite,
		TextDim:    colorBrightBlack,
		TextBold:   bold + colorBrightWhite,

		StateRunning: fmt.Sprintf(fg256, 76),  // green
		StateExited:  fmt.Sprintf(fg256, 196), // red
		StatePaused:  fmt.Sprintf(fg256, 220), // yellow
	}
}

// Screen holds terminal dimensions and the output buffer.
type Screen struct {
	Width  int
	Height int
	theme  Theme
	buf    strings.Builder
}

// NewScreen creates a Screen with the given dimensions.
func NewScreen(w, h int, theme Theme) *Screen {
	return &Screen{Width: w, Height: h, theme: theme}
}

// Clear clears the screen output buffer (not the terminal).
func (s *Screen) Clear() {
	s.buf.Reset()
}

// Write writes raw text to the buffer.
func (s *Screen) Write(text string) {
	s.buf.WriteString(text)
}

// Flush returns the buffer content.
func (s *Screen) Flush() string {
	return s.buf.String()
}

// MoveTo returns ANSI sequence to move cursor to row, col (1-indexed).
func MoveTo(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

// ClearLine returns ANSI sequence to clear to end of line.
func ClearLine() string {
	return "\x1b[K"
}

// SaveCursor saves cursor position.
func SaveCursor() string {
	return "\x1b[s"
}

// RestoreCursor restores cursor position.
func RestoreCursor() string {
	return "\x1b[u"
}

// HideCursor hides the cursor.
func HideCursor() string {
	return "\x1b[?25l"
}

// ShowCursor shows the cursor.
func ShowCursor() string {
	return "\x1b[?25h"
}

// Pad pads or truncates s to exactly width runes.
func Pad(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		if width <= 3 {
			return string(runes[:width])
		}
		return string(runes[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(runes))
}

// PadRight pads s to width from the right.
func PadRight(s string, width int) string {
	return Pad(s, width)
}

// PadLeft left-pads s to width.
func PadLeft(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(runes)) + s
}

// StateColor returns the color code for a container state.
func (t Theme) StateColor(state string) string {
	switch state {
	case "running":
		return t.StateRunning
	case "exited", "dead":
		return t.StateExited
	case "paused":
		return t.StatePaused
	default:
		return t.Muted
	}
}

// StateIcon returns a short icon for a container state.
func StateIcon(state string) string {
	switch state {
	case "running":
		return "▶"
	case "exited":
		return "■"
	case "paused":
		return "⏸"
	case "restarting":
		return "↻"
	case "created":
		return "+"
	default:
		return "?"
	}
}

// FormatBytes formats bytes to human-readable string.
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// TimeAgo formats a unix timestamp as "X ago" string.
func TimeAgo(unixSec int64) string {
	if unixSec == 0 {
		return "unknown"
	}
	d := time.Since(time.Unix(unixSec, 0))
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// HLine draws a horizontal line of given width using box-drawing characters.
func HLine(width int, char string) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat(char, width)
}

// Box draws a box of given width and height.
func Box(label string, width int) string {
	top := "┌" + HLine(width-2, "─") + "┐"
	title := "│ " + Pad(label, width-4) + " │"
	_ = title
	return top
}

// Separator draws a vertical separator character.
func Separator() string {
	return "│"
}

// ColorizePercent applies a color based on the percent value (0-100).
func ColorizePercent(theme Theme, pct float64) string {
	switch {
	case pct >= 90:
		return theme.Error
	case pct >= 70:
		return theme.Warning
	default:
		return theme.Success
	}
}
