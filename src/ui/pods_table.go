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

type PodsTable struct {
	pods            []k8s.PodInfo
	filteredPods    []k8s.PodInfo
	lastUpdate      time.Time
	kubeConfig      *k8s.KubeConfig
	contextName     string
	namespace       string
	isLoading       bool
	error           error
	cursor          int
	searchMode      bool
	searchQuery     string
	selectedPod     *k8s.PodInfo
}

func NewPodsTable(kubeConfig *k8s.KubeConfig, contextName, namespace string) *PodsTable {
	return &PodsTable{
		kubeConfig:  kubeConfig,
		contextName: contextName,
		namespace:   namespace,
		isLoading:   true,
		cursor:      0,
		searchMode:  false,
		searchQuery: "",
	}
}

func (pt *PodsTable) SetNamespace(namespace string) {
	pt.namespace = namespace
	// Force refresh on next update check
	pt.lastUpdate = time.Time{}
	// Clear pods to trigger loading state
	pt.pods = []k8s.PodInfo{}
	pt.filteredPods = []k8s.PodInfo{}
	pt.cursor = 0
}

func (pt *PodsTable) Update() error {
	if pt.kubeConfig == nil {
		return fmt.Errorf("kubeconfig not available")
	}

	// Only set loading to true if this is the first load (no existing pods)
	if len(pt.pods) == 0 {
		pt.isLoading = true
	}
	pt.error = nil

	pods, err := pt.kubeConfig.GetPods(pt.contextName, pt.namespace)
	if err != nil {
		pt.error = err
		pt.isLoading = false
		return err
	}

	// Sort pods by name
	sort.Slice(pods, func(i, j int) bool {
		return pods[i].Name < pods[j].Name
	})

	pt.pods = pods
	pt.filterPods()
	pt.lastUpdate = time.Now()
	pt.isLoading = false
	return nil
}

func (pt *PodsTable) ShouldUpdate() bool {
	// Update every 15 seconds for pods (faster than other resources)
	return time.Since(pt.lastUpdate) > 15*time.Second
}

func (pt *PodsTable) ToggleSearchMode() {
	pt.searchMode = !pt.searchMode
	if !pt.searchMode {
		pt.searchQuery = ""
		pt.filterPods()
		pt.cursor = 0
	}
}

func (pt *PodsTable) UpdateSearch(query string) {
	pt.searchQuery = query
	pt.filterPods()
	pt.cursor = 0
}

func (pt *PodsTable) filterPods() {
	if pt.searchQuery == "" {
		pt.filteredPods = pt.pods
		return
	}

	var filtered []k8s.PodInfo
	query := strings.ToLower(pt.searchQuery)
	
	for _, pod := range pt.pods {
		// Search in name, namespace, status, node
		searchText := strings.ToLower(fmt.Sprintf("%s %s %s %s", 
			pod.Name, pod.Namespace, pod.Status, pod.Node))
		if strings.Contains(searchText, query) {
			filtered = append(filtered, pod)
		}
	}
	
	pt.filteredPods = filtered
}

func (pt *PodsTable) MoveUp() {
	if pt.cursor > 0 {
		pt.cursor--
	}
}

func (pt *PodsTable) MoveDown() {
	if pt.cursor < len(pt.filteredPods)-1 {
		pt.cursor++
	}
}

func (pt *PodsTable) GetSelectedPod() *k8s.PodInfo {
	if pt.cursor < len(pt.filteredPods) {
		return &pt.filteredPods[pt.cursor]
	}
	return nil
}

func (pt *PodsTable) Render() string {
	var b strings.Builder

	// Only show loading screen if we have no pods AND it's the initial load
	if pt.isLoading && len(pt.pods) == 0 && pt.lastUpdate.IsZero() {
		b.WriteString(styles.NormalStyle.Render("Loading pods..."))
		return b.String()
	}

	if pt.error != nil {
		errorStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error loading pods: %v", pt.error)))
		return b.String()
	}

	// Header info
	namespaceStyle := styles.NormalStyle.Foreground(lipgloss.Color("245"))
	namespaceText := fmt.Sprintf("Showing pods in namespace: %s", pt.namespace)
	if pt.namespace == "" {
		namespaceText = "Showing pods across all namespaces"
	}
	if pt.isLoading && len(pt.pods) > 0 {
		namespaceText += " ●"
	}
	b.WriteString(namespaceStyle.Render(namespaceText) + "\n")

	// Search mode indicator
	if pt.searchMode {
		searchStyle := styles.NormalStyle.Foreground(lipgloss.Color("39")).Bold(true)
		searchText := fmt.Sprintf("🔍 Search: %s", pt.searchQuery)
		if pt.searchQuery == "" {
			searchText += "█" // cursor
		}
		b.WriteString(searchStyle.Render(searchText) + "\n")
	}

	// Controls info
	controlsStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
	controls := "Auto-refresh every 15s • / to search • l=logs e=exec d=delete r=restart y=yaml"
	b.WriteString(controlsStyle.Render(controls) + "\n\n")

	if len(pt.filteredPods) == 0 {
		if pt.searchMode && pt.searchQuery != "" {
			b.WriteString(styles.NormalStyle.Render("No pods match your search"))
		} else {
			b.WriteString(styles.NormalStyle.Render("No pods found in the selected namespace(s)"))
		}
		return b.String()
	}

	// Summary
	b.WriteString(pt.renderSummary() + "\n\n")

	// Pods table
	b.WriteString(pt.renderPodsTable())

	return b.String()
}

func (pt *PodsTable) renderSummary() string {
	var b strings.Builder

	// Count by status
	statusCounts := make(map[string]int)
	for _, pod := range pt.filteredPods {
		statusCounts[pod.Status]++
	}

	b.WriteString(styles.HeaderStyle.Render("🚀 Pods Summary") + "\n")
	b.WriteString(styles.NormalStyle.Bold(true).Render(fmt.Sprintf("Total: %d pods", len(pt.filteredPods))) + " | ")

	var statusParts []string
	for status, count := range statusCounts {
		color := getPodStatusColor(status)
		statusStyle := styles.NormalStyle.Foreground(lipgloss.Color(color))
		statusParts = append(statusParts, statusStyle.Render(fmt.Sprintf("%s: %d", status, count)))
	}
	b.WriteString(strings.Join(statusParts, " | "))

	return b.String()
}

func (pt *PodsTable) renderPodsTable() string {
	var b strings.Builder

	b.WriteString(styles.HeaderStyle.Render("📋 Pods") + "\n")

	// Table header
	headerStyle := styles.NormalStyle.Bold(true).Underline(true)
	header := fmt.Sprintf("%-20s %-15s %-12s %-8s %-8s %-15s %s",
		"NAME", "NAMESPACE", "STATUS", "READY", "RESTARTS", "NODE", "AGE")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Determine which pods to show (with scrolling)
	startIndex := 0
	endIndex := len(pt.filteredPods)
	maxVisible := 20 // Show max 20 pods at once

	if len(pt.filteredPods) > maxVisible {
		// Calculate scroll window
		if pt.cursor >= maxVisible/2 {
			startIndex = pt.cursor - maxVisible/2
		}
		endIndex = startIndex + maxVisible
		if endIndex > len(pt.filteredPods) {
			endIndex = len(pt.filteredPods)
			startIndex = endIndex - maxVisible
			if startIndex < 0 {
				startIndex = 0
			}
		}
	}

	// Show scroll indicator if needed
	if len(pt.filteredPods) > maxVisible {
		scrollStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d pods (use ↑↓ to navigate)",
			startIndex+1, endIndex, len(pt.filteredPods))
		b.WriteString(scrollStyle.Render(scrollInfo) + "\n")
	}

	// Table rows
	for i := startIndex; i < endIndex; i++ {
		pod := pt.filteredPods[i]

		// Truncate and format fields
		name := truncateString(pod.Name, 20)
		namespace := truncateString(pod.Namespace, 15)
		status := truncateString(pod.Status, 12)
		ready := truncateString(pod.Ready, 8)
		restarts := fmt.Sprintf("%d", pod.Restarts)
		if len(restarts) > 8 {
			restarts = truncateString(restarts, 8)
		}
		node := truncateString(pod.Node, 15)
		age := formatPodAge(pod.Age)

		row := fmt.Sprintf("%-20s %-15s %-12s %-8s %-8s %-15s %s",
			name, namespace, status, ready, restarts, node, age)

		// Color based on status and highlight selection
		statusColor := getPodStatusColor(pod.Status)
		rowStyle := styles.NormalStyle.Foreground(lipgloss.Color(statusColor))

		// Highlight selected pod
		if i == pt.cursor {
			rowStyle = rowStyle.Background(lipgloss.Color("237")).Bold(true)
		}

		b.WriteString(rowStyle.Render(row))
		if i < endIndex-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func getPodStatusColor(status string) string {
	lowerStatus := strings.ToLower(status)
	
	// Only red for actual errors
	if strings.Contains(lowerStatus, "failed") || 
		strings.Contains(lowerStatus, "error") ||
		strings.Contains(lowerStatus, "crashloopbackoff") ||
		strings.Contains(lowerStatus, "imagepullbackoff") ||
		strings.Contains(lowerStatus, "errimagepull") ||
		strings.Contains(lowerStatus, "invalidimgname") {
		return "196" // Red
	}
	
	// Yellow for warnings/pending states
	if strings.Contains(lowerStatus, "pending") ||
		strings.Contains(lowerStatus, "containercreating") ||
		strings.Contains(lowerStatus, "podinitialized") ||
		strings.Contains(lowerStatus, "imagepullbackoff") {
		return "226" // Yellow
	}
	
	// Everything else is white (running, succeeded, completed, etc.)
	return "252" // White/Default
}

func formatPodAge(age time.Duration) string {
	if age < time.Minute {
		seconds := int(age.Seconds())
		return fmt.Sprintf("%ds", seconds)
	} else if age < time.Hour {
		minutes := int(age.Minutes())
		return fmt.Sprintf("%dm", minutes)
	} else if age < 24*time.Hour {
		hours := int(age.Hours())
		return fmt.Sprintf("%dh", hours)
	} else {
		days := int(age.Hours()) / 24
		return fmt.Sprintf("%dd", days)
	}
}

func (pt *PodsTable) IsSearchMode() bool {
	return pt.searchMode
}

func (pt *PodsTable) GetSearchQuery() string {
	return pt.searchQuery
}