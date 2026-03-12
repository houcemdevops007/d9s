// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package tui

import (
	"os"
	"os/signal"
	"syscall"
)

// Terminal provides raw terminal I/O.
type Terminal struct {
	origState syscall.Termios
}

// NewTerminal allocates a Terminal.
func NewTerminal() *Terminal {
	return &Terminal{}
}

// SetRaw puts the terminal into raw mode (character-by-character, no echo).
func (t *Terminal) SetRaw() error {
	raw, err := getTermios(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	t.origState = raw

	rawNew := raw
	rawNew.Lflag &^= syscall.ECHO | syscall.ECHOE | syscall.ECHOK | syscall.ECHONL
	rawNew.Lflag &^= syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	rawNew.Iflag &^= syscall.IXON | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.BRKINT
	rawNew.Oflag &^= syscall.OPOST
	rawNew.Cc[syscall.VMIN] = 1
	rawNew.Cc[syscall.VTIME] = 0

	return setTermios(int(os.Stdin.Fd()), rawNew)
}

// Restore restores the original terminal state.
func (t *Terminal) Restore() {
	setTermios(int(os.Stdin.Fd()), t.origState) //nolint
}

// Size returns the current terminal width and height.
func (t *Terminal) Size() (int, int) {
	return getTermSize()
}

// NotifyResize returns a channel that fires on SIGWINCH.
func NotifyResize() chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	return ch
}

// Key represents a single keypress event.
type Key struct {
	Rune  rune
	IsKey bool
	Code  KeyCode
}

// KeyCode enumerates special key codes.
type KeyCode int

const (
	KeyNone  KeyCode = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyEsc
	KeyTab
	KeyBackspace
	KeyCtrlC
	KeyCtrlD
	KeyCtrlL
	KeyDelete
)

// ReadKey reads a single keypress from stdin (blocking).
func ReadKey() Key {
	buf := make([]byte, 8)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return Key{Code: KeyCtrlD}
	}

	switch {
	case n == 1 && buf[0] == 3:
		return Key{IsKey: true, Code: KeyCtrlC}
	case n == 1 && buf[0] == 4:
		return Key{IsKey: true, Code: KeyCtrlD}
	case n == 1 && buf[0] == 12:
		return Key{IsKey: true, Code: KeyCtrlL}
	case n == 1 && buf[0] == 13:
		return Key{IsKey: true, Code: KeyEnter}
	case n == 1 && buf[0] == 27:
		return Key{IsKey: true, Code: KeyEsc}
	case n == 1 && buf[0] == 9:
		return Key{IsKey: true, Code: KeyTab}
	case n == 1 && (buf[0] == 127 || buf[0] == 8):
		return Key{IsKey: true, Code: KeyBackspace}
	case n >= 3 && buf[0] == 27 && buf[1] == '[':
		switch buf[2] {
		case 'A':
			return Key{IsKey: true, Code: KeyUp}
		case 'B':
			return Key{IsKey: true, Code: KeyDown}
		case 'C':
			return Key{IsKey: true, Code: KeyRight}
		case 'D':
			return Key{IsKey: true, Code: KeyLeft}
		case '3':
			if n >= 4 && buf[3] == '~' {
				return Key{IsKey: true, Code: KeyDelete}
			}
		}
	}

	if buf[0] >= 32 && buf[0] < 127 {
		return Key{Rune: rune(buf[0])}
	}
	return Key{}
}
