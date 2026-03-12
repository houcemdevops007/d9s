// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
//go:build darwin

package tui

import (
	"syscall"
	"unsafe"
)

func getTermios(fd int) (syscall.Termios, error) {
	var t syscall.Termios
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		syscall.TIOCGETA,
		uintptr(unsafe.Pointer(&t)),
	)
	if errno != 0 {
		return t, errno
	}
	return t, nil
}

func setTermios(fd int, t syscall.Termios) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		syscall.TIOCSETA,
		uintptr(unsafe.Pointer(&t)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func getTermSize() (int, int) {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	var ws winsize
	syscall.Syscall(
		syscall.SYS_IOCTL,
		1,
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(&ws)),
	)
	if ws.Col == 0 {
		ws.Col = 120
	}
	if ws.Row == 0 {
		ws.Row = 40
	}
	return int(ws.Col), int(ws.Row)
}
