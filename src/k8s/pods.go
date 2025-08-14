package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetPods retrieves pods from the specified Kubernetes context and namespace
func (k *KubeConfig) GetPods(contextName, namespace string) ([]PodInfo, error) {
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

	// Get pods
	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	var pods []PodInfo
	for _, pod := range podList.Items {
		podInfo := convertPodToPodInfo(&pod)
		pods = append(pods, podInfo)
	}

	return pods, nil
}

// GetPodLogs retrieves logs from a specific pod
func (k *KubeConfig) GetPodLogs(contextName, namespace, podName, containerName string, lines int64, follow bool) (io.ReadCloser, error) {
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

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Prepare log options
	podLogOpts := corev1.PodLogOptions{
		Container:  containerName,
		Follow:     follow,
		Timestamps: true,
	}
	
	if lines > 0 {
		podLogOpts.TailLines = &lines
	}

	// Get logs
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)
	logs, err := req.Stream(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	return logs, nil
}

// DeletePod deletes a pod
func (k *KubeConfig) DeletePod(contextName, namespace, podName string) error {
	// Create a temporary client config for the specified context
	tempConfig := clientcmd.NewNonInteractiveClientConfig(
		*k.config,
		contextName,
		&clientcmd.ConfigOverrides{},
		nil,
	)

	restConfig, err := tempConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to get client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Delete the pod
	err = clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}

	return nil
}

// RestartPod restarts a pod by deleting it (relies on controller to recreate)
func (k *KubeConfig) RestartPod(contextName, namespace, podName string) error {
	// For now, restarting a pod means deleting it
	// The controller (Deployment, DaemonSet, etc.) will recreate it
	return k.DeletePod(contextName, namespace, podName)
}

// GetPodYAML retrieves the YAML representation of a pod
func (k *KubeConfig) GetPodYAML(contextName, namespace, podName string) (string, error) {
	// Create a temporary client config for the specified context
	tempConfig := clientcmd.NewNonInteractiveClientConfig(
		*k.config,
		contextName,
		&clientcmd.ConfigOverrides{},
		nil,
	)

	restConfig, err := tempConfig.ClientConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get client config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create client: %w", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get the pod
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %w", err)
	}

	// Convert to YAML (simplified - in a real implementation you'd use proper serialization)
	yaml := fmt.Sprintf(`apiVersion: %s
kind: Pod
metadata:
  name: %s
  namespace: %s
  creationTimestamp: %s
  labels:
%s
spec:
  nodeName: %s
  containers:
%s
status:
  phase: %s
  podIP: %s
  startTime: %s`,
		pod.APIVersion,
		pod.Name,
		pod.Namespace,
		pod.CreationTimestamp.Format(time.RFC3339),
		formatLabelsYAML(pod.Labels),
		pod.Spec.NodeName,
		formatContainersYAML(pod.Spec.Containers),
		pod.Status.Phase,
		pod.Status.PodIP,
		formatTimePtr(pod.Status.StartTime))

	return yaml, nil
}

// Helper functions
func convertPodToPodInfo(pod *corev1.Pod) PodInfo {
	// Calculate ready containers
	readyCount := 0
	totalCount := len(pod.Status.ContainerStatuses)
	
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			readyCount++
		}
	}
	
	// Calculate total restarts
	var totalRestarts int32
	for _, status := range pod.Status.ContainerStatuses {
		totalRestarts += status.RestartCount
	}

	// Convert containers
	var containers []ContainerInfo
	for _, containerStatus := range pod.Status.ContainerStatuses {
		state := "Unknown"
		reason := ""
		
		if containerStatus.State.Running != nil {
			state = "Running"
		} else if containerStatus.State.Waiting != nil {
			state = "Waiting"
			reason = containerStatus.State.Waiting.Reason
		} else if containerStatus.State.Terminated != nil {
			state = "Terminated"
			reason = containerStatus.State.Terminated.Reason
		}
		
		// Find the corresponding container spec
		containerName := containerStatus.Name
		image := ""
		for _, container := range pod.Spec.Containers {
			if container.Name == containerName {
				image = container.Image
				break
			}
		}
		
		containers = append(containers, ContainerInfo{
			Name:         containerName,
			Image:        image,
			Ready:        containerStatus.Ready,
			RestartCount: containerStatus.RestartCount,
			State:        state,
			Reason:       reason,
		})
	}

	// Get owner references
	var owners []string
	for _, owner := range pod.OwnerReferences {
		owners = append(owners, fmt.Sprintf("%s/%s", owner.Kind, owner.Name))
	}

	age := time.Since(pod.CreationTimestamp.Time)

	return PodInfo{
		Name:            pod.Name,
		Namespace:       pod.Namespace,
		Status:          getPodStatus(pod),
		Phase:           string(pod.Status.Phase),
		Ready:           fmt.Sprintf("%d/%d", readyCount, totalCount),
		Restarts:        totalRestarts,
		Age:             age,
		CreationTime:    pod.CreationTimestamp.Time,
		Node:            pod.Spec.NodeName,
		IP:              pod.Status.PodIP,
		Labels:          pod.Labels,
		Containers:      containers,
		OwnerReferences: owners,
	}
}

func getPodStatus(pod *corev1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	// Check container states
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil && status.State.Waiting.Reason == "CrashLoopBackOff" {
			return "CrashLoopBackOff"
		}
		if status.State.Waiting != nil && status.State.Waiting.Reason == "ImagePullBackOff" {
			return "ImagePullBackOff"
		}
		if status.State.Waiting != nil {
			return status.State.Waiting.Reason
		}
		if status.State.Terminated != nil {
			return status.State.Terminated.Reason
		}
	}

	return string(pod.Status.Phase)
}

func formatLabelsYAML(labels map[string]string) string {
	if len(labels) == 0 {
		return "    {}"
	}
	
	var result strings.Builder
	for key, value := range labels {
		result.WriteString(fmt.Sprintf("    %s: %s\n", key, value))
	}
	return strings.TrimSuffix(result.String(), "\n")
}

func formatContainersYAML(containers []corev1.Container) string {
	var result strings.Builder
	for _, container := range containers {
		result.WriteString(fmt.Sprintf("  - name: %s\n", container.Name))
		result.WriteString(fmt.Sprintf("    image: %s\n", container.Image))
		if len(container.Ports) > 0 {
			result.WriteString("    ports:\n")
			for _, port := range container.Ports {
				result.WriteString(fmt.Sprintf("    - containerPort: %d\n", port.ContainerPort))
			}
		}
	}
	return strings.TrimSuffix(result.String(), "\n")
}

func formatTimePtr(t *metav1.Time) string {
	if t == nil {
		return "null"
	}
	return t.Format(time.RFC3339)
}