package tui

import (
	"bytes"
	"os/exec"
)

// copyToClipboard attempts to write text to the system clipboard. Best-effort.
func copyToClipboard(text string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = bytes.NewBufferString(text)
	_ = cmd.Run()
}

// pasteFromClipboard attempts to read text from the system clipboard. Returns
// an empty string on failure.
func pasteFromClipboard() string {
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
