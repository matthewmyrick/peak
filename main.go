package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"peek/src/app"
)

func main() {
	p := tea.NewProgram(app.InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
