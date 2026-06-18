package cli

import (
	"os"
	"os/exec"
	"strconv"
)

func runJournalTail(unit string, lines int) error {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return err
	}
	cmd := exec.Command("journalctl", "-u", unit, "-n", strconv.Itoa(lines), "--no-pager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
