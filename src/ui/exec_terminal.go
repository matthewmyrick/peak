package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"peek/src/styles"
)

type ExecTerminal struct {
	isOpen      bool
	podName     string
	namespace   string
	contextName string
}

func NewExecTerminal() *ExecTerminal {
	return &ExecTerminal{
		isOpen: false,
	}
}

func (et *ExecTerminal) Open(contextName, namespace, podName string) {
	et.isOpen = true
	et.podName = podName
	et.namespace = namespace
	et.contextName = contextName
}

func (et *ExecTerminal) Close() {
	et.isOpen = false
}

func (et *ExecTerminal) IsOpen() bool {
	return et.isOpen
}

func (et *ExecTerminal) Render(screenWidth, screenHeight int) string {
	if !et.isOpen {
		return ""
	}

	var content strings.Builder

	// Header
	headerStyle := styles.NormalStyle.Bold(true).Foreground(lipgloss.Color("39"))
	title := fmt.Sprintf("🖥️  SSH/Exec into Pod: %s", et.podName)
	content.WriteString(headerStyle.Render(title) + "\n\n")

	// Pod information
	infoStyle := styles.NormalStyle.Bold(true)
	content.WriteString(infoStyle.Render("Namespace: ") + et.namespace + "\n")
	content.WriteString(infoStyle.Render("Context: ") + et.contextName + "\n\n")

	// Instructions
	instructionStyle := styles.NormalStyle.Foreground(lipgloss.Color("252"))
	content.WriteString(instructionStyle.Render("To exec into this pod, run the following command in your terminal:") + "\n\n")

	// Command
	commandStyle := styles.NormalStyle.
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("46")).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	command := fmt.Sprintf("kubectl exec -it %s -n %s --context %s -- /bin/bash", 
		et.podName, et.namespace, et.contextName)
	
	content.WriteString(commandStyle.Render(command) + "\n\n")

	// Alternative commands
	altStyle := styles.NormalStyle.Foreground(lipgloss.Color("245"))
	content.WriteString(altStyle.Render("Alternative shells if bash is not available:") + "\n\n")

	// Shell alternatives
	shells := []string{
		fmt.Sprintf("kubectl exec -it %s -n %s --context %s -- /bin/sh", et.podName, et.namespace, et.contextName),
		fmt.Sprintf("kubectl exec -it %s -n %s --context %s -- /bin/ash", et.podName, et.namespace, et.contextName),
		fmt.Sprintf("kubectl exec -it %s -n %s --context %s -- /bin/zsh", et.podName, et.namespace, et.contextName),
	}

	shellStyle := styles.NormalStyle.
		Background(lipgloss.Color("237")).
		Foreground(lipgloss.Color("226")).
		Padding(0, 1)

	for i, shell := range shells {
		content.WriteString(shellStyle.Render(shell))
		if i < len(shells)-1 {
			content.WriteString("\n")
		}
	}

	content.WriteString("\n\n")

	// Note
	noteStyle := styles.NormalStyle.Foreground(lipgloss.Color("240")).Italic(true)
	content.WriteString(noteStyle.Render("Note: Copy and paste the command into your terminal. Press Esc to close this dialog."))

	// Create the dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(2).
		Width(90).
		Align(lipgloss.Left)

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