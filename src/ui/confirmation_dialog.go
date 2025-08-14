package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"peek/src/styles"
)

type ConfirmationDialog struct {
	isOpen      bool
	title       string
	message     string
	podName     string
	namespace   string
	action      string // "delete" or "restart"
	confirmed   bool
	cursor      int // 0 = Yes, 1 = No
}

func NewConfirmationDialog() *ConfirmationDialog {
	return &ConfirmationDialog{
		isOpen:    false,
		confirmed: false,
		cursor:    1, // Default to "No" for safety
	}
}

func (cd *ConfirmationDialog) Open(action, podName, namespace string) {
	cd.isOpen = true
	cd.podName = podName
	cd.namespace = namespace
	cd.action = action
	cd.confirmed = false
	cd.cursor = 1 // Default to "No"

	if action == "delete" {
		cd.title = "⚠️  Delete Pod"
		cd.message = "This will permanently delete the pod. The pod controller may recreate it."
	} else if action == "restart" {
		cd.title = "🔄 Restart Pod"
		cd.message = "This will delete and recreate the pod. There may be brief downtime."
	}
}

func (cd *ConfirmationDialog) Close() {
	cd.isOpen = false
	cd.confirmed = false
	cd.cursor = 1
}

func (cd *ConfirmationDialog) IsOpen() bool {
	return cd.isOpen
}

func (cd *ConfirmationDialog) MoveLeft() {
	cd.cursor = 0 // Yes
}

func (cd *ConfirmationDialog) MoveRight() {
	cd.cursor = 1 // No
}

func (cd *ConfirmationDialog) Confirm() bool {
	cd.confirmed = (cd.cursor == 0)
	cd.Close()
	return cd.confirmed
}

func (cd *ConfirmationDialog) GetAction() string {
	return cd.action
}

func (cd *ConfirmationDialog) Render(screenWidth, screenHeight int) string {
	if !cd.isOpen {
		return ""
	}

	var content strings.Builder

	// Title
	titleStyle := styles.NormalStyle.Bold(true).Foreground(lipgloss.Color("196"))
	content.WriteString(titleStyle.Render(cd.title) + "\n\n")

	// Pod information
	podStyle := styles.NormalStyle.Bold(true)
	content.WriteString(podStyle.Render("Pod: ") + cd.podName + "\n")
	content.WriteString(podStyle.Render("Namespace: ") + cd.namespace + "\n\n")

	// Message
	messageStyle := styles.NormalStyle.Foreground(lipgloss.Color("252"))
	content.WriteString(messageStyle.Render(cd.message) + "\n\n")

	// Warning
	warningStyle := styles.NormalStyle.Foreground(lipgloss.Color("226")).Italic(true)
	content.WriteString(warningStyle.Render("Are you sure you want to continue?") + "\n\n")

	// Buttons
	yesStyle := styles.NormalStyle.Padding(0, 2).Border(lipgloss.NormalBorder())
	noStyle := styles.NormalStyle.Padding(0, 2).Border(lipgloss.NormalBorder())

	if cd.cursor == 0 { // Yes selected
		yesStyle = yesStyle.Background(lipgloss.Color("196")).Foreground(lipgloss.Color("255")).Bold(true)
		noStyle = noStyle.Foreground(lipgloss.Color("240"))
	} else { // No selected
		yesStyle = yesStyle.Foreground(lipgloss.Color("240"))
		noStyle = noStyle.Background(lipgloss.Color("46")).Foreground(lipgloss.Color("0")).Bold(true)
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		yesStyle.Render("Yes"),
		"  ",
		noStyle.Render("No"),
	)
	content.WriteString(buttons + "\n\n")

	// Controls
	controlsStyle := styles.NormalStyle.Foreground(lipgloss.Color("240")).Italic(true)
	content.WriteString(controlsStyle.Render("Use ←→ to select, Enter to confirm, Esc to cancel"))

	// Create the dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Background(lipgloss.Color("235")).
		Padding(2).
		Width(60).
		Align(lipgloss.Center)

	dialog := dialogStyle.Render(content.String())

	// Center the dialog on the screen
	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}