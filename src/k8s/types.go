package k8s

import (
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// KubeConfig holds the Kubernetes configuration and client information
type KubeConfig struct {
	CurrentContext string
	Contexts       []string
	config         *api.Config
	clientConfig   clientcmd.ClientConfig
}

// NodeInfo represents information about a Kubernetes node
type NodeInfo struct {
	Name         string
	Status       string
	Roles        []string
	Age          string
	Version      string
	OS           string
	Architecture string
	CPUCapacity  string
	MemCapacity  string
	Ready        bool
	LastUpdated  time.Time
}

// ApplicationInfo represents information about Kubernetes application workloads
type ApplicationInfo struct {
	Name           string
	Type           string // Deployment, DaemonSet, StatefulSet, ReplicaSet, Job, CronJob
	Namespace      string
	Status         string
	Replicas       int32
	ReadyReplicas  int32
	CreationTime   time.Time
	Labels         map[string]string
	Conditions     []string
}

// PodInfo represents information about a Kubernetes pod
type PodInfo struct {
	Name            string
	Namespace       string
	Status          string
	Phase           string
	Ready           string // e.g., "2/3"
	Restarts        int32
	Age             time.Duration
	CreationTime    time.Time
	Node            string
	IP              string
	Labels          map[string]string
	Containers      []ContainerInfo
	OwnerReferences []string
}

// ContainerInfo represents information about a container in a pod
type ContainerInfo struct {
	Name         string
	Image        string
	Ready        bool
	RestartCount int32
	State        string
	Reason       string
}

// EventInfo represents information about a Kubernetes event
type EventInfo struct {
	Type           string
	Reason         string
	Object         string
	Message        string
	Count          int32
	FirstTimestamp time.Time
	LastTimestamp  time.Time
	Namespace      string
	Source         string
}

// ClusterMetrics holds various cluster-wide metrics
type ClusterMetrics struct {
	Nodes      NodeMetrics
	Pods       PodMetrics
	Events     []EventInfo
	LastUpdate time.Time
}

// NodeMetrics represents aggregated node statistics
type NodeMetrics struct {
	Total        int
	Ready        int
	NotReady     int
	CPUCapacity  int64
	CPUAllocated int64
	MemCapacity  int64
	MemAllocated int64
}

// PodMetrics represents aggregated pod statistics
type PodMetrics struct {
	Total     int
	Running   int
	Pending   int
	Failed    int
	Succeeded int
	Unknown   int
}

// ErrorType represents different types of Kubernetes errors
type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorTimeout
	ErrorUnauthorized
	ErrorNetwork
)
