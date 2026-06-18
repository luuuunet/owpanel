package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func runJournalTail(unit string, lines int) error {
	if _, err := exec.LookPath("journalctl"); err != nil {
		return err
	}
	args := []string{"-u", unit, "-n", strconv.Itoa(lines), "--no-pager"}
	if err := execJournalctl(append([]string{"journalctl"}, args...)); err == nil {
		return nil
	} else if os.Getuid() == 0 {
		return err
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return fmt.Errorf("journalctl failed (try: sudo op logs): %w", err)
	}
	return execJournalctl(append([]string{"sudo", "journalctl"}, args...))
}

func execJournalctl(argv []string) error {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
