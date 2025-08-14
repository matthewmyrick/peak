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
	nodesTable    *NodesTable
	eventsTable   *EventsTable
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
	// Initialize nodes table and events table with current context if available
	if kc != nil {
		rp.nodesTable = NewNodesTable(kc, kc.CurrentContext)
		rp.eventsTable = NewEventsTable(kc, kc.CurrentContext)
	}
}

func (rp *RightPane) Render() string {
	var b strings.Builder

	if rp.SelectedItem != "" {
		b.WriteString(styles.HeaderStyle.Render(rp.SelectedItem) + "\n\n")

		// Check if this is the Overview section
		if strings.Contains(strings.ToLower(rp.SelectedItem), "overview") {
			overviewContent := rp.renderOverview()
			b.WriteString(overviewContent)
		} else if strings.Contains(strings.ToLower(rp.SelectedItem), "nodes") {
			// Handle nodes view
			nodesContent := rp.renderNodes()
			b.WriteString(nodesContent)
		} else if strings.Contains(strings.ToLower(rp.SelectedItem), "events") {
			// Handle events view
			eventsContent := rp.renderEvents()
			b.WriteString(eventsContent)
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

	// Cluster Overview Header
	b.WriteString(styles.HeaderStyle.Render("📊 Cluster Overview") + "\n\n")

	// Node metrics with graphs
	nodeMetrics := rp.renderNodeMetrics()
	b.WriteString(nodeMetrics + "\n\n")

	// Pod metrics with graphs
	podMetrics := rp.renderPodMetrics()
	b.WriteString(podMetrics + "\n\n")

	// Events section
	b.WriteString(styles.HeaderStyle.Render("⚡ Recent Events") + "\n")
	eventsTable := rp.renderEventsTable()
	b.WriteString(eventsTable)

	return b.String()
}

func (rp *RightPane) renderNodeMetrics() string {
	if rp.metrics == nil {
		return styles.NormalStyle.Render("No data available")
	}

	var b strings.Builder

	// Node Status Table
	totalNodes := rp.metrics.Nodes.Total
	readyNodes := rp.metrics.Nodes.Ready
	notReadyNodes := rp.metrics.Nodes.NotReady

	b.WriteString(styles.HeaderStyle.Render("🖥️ Node Status") + "\n")
	
	// Table header
	headerStyle := styles.NormalStyle.Bold(true).Underline(true)
	header := fmt.Sprintf("%-12s %-8s %-12s", "STATUS", "COUNT", "PERCENTAGE")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Ready nodes
	readyPercent := 0.0
	if totalNodes > 0 {
		readyPercent = float64(readyNodes) / float64(totalNodes) * 100
	}
	readyStyle := styles.NormalStyle.Foreground(lipgloss.Color("46")) // Green
	readyRow := fmt.Sprintf("%-12s %-8d %-12.1f%%", "✅ Ready", readyNodes, readyPercent)
	b.WriteString(readyStyle.Render(readyRow) + "\n")

	// Not Ready nodes
	if notReadyNodes > 0 {
		notReadyPercent := float64(notReadyNodes) / float64(totalNodes) * 100
		notReadyStyle := styles.NormalStyle.Foreground(lipgloss.Color("196")) // Red
		notReadyRow := fmt.Sprintf("%-12s %-8d %-12.1f%%", "❌ Not Ready", notReadyNodes, notReadyPercent)
		b.WriteString(notReadyStyle.Render(notReadyRow) + "\n")
	}

	// Total row
	totalStyle := styles.NormalStyle.Bold(true)
	totalRow := fmt.Sprintf("%-12s %-8d %-12s", "📊 Total", totalNodes, "100.0%")
	b.WriteString(totalStyle.Render(totalRow) + "\n\n")

	// Resource Usage Table
	b.WriteString(styles.HeaderStyle.Render("💾 Resource Usage") + "\n")
	
	// Table header
	resourceHeader := fmt.Sprintf("%-12s %-15s %-15s %-12s", "RESOURCE", "ALLOCATED", "CAPACITY", "USAGE %")
	b.WriteString(headerStyle.Render(resourceHeader) + "\n")

	// CPU usage
	cpuTotal := rp.metrics.Nodes.CPUCapacity
	cpuAllocated := rp.metrics.Nodes.CPUAllocated
	cpuPercent := 0.0
	if cpuTotal > 0 {
		cpuPercent = float64(cpuAllocated) / float64(cpuTotal) * 100
	}
	
	cpuColor := "46" // Green
	if cpuPercent > 80 {
		cpuColor = "196" // Red
	} else if cpuPercent > 60 {
		cpuColor = "226" // Yellow
	}
	cpuStyle := styles.NormalStyle.Foreground(lipgloss.Color(cpuColor))
	cpuRow := fmt.Sprintf("%-12s %-15s %-15s %-12.1f%%", "🔧 CPU", 
		fmt.Sprintf("%.2f cores", cpuAllocated), 
		fmt.Sprintf("%.2f cores", cpuTotal), 
		cpuPercent)
	b.WriteString(cpuStyle.Render(cpuRow) + "\n")

	// Memory usage
	memTotal := rp.metrics.Nodes.MemCapacity
	memAllocated := rp.metrics.Nodes.MemAllocated
	memPercent := 0.0
	if memTotal > 0 {
		memPercent = float64(memAllocated) / float64(memTotal) * 100
	}
	
	memColor := "46" // Green
	if memPercent > 80 {
		memColor = "196" // Red
	} else if memPercent > 60 {
		memColor = "226" // Yellow
	}
	memStyle := styles.NormalStyle.Foreground(lipgloss.Color(memColor))
	memRow := fmt.Sprintf("%-12s %-15s %-15s %-12.1f%%", "🧠 Memory", 
		fmt.Sprintf("%.1f GB", memAllocated/(1024*1024*1024)), 
		fmt.Sprintf("%.1f GB", memTotal/(1024*1024*1024)), 
		memPercent)
	b.WriteString(memStyle.Render(memRow))

	return b.String()
}

func (rp *RightPane) renderPodMetrics() string {
	if rp.metrics == nil {
		return styles.NormalStyle.Render("No data available")
	}

	var b strings.Builder
	pods := rp.metrics.Pods

	// Calculate totals
	totalPods := pods.Running + pods.Pending + pods.Failed + pods.Unknown

	b.WriteString(styles.HeaderStyle.Render("🚀 Pod Status") + "\n")
	
	// Table header
	headerStyle := styles.NormalStyle.Bold(true).Underline(true)
	header := fmt.Sprintf("%-12s %-8s %-12s", "STATUS", "COUNT", "PERCENTAGE")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Running pods
	if pods.Running > 0 {
		runningPercent := 0.0
		if totalPods > 0 {
			runningPercent = float64(pods.Running) / float64(totalPods) * 100
		}
		runningStyle := styles.NormalStyle.Foreground(lipgloss.Color("46")) // Green
		runningRow := fmt.Sprintf("%-12s %-8d %-12.1f%%", "✅ Running", pods.Running, runningPercent)
		b.WriteString(runningStyle.Render(runningRow) + "\n")
	}

	// Pending pods
	if pods.Pending > 0 {
		pendingPercent := float64(pods.Pending) / float64(totalPods) * 100
		pendingStyle := styles.NormalStyle.Foreground(lipgloss.Color("226")) // Yellow
		pendingRow := fmt.Sprintf("%-12s %-8d %-12.1f%%", "⏳ Pending", pods.Pending, pendingPercent)
		b.WriteString(pendingStyle.Render(pendingRow) + "\n")
	}

	// Failed pods
	if pods.Failed > 0 {
		failedPercent := float64(pods.Failed) / float64(totalPods) * 100
		failedStyle := styles.NormalStyle.Foreground(lipgloss.Color("196")) // Red
		failedRow := fmt.Sprintf("%-12s %-8d %-12.1f%%", "❌ Failed", pods.Failed, failedPercent)
		b.WriteString(failedStyle.Render(failedRow) + "\n")
	}

	// Unknown pods
	if pods.Unknown > 0 {
		unknownPercent := float64(pods.Unknown) / float64(totalPods) * 100
		unknownStyle := styles.NormalStyle.Foreground(lipgloss.Color("240")) // Gray
		unknownRow := fmt.Sprintf("%-12s %-8d %-12.1f%%", "❓ Unknown", pods.Unknown, unknownPercent)
		b.WriteString(unknownStyle.Render(unknownRow) + "\n")
	}

	// Total row
	totalStyle := styles.NormalStyle.Bold(true)
	totalRow := fmt.Sprintf("%-12s %-8d %-12s", "📊 Total", totalPods, "100.0%")
	b.WriteString(totalStyle.Render(totalRow) + "\n\n")

	// Health summary
	if pods.Failed > 0 || pods.Unknown > 0 {
		problemStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		healthRow := fmt.Sprintf("⚠️ Health Status: %d failed, %d unknown pods detected", pods.Failed, pods.Unknown)
		b.WriteString(problemStyle.Render(healthRow))
	} else if totalPods > 0 {
		healthStyle := styles.NormalStyle.Foreground(lipgloss.Color("46"))
		b.WriteString(healthStyle.Render("✅ Health Status: All pods are healthy"))
	} else {
		healthStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
		b.WriteString(healthStyle.Render("📊 Health Status: No pods found"))
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
		age := k8s.FormatTimeAgo(event.LastTimestamp)

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

func (rp *RightPane) renderNodes() string {
	if rp.nodesTable == nil {
		if rp.KubeConfig != nil {
			rp.nodesTable = NewNodesTable(rp.KubeConfig, rp.KubeConfig.CurrentContext)
		} else {
			return styles.NormalStyle.Render("Kubernetes configuration not available")
		}
	}

	// Update nodes if needed
	if rp.nodesTable.ShouldUpdate() {
		go func() {
			rp.nodesTable.Update()
		}()
	}

	return rp.nodesTable.Render()
}

func (rp *RightPane) renderEvents() string {
	if rp.eventsTable == nil {
		if rp.KubeConfig != nil {
			rp.eventsTable = NewEventsTable(rp.KubeConfig, rp.KubeConfig.CurrentContext)
			// Trigger initial load immediately for first time
			go func() {
				rp.eventsTable.Update()
			}()
		} else {
			return styles.NormalStyle.Render("Kubernetes configuration not available")
		}
	}

	// Update events if needed (background updates)
	if rp.eventsTable.ShouldUpdate() {
		go func() {
			rp.eventsTable.Update()
		}()
	}

	return rp.eventsTable.Render()
}

func (rp *RightPane) UpdateNodes() {
	if rp.nodesTable != nil {
		go func() {
			rp.nodesTable.Update()
		}()
	}
}

func (rp *RightPane) UpdateEvents() {
	if rp.eventsTable != nil {
		go func() {
			rp.eventsTable.Update()
		}()
	}
}

func (rp *RightPane) GetEventsTable() *EventsTable {
	return rp.eventsTable
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
