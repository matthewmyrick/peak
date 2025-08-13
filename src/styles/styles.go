package styles

import "github.com/charmbracelet/lipgloss"

var (
	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	SelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)

	NormalStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	FolderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	ItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	HeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true).
		Padding(0, 1)
)