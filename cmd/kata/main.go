package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/boatnoah/kata/internal/tui"
)

func main() {
	if _, err := tea.NewProgram(tui.NewApp(), tea.WithAltScreen()).Run(); err != nil {
		log.Fatal(err)
	}
}
