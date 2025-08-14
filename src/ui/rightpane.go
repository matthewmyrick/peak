package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"peek/src/k8s"
	"peek/src/styles"
)

type RightPane struct {
	SelectedItem  string
	Width         int
	Height        int
	SearchMode    bool
	Notifications *NotificationManager
	KubeConfig    *k8s.KubeConfig
	metrics       *k8s.ClusterMetrics
	lastUpdate    time.Time
}

func NewRightPane(width, height int) *RightPane {
	return &RightPane{
		Width:  width,
		Height: height,
	}
}

func (rp *RightPane) SetSelectedItem(item string) {
	rp.SelectedItem = item
}

func (rp *RightPane) SetSearchMode(searchMode bool) {
	rp.SearchMode = searchMode
}

func (rp *RightPane) SetNotifications(nm *NotificationManager) {
	rp.Notifications = nm
}

func (rp *RightPane) SetKubeConfig(kc *k8s.KubeConfig) {
	rp.KubeConfig = kc
}

func (rp *RightPane) Render() string {
	var b strings.Builder
	
	// Check if we have notifications to display
	if rp.Notifications != nil && rp.Notifications.HasNotifications() {
		// Render notifications at the top of the right pane
		notificationContent := rp.renderNotifications()
		b.WriteString(notificationContent)
		b.WriteString("\n\n")
	}
	
	if rp.SelectedItem != "" {
		b.WriteString(styles.HeaderStyle.Render(rp.SelectedItem) + "\n\n")
		
		// Check if this is the Overview section
		if strings.Contains(strings.ToLower(rp.SelectedItem), "overview") {
			overviewContent := rp.renderOverview()
			b.WriteString(overviewContent)
		} else {
			b.WriteString(styles.NormalStyle.Render("Content will appear here"))
		}
	} else {
		b.WriteString(styles.HeaderStyle.Render("Welcome to Peek") + "\n\n")
		b.WriteString(styles.NormalStyle.Render("Select an item from the left panel to view details\n\n"))
		b.WriteString(styles.NormalStyle.Render("Available commands are shown in the footer below."))
	}
	
	return b.String()
}

func (rp *RightPane) renderOverview() string {
	// Update metrics if needed (every 30 seconds)
	if rp.KubeConfig != nil && (rp.metrics == nil || time.Since(rp.lastUpdate) > 30*time.Second) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		metrics, err := rp.KubeConfig.GetClusterMetrics(ctx)
		if err != nil {
			return styles.NormalStyle.Render(fmt.Sprintf("Failed to load cluster metrics: %v", err))
		}
		
		rp.metrics = metrics
		rp.lastUpdate = time.Now()
	}
	
	if rp.metrics == nil {
		return styles.NormalStyle.Render("Loading cluster metrics...")
	}
	
	var b strings.Builder
	
	// Node metrics
	b.WriteString(styles.HeaderStyle.Render("Node Metrics") + "\n")
	nodeMetrics := rp.renderNodeMetrics()
	b.WriteString(nodeMetrics + "\n\n")
	
	// Pod metrics
	b.WriteString(styles.HeaderStyle.Render("Pod Metrics") + "\n")
	podMetrics := rp.renderPodMetrics()
	b.WriteString(podMetrics + "\n\n")
	
	// Events
	b.WriteString(styles.HeaderStyle.Render("Recent Events") + "\n")
	eventsTable := rp.renderEventsTable()
	b.WriteString(eventsTable)
	
	return b.String()
}

func (rp *RightPane) renderNodeMetrics() string {
	if rp.metrics == nil {
		return styles.NormalStyle.Render("No data available")
	}
	
	var b strings.Builder
	
	// Node status
	totalNodes := rp.metrics.Nodes.Total
	readyNodes := rp.metrics.Nodes.Ready
	notReadyNodes := rp.metrics.Nodes.NotReady
	
	statusLine := fmt.Sprintf("Total: %d | Ready: %d | Not Ready: %d", 
		totalNodes, readyNodes, notReadyNodes)
	
	if notReadyNodes > 0 {
		statusStyle := styles.NormalStyle.Foreground(lipgloss.Color("214"))
		b.WriteString(statusStyle.Render(statusLine) + "\n")
	} else {
		b.WriteString(styles.NormalStyle.Render(statusLine) + "\n")
	}
	
	// CPU metrics
	cpuTotal := rp.metrics.Nodes.CPUCapacity
	cpuUsed := cpuTotal - rp.metrics.Nodes.CPUAllocated
	cpuPercent := k8s.FormatPercentage(cpuUsed, cpuTotal)
	
	cpuLine := fmt.Sprintf("CPU: %s used of %s (%s)", 
		k8s.FormatMilliCPU(cpuUsed), 
		k8s.FormatMilliCPU(cpuTotal), 
		cpuPercent)
	b.WriteString(styles.NormalStyle.Render(cpuLine) + "\n")
	
	// Memory metrics
	memTotal := rp.metrics.Nodes.MemCapacity
	memUsed := memTotal - rp.metrics.Nodes.MemAllocated
	memPercent := k8s.FormatPercentage(memUsed, memTotal)
	
	memLine := fmt.Sprintf("Memory: %s used of %s (%s)", 
		k8s.FormatBytes(memUsed), 
		k8s.FormatBytes(memTotal), 
		memPercent)
	b.WriteString(styles.NormalStyle.Render(memLine))
	
	return b.String()
}

func (rp *RightPane) renderPodMetrics() string {
	if rp.metrics == nil {
		return styles.NormalStyle.Render("No data available")
	}
	
	var b strings.Builder
	
	pods := rp.metrics.Pods
	
	// Pod status summary
	statusLine := fmt.Sprintf("Total: %d | Running: %d | Pending: %d", 
		pods.Total, pods.Running, pods.Pending)
	b.WriteString(styles.NormalStyle.Render(statusLine) + "\n")
	
	// Show failed/unknown pods if any
	if pods.Failed > 0 || pods.Unknown > 0 {
		problemLine := fmt.Sprintf("Failed: %d | Unknown: %d", pods.Failed, pods.Unknown)
		problemStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		b.WriteString(problemStyle.Render(problemLine))
	} else {
		b.WriteString(styles.NormalStyle.Render("All pods healthy"))
	}
	
	return b.String()
}

func (rp *RightPane) renderEventsTable() string {
	if rp.metrics == nil || len(rp.metrics.Events) == 0 {
		return styles.NormalStyle.Render("No recent warning or error events")
	}
	
	var b strings.Builder
	
	// Table header
	headerStyle := styles.NormalStyle.Bold(true).Underline(true)
	header := fmt.Sprintf("%-8s %-12s %-15s %-30s %s", 
		"TYPE", "REASON", "OBJECT", "MESSAGE", "AGE")
	b.WriteString(headerStyle.Render(header) + "\n")
	
	// Event rows
	for i, event := range rp.metrics.Events {
		if i >= 10 { // Limit to 10 events
			break
		}
		
		// Truncate fields to fit
		eventType := k8s.TruncateString(event.Type, 8)
		reason := k8s.TruncateString(event.Reason, 12)
		object := k8s.TruncateString(event.Object, 15)
		message := k8s.TruncateString(event.Message, 30)
		age := k8s.FormatTimeAgo(event.LastSeen)
		
		row := fmt.Sprintf("%-8s %-12s %-15s %-30s %s", 
			eventType, reason, object, message, age)
		
		// Color based on event type
		color := k8s.GetEventColor(event.Type)
		rowStyle := styles.NormalStyle.Foreground(lipgloss.Color(color))
		
		b.WriteString(rowStyle.Render(row))
		if i < len(rp.metrics.Events)-1 && i < 9 {
			b.WriteString("\n")
		}
	}
	
	return b.String()
}

func (rp *RightPane) renderNotifications() string {
	if rp.Notifications == nil {
		return ""
	}
	
	rp.Notifications.CleanExpired()
	notifications := rp.Notifications.Notifications
	
	if len(notifications) == 0 {
		return ""
	}
	
	var b strings.Builder
	maxVisible := 3
	visibleCount := len(notifications)
	if visibleCount > maxVisible {
		visibleCount = maxVisible
	}
	
	for i := 0; i < visibleCount; i++ {
		notif := notifications[i]
		
		// Choose icon and color based on type
		var icon, color string
		switch notif.Type {
		case NotificationError:
			icon = "✗"
			color = "196"
		case NotificationWarning:
			icon = "⚠"
			color = "214"
		case NotificationSuccess:
			icon = "✓"
			color = "46"
		default:
			icon = "ℹ"
			color = "39"
		}
		
		notifStyle := styles.NormalStyle.
			Foreground(lipgloss.Color(color)).
			Bold(true)
		
		timeAgo := rp.formatTimeAgo(notif.Timestamp)
		
		line := fmt.Sprintf("%s %s - %s (%s)", icon, notif.Title, notif.Message, timeAgo)
		b.WriteString(notifStyle.Render(line))
		
		if i < visibleCount-1 {
			b.WriteString("\n")
		}
	}
	
	return b.String()
}

func (rp *RightPane) formatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Second {
		return "now"
	} else if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	
	return fmt.Sprintf("%dh", int(duration.Hours()))
}