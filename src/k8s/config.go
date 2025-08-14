package k8s

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type KubeConfig struct {
	CurrentContext string
	Contexts       []string
	config         *api.Config
	clientConfig   clientcmd.ClientConfig
}

func NewKubeConfig() (*KubeConfig, error) {
	// Get kubeconfig path
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Load config
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, err
	}

	// Build client config
	clientConfig := clientcmd.NewNonInteractiveClientConfig(
		*config,
		config.CurrentContext,
		&clientcmd.ConfigOverrides{},
		nil,
	)

	// Get context names
	var contexts []string
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}
	sort.Strings(contexts)

	return &KubeConfig{
		CurrentContext: config.CurrentContext,
		Contexts:       contexts,
		config:         config,
		clientConfig:   clientConfig,
	}, nil
}

func (k *KubeConfig) SwitchContext(contextName string) error {
	// Update the current context in memory
	k.config.CurrentContext = contextName
	k.CurrentContext = contextName

	// Rebuild client config with new context
	k.clientConfig = clientcmd.NewNonInteractiveClientConfig(
		*k.config,
		contextName,
		&clientcmd.ConfigOverrides{},
		nil,
	)

	return nil
}

func (k *KubeConfig) GetNamespaces(contextName string) ([]string, error) {
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
	restConfig.Timeout = 5 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create a context with timeout for the API call
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get namespaces
	namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		// Categorize the error for better user feedback
		errorType := categorizeError(err)
		
		// Return default namespaces with wrapped error
		defaultNamespaces := []string{"default", "kube-system", "kube-public", "kube-node-lease"}
		
		switch errorType {
		case ErrorTimeout:
			return defaultNamespaces, fmt.Errorf("connection timeout to cluster '%s': %w", contextName, err)
		case ErrorUnauthorized:
			return defaultNamespaces, fmt.Errorf("authentication failed for cluster '%s': %w", contextName, err)
		case ErrorNetwork:
			return defaultNamespaces, fmt.Errorf("cannot reach cluster '%s': %w", contextName, err)
		default:
			return defaultNamespaces, fmt.Errorf("failed to list namespaces: %w", err)
		}
	}

	var namespaces []string
	for _, ns := range namespaceList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	sort.Strings(namespaces)

	return namespaces, nil
}

type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorTimeout
	ErrorUnauthorized
	ErrorNetwork
)

func categorizeError(err error) ErrorType {
	if err == nil {
		return ErrorUnknown
	}

	errStr := err.Error()
	
	// Check for timeout errors
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return ErrorTimeout
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return ErrorTimeout
	}
	
	// Check for authentication errors
	if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "401") || 
	   strings.Contains(errStr, "forbidden") || strings.Contains(errStr, "403") {
		return ErrorUnauthorized
	}
	
	// Check for network errors
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no such host") ||
	   strings.Contains(errStr, "network is unreachable") || strings.Contains(errStr, "no route to host") {
		return ErrorNetwork
	}
	
	return ErrorUnknown
}

func (k *KubeConfig) GetCurrentNamespace() string {
	namespace, _, err := k.clientConfig.Namespace()
	if err != nil || namespace == "" {
		return "default"
	}
	return namespace
}