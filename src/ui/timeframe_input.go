package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"peek/src/styles"
)

type TimeframeInput struct {
	isOpen      bool
	input       string
	placeholder string
	title       string
	width       int
	height      int
}

func NewTimeframeInput() *TimeframeInput {
	return &TimeframeInput{
		isOpen:      false,
		input:       "",
		placeholder: "Enter minutes (e.g., 30)",
		title:       "Change Timeframe",
		width:       50,
		height:      6,
	}
}

func (ti *TimeframeInput) Open() {
	ti.isOpen = true
	ti.input = ""
}

func (ti *TimeframeInput) Close() {
	ti.isOpen = false
	ti.input = ""
}

func (ti *TimeframeInput) IsOpen() bool {
	return ti.isOpen
}

func (ti *TimeframeInput) GetInput() string {
	return ti.input
}

func (ti *TimeframeInput) SetInput(input string) {
	ti.input = input
}

func (ti *TimeframeInput) AddChar(char string) {
	// Only allow numeric input
	if char >= "0" && char <= "9" {
		ti.input += char
	}
}

func (ti *TimeframeInput) Backspace() {
	if len(ti.input) > 0 {
		ti.input = ti.input[:len(ti.input)-1]
	}
}

func (ti *TimeframeInput) Render(screenWidth, screenHeight int) string {
	if !ti.isOpen {
		return ""
	}

	// Create the input box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1, 2).
		Width(ti.width - 4). // Account for padding and border
		Height(ti.height - 2) // Account for padding and border

	var content strings.Builder

	// Title
	titleStyle := styles.NormalStyle.
		Bold(true).
		Foreground(lipgloss.Color("39"))
	content.WriteString(titleStyle.Render(ti.title))
	content.WriteString("\n\n")

	// Input field
	inputFieldStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("240")).
		Width(ti.width - 8).
		Padding(0, 1)

	displayText := ti.input
	if displayText == "" {
		placeholderStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
		displayText = placeholderStyle.Render(ti.placeholder)
	} else {
		displayText = styles.NormalStyle.Render(displayText + "█") // Add cursor
	}

	content.WriteString(inputFieldStyle.Render(displayText))
	content.WriteString("\n\n")

	// Instructions
	instructStyle := styles.NormalStyle.
		Foreground(lipgloss.Color("245")).
		Italic(true)
	content.WriteString(instructStyle.Render("Press Enter to confirm • Esc to cancel"))

	// Render the complete box
	box := boxStyle.Render(content.String())

	// Center the box on the screen
	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}