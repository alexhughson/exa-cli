//go:build darwin

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func readSecret(prompt string, stdin io.Reader, stderr io.Writer) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return readSecretFallback(prompt, stdin, stderr)
	}
	defer tty.Close()

	var original syscall.Termios
	if err := ioctl(tty.Fd(), syscall.TIOCGETA, uintptr(unsafe.Pointer(&original))); err != nil {
		return readSecretFallback(prompt, stdin, stderr)
	}

	hidden := original
	hidden.Lflag &^= syscall.ECHO
	if err := ioctl(tty.Fd(), syscall.TIOCSETA, uintptr(unsafe.Pointer(&hidden))); err != nil {
		return readSecretFallback(prompt, stdin, stderr)
	}
	defer func() {
		_ = ioctl(tty.Fd(), syscall.TIOCSETA, uintptr(unsafe.Pointer(&original)))
	}()

	fmt.Fprint(tty, prompt)
	scanner := bufio.NewScanner(tty)
	if !scanner.Scan() {
		fmt.Fprintln(tty)
		return "", scanner.Err()
	}
	fmt.Fprintln(tty)
	return strings.TrimSpace(scanner.Text()), nil
}

func ioctl(fd uintptr, request uint, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(request), arg)
	if errno != 0 {
		return errno
	}
	return nil
}
