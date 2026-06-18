package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var stdinReader = bufio.NewReader(os.Stdin)

func readLine(prompt string) string {
	if prompt != "" {
		fmt.Print(prompt)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil && err != io.EOF {
		return ""
	}
	return strings.TrimSpace(line)
}

func readPassword(prompt string) string {
	// Same as readLine for now; terminals often echo password input.
	return readLine(prompt)
}

func pause() {
	readLine("\nPress Enter to continue...")
}
