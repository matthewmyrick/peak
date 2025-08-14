package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Footer struct {
	Width int
}

type Command struct {
	Key         string
	Description string
}

func NewFooter(width int) *Footer {
	return &Footer{
		Width: width,
	}
}

func (f *Footer) GetNavigationCommands() []Command {
	return []Command{
		{"1", "Left Pane"},
		{"2", "Right Pane"},
		{"↑", "Up"},
		{"↓", "Down"},
		{"↵", "Select/Expand"},
		{"Esc", "Collapse"},
		{"/", "Search"},
		{"Ctrl+K", "Context"},
		{"Ctrl+N", "Namespace"},
		{"Ctrl+Q", "Quit"},
	}
}

func (f *Footer) GetSearchCommands() []Command {
	return []Command{
		{"Type", "Search"},
		{"↑", "Up"},
		{"↓", "Down"},
		{"↵", "Select"},
		{"Esc", "Exit Search"},
		{"Backspace", "Delete"},
		{"Ctrl+Q", "Quit"},
	}
}

func (f *Footer) GetNamespaceSelectorCommands() []Command {
	return []Command{
		{"Type", "Filter"},
		{"↑", "Up"},
		{"↓", "Down"},
		{"↵", "Select"},
		{"Esc", "Cancel"},
		{"Backspace", "Delete"},
		{"Ctrl+Q", "Quit"},
	}
}

func (f *Footer) Render(isSearchMode bool) string {
	var commands []Command

	if isSearchMode {
		commands = f.GetSearchCommands()
	} else {
		commands = f.GetNavigationCommands()
	}

	return f.renderCommands(commands)
}

func (f *Footer) RenderWithMode(isSearchMode bool, isNamespaceMode bool) string {
	var commands []Command

	if isNamespaceMode {
		commands = f.GetNamespaceSelectorCommands()
	} else if isSearchMode {
		commands = f.GetSearchCommands()
	} else {
		commands = f.GetNavigationCommands()
	}

	return f.renderCommands(commands)
}

func (f *Footer) renderCommands(commands []Command) string {
	var commandStrings []string

	for _, cmd := range commands {
		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("240")).
			Padding(0, 1).
			Bold(true)

		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

		commandStr := keyStyle.Render(cmd.Key) + " " + descStyle.Render(cmd.Description)
		commandStrings = append(commandStrings, commandStr)
	}

	content := strings.Join(commandStrings, "  ")

	footerStyle := lipgloss.NewStyle().
		Width(f.Width).
		Padding(0, 1).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252"))

	return footerStyle.Render(content)
}
