package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"peek/src/k8s"
	"peek/src/styles"
)

type ApplicationsTable struct {
	applications []k8s.ApplicationInfo
	lastUpdate   time.Time
	kubeConfig   *k8s.KubeConfig
	contextName  string
	namespace    string
	isLoading    bool
	error        error
}

func NewApplicationsTable(kubeConfig *k8s.KubeConfig, contextName, namespace string) *ApplicationsTable {
	return &ApplicationsTable{
		kubeConfig:  kubeConfig,
		contextName: contextName,
		namespace:   namespace,
		isLoading:   true,
	}
}

func (at *ApplicationsTable) SetNamespace(namespace string) {
	at.namespace = namespace
	// Force refresh on next update check
	at.lastUpdate = time.Time{}
	// Clear applications to trigger loading state
	at.applications = []k8s.ApplicationInfo{}
}

func (at *ApplicationsTable) Update() error {
	if at.kubeConfig == nil {
		return fmt.Errorf("kubeconfig not available")
	}

	// Only set loading to true if this is the first load (no existing applications)
	if len(at.applications) == 0 {
		at.isLoading = true
	}
	at.error = nil

	applications, err := at.kubeConfig.GetApplications(at.contextName, at.namespace)
	if err != nil {
		at.error = err
		at.isLoading = false
		return err
	}

	// Sort applications by type first, then by name
	sort.Slice(applications, func(i, j int) bool {
		if applications[i].Type != applications[j].Type {
			return applications[i].Type < applications[j].Type
		}
		return applications[i].Name < applications[j].Name
	})

	at.applications = applications
	at.lastUpdate = time.Now()
	at.isLoading = false
	return nil
}

func (at *ApplicationsTable) ShouldUpdate() bool {
	// Update every 30 seconds for applications
	return time.Since(at.lastUpdate) > 30*time.Second
}

func (at *ApplicationsTable) Render() string {
	var b strings.Builder

	// Only show loading screen if we have no applications AND it's the initial load
	if at.isLoading && len(at.applications) == 0 && at.lastUpdate.IsZero() {
		b.WriteString(styles.NormalStyle.Render("Loading applications..."))
		return b.String()
	}

	if at.error != nil {
		errorStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error loading applications: %v", at.error)))
		return b.String()
	}

	// Namespace info - show updating status more subtly
	namespaceStyle := styles.NormalStyle.Foreground(lipgloss.Color("245"))
	namespaceText := fmt.Sprintf("Showing applications in namespace: %s", at.namespace)
	if at.namespace == "" {
		namespaceText = "Showing applications across all namespaces"
	}
	if at.isLoading && len(at.applications) > 0 {
		// Show a subtle updating indicator only if we already have data
		namespaceText += " ●"
	}
	b.WriteString(namespaceStyle.Render(namespaceText) + "\n")

	// Controls info
	controlsStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
	b.WriteString(controlsStyle.Render("Auto-refresh every 30s • Use Ctrl+N to change namespace") + "\n\n")

	if len(at.applications) == 0 {
		b.WriteString(styles.NormalStyle.Render("No applications found in the selected namespace(s)"))
		return b.String()
	}

	// Summary statistics
	b.WriteString(at.renderSummary() + "\n\n")

	// Applications table
	b.WriteString(at.renderApplicationsTable())

	return b.String()
}

func (at *ApplicationsTable) renderSummary() string {
	var b strings.Builder

	// Count by type and status
	typeCounts := make(map[string]int)
	statusCounts := make(map[string]int)

	for _, app := range at.applications {
		typeCounts[app.Type]++
		statusCounts[app.Status]++
	}

	b.WriteString(styles.HeaderStyle.Render("📊 Applications Summary") + "\n")

	// Type summary
	b.WriteString(styles.NormalStyle.Bold(true).Render("By Type:") + " ")
	var typeParts []string
	for appType, count := range typeCounts {
		color := getTypeColor(appType)
		typeStyle := styles.NormalStyle.Foreground(lipgloss.Color(color))
		typeParts = append(typeParts, typeStyle.Render(fmt.Sprintf("%s: %d", appType, count)))
	}
	b.WriteString(strings.Join(typeParts, " | ") + "\n")

	// Status summary
	b.WriteString(styles.NormalStyle.Bold(true).Render("By Status:") + " ")
	var statusParts []string
	for status, count := range statusCounts {
		color := getStatusColor(status)
		statusStyle := styles.NormalStyle.Foreground(lipgloss.Color(color))
		statusParts = append(statusParts, statusStyle.Render(fmt.Sprintf("%s: %d", status, count)))
	}
	b.WriteString(strings.Join(statusParts, " | "))

	return b.String()
}

func (at *ApplicationsTable) renderApplicationsTable() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("🚀 Applications") + "\n")

	// Table header
	headerStyle := styles.NormalStyle.Bold(true).Underline(true)
	header := fmt.Sprintf("%-12s %-20s %-15s %-10s %-8s %-12s %s",
		"TYPE", "NAME", "NAMESPACE", "STATUS", "READY", "REPLICAS", "AGE")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Table rows
	for i, app := range at.applications {
		if i >= 50 { // Limit to 50 applications for performance
			moreStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
			b.WriteString(moreStyle.Render(fmt.Sprintf("\n... and %d more applications", len(at.applications)-50)))
			break
		}

		// Truncate and format fields
		appType := truncateString(app.Type, 12)
		name := truncateString(app.Name, 20)
		namespace := truncateString(app.Namespace, 15)
		status := truncateString(app.Status, 10)
		ready := fmt.Sprintf("%d/%d", app.ReadyReplicas, app.Replicas)
		if len(ready) > 8 {
			ready = truncateString(ready, 8)
		}
		replicas := fmt.Sprintf("%d", app.Replicas)
		if len(replicas) > 12 {
			replicas = truncateString(replicas, 12)
		}
		age := formatAppAge(app.CreationTime)

		row := fmt.Sprintf("%-12s %-20s %-15s %-10s %-8s %-12s %s",
			appType, name, namespace, status, ready, replicas, age)

		// Color based on status
		statusColor := getStatusColor(app.Status)
		rowStyle := styles.NormalStyle.Foreground(lipgloss.Color(statusColor))

		b.WriteString(rowStyle.Render(row))
		if i < len(at.applications)-1 && i < 49 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// getTypeColor returns the color for different application types
func getTypeColor(appType string) string {
	switch strings.ToLower(appType) {
	case "deployment":
		return "39" // Blue
	case "daemonset":
		return "214" // Orange
	case "statefulset":
		return "129" // Purple
	case "replicaset":
		return "51" // Cyan
	case "job":
		return "226" // Yellow
	case "cronjob":
		return "208" // Orange-Red
	default:
		return "252" // White/Default
	}
}

// getStatusColor returns the color for different application statuses
func getStatusColor(status string) string {
	switch strings.ToLower(status) {
	case "running", "available", "ready", "complete":
		return "46" // Green
	case "pending", "progressing":
		return "226" // Yellow
	case "failed", "error", "crashloopbackoff":
		return "196" // Red
	case "unknown", "terminating":
		return "240" // Gray
	default:
		return "252" // White/Default
	}
}

// formatAppAge formats the age of an application
func formatAppAge(creationTime time.Time) string {
	if creationTime.IsZero() {
		return "unknown"
	}

	duration := time.Since(creationTime)

	if duration < time.Minute {
		seconds := int(duration.Seconds())
		return fmt.Sprintf("%ds", seconds)
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh", hours)
	} else {
		days := int(duration.Hours()) / 24
		return fmt.Sprintf("%dd", days)
	}
}