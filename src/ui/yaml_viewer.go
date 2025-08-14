package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"peek/src/k8s"
	"peek/src/styles"
)

type YAMLViewer struct {
	isOpen       bool
	podName      string
	namespace    string
	kubeConfig   *k8s.KubeConfig
	contextName  string
	yamlContent  string
	scrollOffset int
	error        error
	isLoading    bool
}

func NewYAMLViewer() *YAMLViewer {
	return &YAMLViewer{
		isOpen:       false,
		yamlContent:  "",
		scrollOffset: 0,
		isLoading:    false,
	}
}

func (yv *YAMLViewer) Open(kubeConfig *k8s.KubeConfig, contextName, namespace, podName string) {
	yv.isOpen = true
	yv.podName = podName
	yv.namespace = namespace
	yv.kubeConfig = kubeConfig
	yv.contextName = contextName
	yv.yamlContent = ""
	yv.scrollOffset = 0
	yv.error = nil
	yv.isLoading = true

	// Start fetching YAML
	go yv.fetchYAML()
}

func (yv *YAMLViewer) Close() {
	yv.isOpen = false
	yv.yamlContent = ""
	yv.scrollOffset = 0
	yv.isLoading = false
}

func (yv *YAMLViewer) IsOpen() bool {
	return yv.isOpen
}

func (yv *YAMLViewer) ScrollUp() {
	if yv.scrollOffset > 0 {
		yv.scrollOffset--
	}
}

func (yv *YAMLViewer) ScrollDown() {
	lines := strings.Split(yv.yamlContent, "\n")
	maxScroll := len(lines) - 20 // Assuming 20 visible lines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if yv.scrollOffset < maxScroll {
		yv.scrollOffset++
	}
}

func (yv *YAMLViewer) PageUp() {
	yv.scrollOffset -= 10
	if yv.scrollOffset < 0 {
		yv.scrollOffset = 0
	}
}

func (yv *YAMLViewer) PageDown() {
	lines := strings.Split(yv.yamlContent, "\n")
	maxScroll := len(lines) - 20
	if maxScroll < 0 {
		maxScroll = 0
	}
	yv.scrollOffset += 10
	if yv.scrollOffset > maxScroll {
		yv.scrollOffset = maxScroll
	}
}

func (yv *YAMLViewer) fetchYAML() {
	yaml, err := yv.kubeConfig.GetPodYAML(yv.contextName, yv.namespace, yv.podName)
	if err != nil {
		yv.error = err
	} else {
		yv.yamlContent = yaml
	}
	yv.isLoading = false
}

func (yv *YAMLViewer) Render(screenWidth, screenHeight int) string {
	if !yv.isOpen {
		return ""
	}

	// Calculate dimensions
	width := screenWidth - 4
	height := screenHeight - 4
	if width < 60 {
		width = 60
	}
	if height < 15 {
		height = 15
	}

	var content strings.Builder

	// Header
	headerStyle := styles.NormalStyle.Bold(true).Foreground(lipgloss.Color("39"))
	title := fmt.Sprintf("📄 YAML: %s", yv.podName)
	content.WriteString(headerStyle.Render(title) + "\n")

	// Status line
	statusStyle := styles.NormalStyle.Foreground(lipgloss.Color("245"))
	status := fmt.Sprintf("Namespace: %s", yv.namespace)
	content.WriteString(statusStyle.Render(status) + "\n")

	// Controls
	controlsStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
	controls := "↑↓=scroll PgUp/PgDn=page Esc=close"
	content.WriteString(controlsStyle.Render(controls) + "\n\n")

	// Content
	if yv.isLoading {
		content.WriteString(styles.NormalStyle.Render("Loading YAML..."))
	} else if yv.error != nil {
		errorStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", yv.error)))
	} else if yv.yamlContent == "" {
		content.WriteString(styles.NormalStyle.Render("No YAML content available"))
	} else {
		// Render YAML content
		content.WriteString(yv.renderYAML(height - 6)) // Reserve space for header and controls
	}

	// Create the box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Background(lipgloss.Color("235")).
		Padding(1).
		Width(width).
		Height(height)

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

func (yv *YAMLViewer) renderYAML(maxLines int) string {
	if yv.yamlContent == "" {
		return "No content"
	}

	lines := strings.Split(yv.yamlContent, "\n")
	var result strings.Builder

	// Calculate which lines to show
	startLine := yv.scrollOffset
	endLine := startLine + maxLines
	
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine >= len(lines) {
		startLine = len(lines) - 1
		if startLine < 0 {
			startLine = 0
		}
	}

	// Show YAML with syntax highlighting
	for i := startLine; i < endLine; i++ {
		line := lines[i]
		styledLine := yv.styleYAMLLine(line)
		result.WriteString(styledLine)
		if i < endLine-1 {
			result.WriteString("\n")
		}
	}

	// Show scroll indicator
	if len(lines) > maxLines {
		scrollInfo := fmt.Sprintf("\nShowing lines %d-%d of %d", startLine+1, endLine, len(lines))
		scrollStyle := styles.NormalStyle.Foreground(lipgloss.Color("240")).Italic(true)
		result.WriteString(scrollStyle.Render(scrollInfo))
	}

	return result.String()
}

func (yv *YAMLViewer) styleYAMLLine(line string) string {
	// Basic YAML syntax highlighting
	trimmed := strings.TrimSpace(line)
	
	// Comments
	if strings.HasPrefix(trimmed, "#") {
		return styles.NormalStyle.Foreground(lipgloss.Color("240")).Render(line)
	}
	
	// Keys (lines ending with :)
	if strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "-") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			keyStyle := styles.NormalStyle.Foreground(lipgloss.Color("39")).Bold(true)
			valueStyle := styles.NormalStyle.Foreground(lipgloss.Color("252"))
			return keyStyle.Render(parts[0]+":") + valueStyle.Render(parts[1])
		}
	}
	
	// List items (lines starting with -)
	if strings.HasPrefix(trimmed, "-") {
		listStyle := styles.NormalStyle.Foreground(lipgloss.Color("226"))
		return listStyle.Render(line)
	}
	
	// String values (containing quotes)
	if strings.Contains(line, `"`) || strings.Contains(line, `'`) {
		stringStyle := styles.NormalStyle.Foreground(lipgloss.Color("46"))
		return stringStyle.Render(line)
	}
	
	// Numbers and booleans
	if strings.Contains(trimmed, "true") || strings.Contains(trimmed, "false") ||
		strings.Contains(trimmed, "null") || strings.Contains(trimmed, "~") {
		boolStyle := styles.NormalStyle.Foreground(lipgloss.Color("208"))
		return boolStyle.Render(line)
	}
	
	// Default
	return styles.NormalStyle.Render(line)
}