package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

var keylogEnabled = os.Getenv("KATA_KEYLOG") != ""

func logKey(msg tea.KeyMsg) {
	if !keylogEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "key: type=%v runes=%q alt=%v\n", msg.Type, string(msg.Runes), msg.Alt)
}
