package tui

import (
	"bytes"
	"os/exec"
	"sync"
)

var (
	fallbackClipboardMu sync.Mutex
	fallbackClipboard   string
)

// copyToClipboard attempts to write text to the system clipboard. Best-effort.
func copyToClipboard(text string) {
	// macOS
	if err := runCopyCmd("pbcopy", nil, text); err == nil {
		return
	}
	// Wayland (Linux)
	if err := runCopyCmd("wl-copy", nil, text); err == nil {
		return
	}
	// X11 (Linux)
	if err := runCopyCmd("xclip", []string{"-selection", "clipboard"}, text); err == nil {
		return
	}
	if err := runCopyCmd("xsel", []string{"--clipboard", "--input"}, text); err == nil {
		return
	}

	// Last resort: keep an in-memory clipboard so the app and tests behave
	// consistently even when no system clipboard tool is available.
	fallbackClipboardMu.Lock()
	fallbackClipboard = text
	fallbackClipboardMu.Unlock()
}

// pasteFromClipboard attempts to read text from the system clipboard. Returns
// an empty string on failure.
func pasteFromClipboard() string {
	// macOS
	if out, err := exec.Command("pbpaste").Output(); err == nil {
		return string(out)
	}
	// Wayland (Linux)
	if out, err := exec.Command("wl-paste", "--no-newline").Output(); err == nil {
		return string(out)
	}
	// X11 (Linux)
	if out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output(); err == nil {
		return string(out)
	}
	if out, err := exec.Command("xsel", "--clipboard", "--output").Output(); err == nil {
		return string(out)
	}

	fallbackClipboardMu.Lock()
	defer fallbackClipboardMu.Unlock()
	return fallbackClipboard
}

func runCopyCmd(name string, args []string, text string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = bytes.NewBufferString(text)
	return cmd.Run()
}
