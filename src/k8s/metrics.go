package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

type ClusterMetrics struct {
	Nodes      NodeMetrics
	Pods       PodMetrics
	Events     []EventInfo
	LastUpdate time.Time
}

type NodeMetrics struct {
	Total         int
	Ready         int
	NotReady      int
	CPUCapacity   int64
	CPUAllocated  int64
	MemCapacity   int64
	MemAllocated  int64
}

type PodMetrics struct {
	Total     int
	Running   int
	Pending   int
	Failed    int
	Succeeded int
	Unknown   int
}

type EventInfo struct {
	Type      string
	Reason    string
	Object    string
	Message   string
	Count     int32
	FirstSeen time.Time
	LastSeen  time.Time
	Namespace string
}

func (k *KubeConfig) GetClusterMetrics(ctx context.Context) (*ClusterMetrics, error) {
	// Get client config
	restConfig, err := k.clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get client config: %w", err)
	}

	// Set timeout
	restConfig.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	metrics := &ClusterMetrics{
		LastUpdate: time.Now(),
	}

	// Get node metrics
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	metrics.Nodes = calculateNodeMetrics(nodes.Items)

	// Get pod metrics
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	metrics.Pods = calculatePodMetrics(pods.Items)

	// Get events (warnings and errors)
	events, err := k.GetClusterEvents(ctx, clientset)
	if err != nil {
		// Don't fail if we can't get events, just log
		metrics.Events = []EventInfo{}
	} else {
		metrics.Events = events
	}

	return metrics, nil
}

func calculateNodeMetrics(nodes []v1.Node) NodeMetrics {
	metrics := NodeMetrics{
		Total: len(nodes),
	}

	for _, node := range nodes {
		// Check node status
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady {
				if condition.Status == v1.ConditionTrue {
					metrics.Ready++
				} else {
					metrics.NotReady++
				}
				break
			}
		}

		// Calculate capacity
		if cpu := node.Status.Capacity.Cpu(); cpu != nil {
			metrics.CPUCapacity += cpu.MilliValue()
		}
		if mem := node.Status.Capacity.Memory(); mem != nil {
			metrics.MemCapacity += mem.Value()
		}

		// Calculate allocated resources (from pods)
		if cpu := node.Status.Allocatable.Cpu(); cpu != nil {
			metrics.CPUAllocated += cpu.MilliValue()
		}
		if mem := node.Status.Allocatable.Memory(); mem != nil {
			metrics.MemAllocated += mem.Value()
		}
	}

	return metrics
}

func calculatePodMetrics(pods []v1.Pod) PodMetrics {
	metrics := PodMetrics{
		Total: len(pods),
	}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case v1.PodRunning:
			metrics.Running++
		case v1.PodPending:
			metrics.Pending++
		case v1.PodFailed:
			metrics.Failed++
		case v1.PodSucceeded:
			metrics.Succeeded++
		default:
			metrics.Unknown++
		}
	}

	return metrics
}

func (k *KubeConfig) GetClusterEvents(ctx context.Context, clientset *kubernetes.Clientset) ([]EventInfo, error) {
	// Get events from all namespaces
	eventList, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("type", "Warning").String() + "," +
			fields.OneTermEqualSelector("type", "Error").String(),
	})

	// If field selector doesn't work, get all and filter
	if err != nil || len(eventList.Items) == 0 {
		eventList, err = clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list events: %w", err)
		}
	}

	var events []EventInfo
	for _, event := range eventList.Items {
		// Filter for Warning and Error events
		if event.Type != "Warning" && event.Type != "Error" && event.Type != "Failed" {
			continue
		}

		info := EventInfo{
			Type:      event.Type,
			Reason:    event.Reason,
			Object:    fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			Message:   event.Message,
			Count:     event.Count,
			FirstSeen: event.FirstTimestamp.Time,
			LastSeen:  event.LastTimestamp.Time,
			Namespace: event.Namespace,
		}

		// If timestamps are zero, use event metadata
		if info.FirstSeen.IsZero() && event.CreationTimestamp.Time != (time.Time{}) {
			info.FirstSeen = event.CreationTimestamp.Time
			info.LastSeen = event.CreationTimestamp.Time
		}

		events = append(events, info)
	}

	// Sort events by last seen time (most recent first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].LastSeen.After(events[j].LastSeen)
	})

	// Limit to most recent 20 events
	if len(events) > 20 {
		events = events[:20]
	}

	return events, nil
}

// Helper functions for formatting
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func FormatMilliCPU(milliCPU int64) string {
	if milliCPU < 1000 {
		return fmt.Sprintf("%dm", milliCPU)
	}
	return fmt.Sprintf("%.2f", float64(milliCPU)/1000)
}

func FormatPercentage(used, total int64) string {
	if total == 0 {
		return "0%"
	}
	percentage := float64(used) / float64(total) * 100
	return fmt.Sprintf("%.1f%%", percentage)
}

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func FormatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	
	duration := time.Since(t)
	
	if duration < time.Minute {
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
}

func GetEventColor(eventType string) string {
	switch strings.ToLower(eventType) {
	case "warning":
		return "214" // Orange
	case "error", "failed":
		return "196" // Red
	default:
		return "252" // Normal
	}
}