//go:build !darwin

package cli

import (
	"io"
)

func readSecret(prompt string, stdin io.Reader, stderr io.Writer) (string, error) {
	return readSecretFallback(prompt, stdin, stderr)
}
