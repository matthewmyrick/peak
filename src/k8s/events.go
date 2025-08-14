package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetEvents retrieves events from the specified Kubernetes context within the given timeframe
func (k *KubeConfig) GetEvents(contextName string, timeframeMinutes int) ([]EventInfo, error) {
	// Create a temporary client config for the specified context
	tempConfig := clientcmd.NewNonInteractiveClientConfig(
		*k.config,
		contextName,
		&clientcmd.ConfigOverrides{},
		nil,
	)

	restConfig, err := tempConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	// Set a reasonable timeout
	restConfig.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create a context with timeout for the API call
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Calculate time threshold
	timeThreshold := time.Now().Add(-time.Duration(timeframeMinutes) * time.Minute)

	// Get events from all namespaces
	eventList, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []EventInfo
	for _, event := range eventList.Items {
		// Filter events by timeframe - check LastTimestamp first, then FirstTimestamp
		eventTime := event.LastTimestamp.Time
		if eventTime.IsZero() {
			eventTime = event.FirstTimestamp.Time
		}

		// Skip events older than our timeframe
		if eventTime.Before(timeThreshold) {
			continue
		}

		eventInfo := EventInfo{
			Type:           event.Type,
			Reason:         event.Reason,
			Object:         fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			Message:        event.Message,
			Count:          event.Count,
			FirstTimestamp: event.FirstTimestamp.Time,
			LastTimestamp:  event.LastTimestamp.Time,
			Namespace:      event.Namespace,
			Source:         event.Source.Component,
		}

		events = append(events, eventInfo)
	}

	// Sort events by timestamp (most recent first)
	sort.Slice(events, func(i, j int) bool {
		timeI := events[i].LastTimestamp
		if timeI.IsZero() {
			timeI = events[i].FirstTimestamp
		}
		timeJ := events[j].LastTimestamp
		if timeJ.IsZero() {
			timeJ = events[j].FirstTimestamp
		}
		return timeI.After(timeJ)
	})

	return events, nil
}

// getRecentEvents retrieves recent warning and error events for metrics display
func (k *KubeConfig) getRecentEvents(ctx context.Context, clientset *kubernetes.Clientset) ([]EventInfo, error) {
	// Get events from the last 10 minutes for overview display
	eventList, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []EventInfo
	cutoff := time.Now().Add(-10 * time.Minute)

	for _, event := range eventList.Items {
		// Only include Warning, Error, and Failed events for overview
		if event.Type != "Warning" && event.Type != "Error" && event.Type != "Failed" {
			continue
		}

		info := EventInfo{
			Type:           event.Type,
			Reason:         event.Reason,
			Object:         fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			Message:        event.Message,
			Count:          event.Count,
			FirstTimestamp: event.FirstTimestamp.Time,
			LastTimestamp:  event.LastTimestamp.Time,
			Namespace:      event.Namespace,
			Source:         event.Source.Component,
		}

		// If timestamps are zero, use event metadata
		if info.FirstTimestamp.IsZero() && event.CreationTimestamp.Time != (time.Time{}) {
			info.FirstTimestamp = event.CreationTimestamp.Time
			info.LastTimestamp = event.CreationTimestamp.Time
		}

		// Skip events older than cutoff
		eventTime := info.LastTimestamp
		if eventTime.IsZero() {
			eventTime = info.FirstTimestamp
		}
		if eventTime.Before(cutoff) {
			continue
		}

		events = append(events, info)
	}

	// Sort events by last seen time (most recent first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastTimestamp.After(events[j].LastTimestamp)
	})

	// Limit to most recent 20 events
	if len(events) > 20 {
		events = events[:20]
	}

	return events, nil
}

// GetEventColor returns the appropriate color code for an event type
func GetEventColor(eventType string) string {
	switch strings.ToLower(eventType) {
	case "warning":
		return "226" // Yellow
	case "error", "failed":
		return "196" // Red
	default:
		return "252" // White/Default
	}
}

// FormatTimeAgo formats a timestamp into a human-readable "time ago" string
func FormatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	duration := time.Since(t)

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

// TruncateString truncates a string to the specified length, adding "..." if truncated
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
