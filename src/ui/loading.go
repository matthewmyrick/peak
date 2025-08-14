package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"peek/src/styles"
)

type LoadingSpinner struct {
	Message   string
	StartTime time.Time
	frames    []string
	frame     int
}

func NewLoadingSpinner(message string) *LoadingSpinner {
	return &LoadingSpinner{
		Message:   message,
		StartTime: time.Now(),
		frames:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		frame:     0,
	}
}

func (ls *LoadingSpinner) Update() {
	ls.frame = (ls.frame + 1) % len(ls.frames)
}

func (ls *LoadingSpinner) Render(width, height int) string {
	elapsed := time.Since(ls.StartTime)

	// Create loading content
	spinner := ls.frames[ls.frame]
	message := fmt.Sprintf("%s %s", spinner, ls.Message)
	elapsedStr := fmt.Sprintf("Time elapsed: %s", elapsed.Round(time.Second))

	// Style the loading screen
	loadingStyle := styles.NormalStyle.
		Bold(true).
		Foreground(lipgloss.Color("39"))

	timeStyle := styles.NormalStyle.
		Foreground(lipgloss.Color("243"))

	// Center the content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		loadingStyle.Render(message),
		"",
		timeStyle.Render(elapsedStr),
	)

	// Create a border box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(2, 4).
		Width(width / 2).
		Height(height / 3)

	box := boxStyle.Render(content)

	// Center the box on screen
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
