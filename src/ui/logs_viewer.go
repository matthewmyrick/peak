package ui

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"peek/src/k8s"
	"peek/src/styles"
)

type LogsViewer struct {
	isOpen       bool
	podName      string
	namespace    string
	containerName string
	kubeConfig   *k8s.KubeConfig
	contextName  string
	logs         []string
	scrollOffset int
	isFollowing  bool
	lastUpdate   time.Time
	error        error
	cancel       context.CancelFunc
}

func NewLogsViewer() *LogsViewer {
	return &LogsViewer{
		isOpen:      false,
		logs:        []string{},
		scrollOffset: 0,
		isFollowing: false,
	}
}

func (lv *LogsViewer) Open(kubeConfig *k8s.KubeConfig, contextName, namespace, podName, containerName string) {
	lv.isOpen = true
	lv.podName = podName
	lv.namespace = namespace
	lv.containerName = containerName
	lv.kubeConfig = kubeConfig
	lv.contextName = contextName
	lv.logs = []string{}
	lv.scrollOffset = 0
	lv.error = nil
	lv.isFollowing = true

	// Start fetching logs
	go lv.fetchLogs()
}

func (lv *LogsViewer) Close() {
	lv.isOpen = false
	lv.logs = []string{}
	lv.scrollOffset = 0
	lv.isFollowing = false
	if lv.cancel != nil {
		lv.cancel()
		lv.cancel = nil
	}
}

func (lv *LogsViewer) IsOpen() bool {
	return lv.isOpen
}

func (lv *LogsViewer) ScrollUp() {
	if lv.scrollOffset > 0 {
		lv.scrollOffset--
	}
	lv.isFollowing = false
}

func (lv *LogsViewer) ScrollDown() {
	maxScroll := len(lv.logs) - 20 // Assuming 20 visible lines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if lv.scrollOffset < maxScroll {
		lv.scrollOffset++
	}
	
	// Resume following if we're at the bottom
	if lv.scrollOffset >= maxScroll {
		lv.isFollowing = true
	}
}

func (lv *LogsViewer) PageUp() {
	lv.scrollOffset -= 10
	if lv.scrollOffset < 0 {
		lv.scrollOffset = 0
	}
	lv.isFollowing = false
}

func (lv *LogsViewer) PageDown() {
	maxScroll := len(lv.logs) - 20
	if maxScroll < 0 {
		maxScroll = 0
	}
	lv.scrollOffset += 10
	if lv.scrollOffset > maxScroll {
		lv.scrollOffset = maxScroll
	}
	
	// Resume following if we're at the bottom
	if lv.scrollOffset >= maxScroll {
		lv.isFollowing = true
	}
}

func (lv *LogsViewer) ToggleFollow() {
	lv.isFollowing = !lv.isFollowing
	if lv.isFollowing {
		// Scroll to bottom
		maxScroll := len(lv.logs) - 20
		if maxScroll < 0 {
			maxScroll = 0
		}
		lv.scrollOffset = maxScroll
	}
}

func (lv *LogsViewer) fetchLogs() {
	ctx, cancel := context.WithCancel(context.Background())
	lv.cancel = cancel

	// First, get the last 100 lines
	logReader, err := lv.kubeConfig.GetPodLogs(lv.contextName, lv.namespace, lv.podName, lv.containerName, 100, false)
	if err != nil {
		lv.error = err
		return
	}

	// Read initial logs
	scanner := bufio.NewScanner(logReader)
	for scanner.Scan() {
		line := scanner.Text()
		lv.logs = append(lv.logs, line)
	}
	logReader.Close()

	// Start following logs
	followReader, err := lv.kubeConfig.GetPodLogs(lv.contextName, lv.namespace, lv.podName, lv.containerName, 0, true)
	if err != nil {
		lv.error = err
		return
	}

	go func() {
		defer followReader.Close()
		scanner := bufio.NewScanner(followReader)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				line := scanner.Text()
				lv.logs = append(lv.logs, line)
				
				// Limit to last 1000 lines to prevent memory issues
				if len(lv.logs) > 1000 {
					lv.logs = lv.logs[len(lv.logs)-1000:]
				}
				
				// Auto-scroll if following
				if lv.isFollowing {
					maxScroll := len(lv.logs) - 20
					if maxScroll < 0 {
						maxScroll = 0
					}
					lv.scrollOffset = maxScroll
				}
			}
		}
	}()
}

func (lv *LogsViewer) Render(screenWidth, screenHeight int) string {
	if !lv.isOpen {
		return ""
	}

	// Calculate dimensions
	width := screenWidth - 4
	height := screenHeight - 4
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}

	var content strings.Builder

	// Header
	headerStyle := styles.NormalStyle.Bold(true).Foreground(lipgloss.Color("39"))
	title := fmt.Sprintf("📋 Logs: %s/%s", lv.podName, lv.containerName)
	if lv.containerName == "" {
		title = fmt.Sprintf("📋 Logs: %s", lv.podName)
	}
	content.WriteString(headerStyle.Render(title) + "\n")

	// Status line
	statusStyle := styles.NormalStyle.Foreground(lipgloss.Color("245"))
	status := fmt.Sprintf("Namespace: %s", lv.namespace)
	if lv.isFollowing {
		status += " • Following (press 'f' to stop)"
	} else {
		status += " • Paused (press 'f' to follow)"
	}
	content.WriteString(statusStyle.Render(status) + "\n")

	// Controls
	controlsStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
	controls := "↑↓=scroll PgUp/PgDn=page f=follow/pause Esc=close"
	content.WriteString(controlsStyle.Render(controls) + "\n\n")

	// Error handling
	if lv.error != nil {
		errorStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", lv.error)))
	} else if len(lv.logs) == 0 {
		content.WriteString(styles.NormalStyle.Render("Loading logs..."))
	} else {
		// Render logs
		content.WriteString(lv.renderLogs(height - 6)) // Reserve space for header and controls
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

func (lv *LogsViewer) renderLogs(maxLines int) string {
	if len(lv.logs) == 0 {
		return "No logs available"
	}

	var result strings.Builder
	
	// Calculate which logs to show
	startLine := lv.scrollOffset
	endLine := startLine + maxLines
	
	if endLine > len(lv.logs) {
		endLine = len(lv.logs)
	}
	if startLine >= len(lv.logs) {
		startLine = len(lv.logs) - 1
		if startLine < 0 {
			startLine = 0
		}
	}

	// Show logs
	for i := startLine; i < endLine; i++ {
		line := lv.logs[i]
		
		// Color code based on log level
		lineStyle := styles.NormalStyle
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "err") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("196")) // Red
		} else if strings.Contains(lowerLine, "warn") || strings.Contains(lowerLine, "warning") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("226")) // Yellow
		} else if strings.Contains(lowerLine, "info") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("39")) // Blue
		} else if strings.Contains(lowerLine, "debug") {
			lineStyle = lineStyle.Foreground(lipgloss.Color("240")) // Gray
		}
		
		result.WriteString(lineStyle.Render(line))
		if i < endLine-1 {
			result.WriteString("\n")
		}
	}

	// Show scroll indicator
	if len(lv.logs) > maxLines {
		scrollInfo := fmt.Sprintf("\nShowing lines %d-%d of %d", startLine+1, endLine, len(lv.logs))
		scrollStyle := styles.NormalStyle.Foreground(lipgloss.Color("240")).Italic(true)
		result.WriteString(scrollStyle.Render(scrollInfo))
	}

	return result.String()
}