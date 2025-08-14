package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"peek/src/k8s"
	"peek/src/styles"
)

type NodesTable struct {
	nodes       []k8s.NodeInfo
	lastUpdate  time.Time
	kubeConfig  *k8s.KubeConfig
	contextName string
	isLoading   bool
	error       error
}

func NewNodesTable(kubeConfig *k8s.KubeConfig, contextName string) *NodesTable {
	return &NodesTable{
		kubeConfig:  kubeConfig,
		contextName: contextName,
		isLoading:   true,
	}
}

func (nt *NodesTable) Update() error {
	if nt.kubeConfig == nil {
		return fmt.Errorf("kubeconfig not available")
	}

	nt.isLoading = true
	nt.error = nil

	nodes, err := nt.kubeConfig.GetNodes(nt.contextName)
	if err != nil {
		nt.error = err
		nt.isLoading = false
		return err
	}

	nt.nodes = nodes
	nt.lastUpdate = time.Now()
	nt.isLoading = false
	return nil
}

func (nt *NodesTable) ShouldUpdate() bool {
	// Update every 30 seconds or if never updated
	return time.Since(nt.lastUpdate) > 30*time.Second
}

func (nt *NodesTable) Render() string {
	var b strings.Builder

	if nt.isLoading && len(nt.nodes) == 0 {
		b.WriteString(styles.NormalStyle.Render("Loading nodes..."))
		return b.String()
	}

	if nt.error != nil {
		errorStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error loading nodes: %v", nt.error)))
		return b.String()
	}

	if len(nt.nodes) == 0 {
		b.WriteString(styles.NormalStyle.Render("No nodes found"))
		return b.String()
	}

	// Table header
	headerStyle := styles.NormalStyle.Bold(true).Underline(true)
	header := fmt.Sprintf("%-20s %-10s %-15s %-8s %-12s %-10s %-8s %s",
		"NAME", "STATUS", "ROLES", "AGE", "VERSION", "OS", "ARCH", "MEMORY")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Table rows
	for i, node := range nt.nodes {
		// Truncate and format fields
		name := truncateString(node.Name, 20)
		status := truncateString(node.Status, 10)
		roles := truncateString(strings.Join(node.Roles, ","), 15)
		age := truncateString(node.Age, 8)
		version := truncateString(node.Version, 12)
		os := truncateString(node.OS, 10)
		arch := truncateString(node.Architecture, 8)
		memory := truncateString(node.MemCapacity, 12)

		row := fmt.Sprintf("%-20s %-10s %-15s %-8s %-12s %-10s %-8s %s",
			name, status, roles, age, version, os, arch, memory)

		// Color based on status
		var rowStyle lipgloss.Style
		if node.Ready {
			rowStyle = styles.NormalStyle.Foreground(lipgloss.Color("46")) // Green
		} else {
			rowStyle = styles.NormalStyle.Foreground(lipgloss.Color("196")) // Red
		}

		b.WriteString(rowStyle.Render(row))
		if i < len(nt.nodes)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
