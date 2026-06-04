package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func readSecretFallback(prompt string, stdin io.Reader, stderr io.Writer) (string, error) {
	fmt.Fprint(stderr, prompt)
	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		return "", scanner.Err()
	}
	return strings.TrimSpace(scanner.Text()), nil
}
