package k8s

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetClusterMetrics retrieves comprehensive cluster metrics including nodes, pods, and events
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

	// Get events (warnings and errors for overview)
	events, err := k.getRecentEvents(ctx, clientset)
	if err != nil {
		// Don't fail if we can't get events, just return empty slice
		metrics.Events = []EventInfo{}
	} else {
		metrics.Events = events
	}

	return metrics, nil
}

// calculatePodMetrics computes aggregated pod statistics
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

// Helper functions for formatting metrics

// FormatBytes formats bytes into human-readable units (B, KiB, MiB, GiB, etc.)
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

// FormatMilliCPU formats millicpu values into human-readable strings
func FormatMilliCPU(milliCPU int64) string {
	if milliCPU < 1000 {
		return fmt.Sprintf("%dm", milliCPU)
	}
	return fmt.Sprintf("%.2f", float64(milliCPU)/1000)
}

// FormatPercentage calculates and formats a percentage from used/total values
func FormatPercentage(used, total int64) string {
	if total == 0 {
		return "0%"
	}
	percentage := float64(used) / float64(total) * 100
	return fmt.Sprintf("%.1f%%", percentage)
}
