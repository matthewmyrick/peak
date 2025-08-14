package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"peek/src/k8s"
	"peek/src/styles"
)

type EventsTable struct {
	events       []k8s.EventInfo
	lastUpdate   time.Time
	kubeConfig   *k8s.KubeConfig
	contextName  string
	timeframeMin int
	isLoading    bool
	error        error
}

func NewEventsTable(kubeConfig *k8s.KubeConfig, contextName string) *EventsTable {
	return &EventsTable{
		kubeConfig:   kubeConfig,
		contextName:  contextName,
		timeframeMin: 10, // Default to 10 minutes
		isLoading:    true,
	}
}

func (et *EventsTable) SetTimeframe(minutes int) {
	if minutes > 0 {
		et.timeframeMin = minutes
		// Force refresh on next update check
		et.lastUpdate = time.Time{}
		// Clear events to trigger loading state for timeframe change
		et.events = []k8s.EventInfo{}
	}
}

func (et *EventsTable) GetTimeframe() int {
	return et.timeframeMin
}

func (et *EventsTable) Update() error {
	if et.kubeConfig == nil {
		return fmt.Errorf("kubeconfig not available")
	}

	// Only set loading to true if this is the first load (no existing events)
	if len(et.events) == 0 {
		et.isLoading = true
	}
	et.error = nil

	events, err := et.kubeConfig.GetEvents(et.contextName, et.timeframeMin)
	if err != nil {
		et.error = err
		et.isLoading = false
		return err
	}

	et.events = events
	et.lastUpdate = time.Now()
	et.isLoading = false
	return nil
}

func (et *EventsTable) ShouldUpdate() bool {
	// Update every 15 seconds for events (more frequent than other resources)
	return time.Since(et.lastUpdate) > 15*time.Second
}

func (et *EventsTable) Render() string {
	var b strings.Builder

	// Only show loading screen if we have no events AND it's the initial load
	if et.isLoading && len(et.events) == 0 && et.lastUpdate.IsZero() {
		b.WriteString(styles.NormalStyle.Render("Loading events..."))
		return b.String()
	}

	if et.error != nil {
		errorStyle := styles.NormalStyle.Foreground(lipgloss.Color("196"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error loading events: %v", et.error)))
		return b.String()
	}

	// Timeframe info - show updating status more subtly
	timeframeStyle := styles.NormalStyle.Foreground(lipgloss.Color("245"))
	timeframeText := fmt.Sprintf("Showing events from the past %d minutes", et.timeframeMin)
	if et.isLoading && len(et.events) > 0 {
		// Show a subtle updating indicator only if we already have data
		timeframeText += " ●"
	}
	b.WriteString(timeframeStyle.Render(timeframeText) + "\n")

	// Controls info
	controlsStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
	b.WriteString(controlsStyle.Render("Use 't' to change timeframe • Events auto-refresh every 15s") + "\n\n")

	if len(et.events) == 0 {
		b.WriteString(styles.NormalStyle.Render("No events found in the specified timeframe"))
		return b.String()
	}

	// Table header
	headerStyle := styles.NormalStyle.Bold(true).Underline(true)
	header := fmt.Sprintf("%-8s %-12s %-15s %-20s %-8s %-15s %s",
		"TYPE", "REASON", "OBJECT", "MESSAGE", "COUNT", "NAMESPACE", "AGE")
	b.WriteString(headerStyle.Render(header) + "\n")

	// Table rows
	for i, event := range et.events {
		if i >= 50 { // Limit to 50 events for performance
			moreStyle := styles.NormalStyle.Foreground(lipgloss.Color("240"))
			b.WriteString(moreStyle.Render(fmt.Sprintf("\n... and %d more events (adjust timeframe to see fewer)", len(et.events)-50)))
			break
		}

		// Truncate and format fields
		eventType := truncateString(event.Type, 8)
		reason := truncateString(event.Reason, 12)
		object := truncateString(event.Object, 15)
		message := truncateString(event.Message, 20)
		count := truncateString(fmt.Sprintf("%d", event.Count), 8)
		namespace := truncateString(event.Namespace, 15)
		age := formatEventAge(event)

		row := fmt.Sprintf("%-8s %-12s %-15s %-20s %-8s %-15s %s",
			eventType, reason, object, message, count, namespace, age)

		// Color based on event type
		var rowStyle lipgloss.Style
		switch strings.ToLower(event.Type) {
		case "warning":
			rowStyle = styles.NormalStyle.Foreground(lipgloss.Color("226")) // Yellow
		case "error":
			rowStyle = styles.NormalStyle.Foreground(lipgloss.Color("196")) // Red
		default:
			rowStyle = styles.NormalStyle.Foreground(lipgloss.Color("252")) // White/Default
		}

		b.WriteString(rowStyle.Render(row))
		if i < len(et.events)-1 && i < 49 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// formatEventAge formats the age of an event
func formatEventAge(event k8s.EventInfo) string {
	// Use LastTimestamp if available, otherwise FirstTimestamp
	eventTime := event.LastTimestamp
	if eventTime.IsZero() {
		eventTime = event.FirstTimestamp
	}

	if eventTime.IsZero() {
		return "unknown"
	}

	duration := time.Since(eventTime)

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

// HandleTimeframeInput processes user input for timeframe changes
func (et *EventsTable) HandleTimeframeInput(input string) error {
	minutes, err := strconv.Atoi(input)
	if err != nil {
		return fmt.Errorf("invalid timeframe: must be a number in minutes")
	}

	if minutes <= 0 {
		return fmt.Errorf("timeframe must be greater than 0 minutes")
	}

	if minutes > 1440 { // 24 hours
		return fmt.Errorf("timeframe cannot exceed 1440 minutes (24 hours)")
	}

	et.SetTimeframe(minutes)
	return nil
}
