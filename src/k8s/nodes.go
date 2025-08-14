package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetNodes retrieves all nodes from the specified Kubernetes context
func (k *KubeConfig) GetNodes(contextName string) ([]NodeInfo, error) {
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

	// Get nodes
	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var nodes []NodeInfo
	for _, node := range nodeList.Items {
		nodeInfo := NodeInfo{
			Name:        node.Name,
			LastUpdated: time.Now(),
		}

		// Extract node status
		nodeInfo.Ready = false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				nodeInfo.Ready = true
				nodeInfo.Status = "Ready"
				break
			}
		}
		if !nodeInfo.Ready {
			nodeInfo.Status = "NotReady"
		}

		// Extract roles
		roles := []string{}
		for label := range node.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
				if role == "" {
					role = "master"
				}
				roles = append(roles, role)
			}
		}
		if len(roles) == 0 {
			roles = append(roles, "worker")
		}
		sort.Strings(roles)
		nodeInfo.Roles = roles

		// Extract age
		age := time.Since(node.CreationTimestamp.Time)
		nodeInfo.Age = formatDuration(age)

		// Extract version
		nodeInfo.Version = node.Status.NodeInfo.KubeletVersion

		// Extract OS and architecture
		nodeInfo.OS = node.Status.NodeInfo.OperatingSystem
		nodeInfo.Architecture = node.Status.NodeInfo.Architecture

		// Extract capacity
		if cpu, ok := node.Status.Capacity[corev1.ResourceCPU]; ok {
			nodeInfo.CPUCapacity = cpu.String()
		}
		if mem, ok := node.Status.Capacity[corev1.ResourceMemory]; ok {
			nodeInfo.MemCapacity = formatBytes(mem.Value())
		}

		nodes = append(nodes, nodeInfo)
	}

	// Sort nodes alphabetically by name
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, nil
}

// calculateNodeMetrics computes aggregated metrics for a list of nodes
func calculateNodeMetrics(nodes []corev1.Node) NodeMetrics {
	metrics := NodeMetrics{}

	for _, node := range nodes {
		metrics.Total++

		// Check if node is ready
		ready := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}

		if ready {
			metrics.Ready++
		} else {
			metrics.NotReady++
		}

		// Aggregate capacity
		if cpu, ok := node.Status.Capacity[corev1.ResourceCPU]; ok {
			metrics.CPUCapacity += cpu.MilliValue()
		}
		if mem, ok := node.Status.Capacity[corev1.ResourceMemory]; ok {
			metrics.MemCapacity += mem.Value()
		}

		// Aggregate allocatable (available resources)
		if cpu, ok := node.Status.Allocatable[corev1.ResourceCPU]; ok {
			metrics.CPUAllocated += cpu.MilliValue()
		}
		if mem, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
			metrics.MemAllocated += mem.Value()
		}
	}

	return metrics
}

// formatDuration formats a time duration into a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%ds", int(d.Seconds()))
}

// formatBytes formats bytes into human-readable units (Ki, Mi, Gi, Ti)
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fTi", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1fGi", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMi", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKi", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
