package k8s

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetApplications retrieves application workloads from the specified Kubernetes context and namespace
func (k *KubeConfig) GetApplications(contextName, namespace string) ([]ApplicationInfo, error) {
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

	// Create a context with timeout for the API calls
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var applications []ApplicationInfo

	// Get Deployments
	deployments, err := getDeployments(ctx, clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}
	applications = append(applications, deployments...)

	// Get DaemonSets
	daemonSets, err := getDaemonSets(ctx, clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonsets: %w", err)
	}
	applications = append(applications, daemonSets...)

	// Get StatefulSets
	statefulSets, err := getStatefulSets(ctx, clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulsets: %w", err)
	}
	applications = append(applications, statefulSets...)

	// Get ReplicaSets (only standalone ones, not owned by Deployments)
	replicaSets, err := getReplicaSets(ctx, clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get replicasets: %w", err)
	}
	applications = append(applications, replicaSets...)

	// Get Jobs
	jobs, err := getJobs(ctx, clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}
	applications = append(applications, jobs...)

	// Get CronJobs
	cronJobs, err := getCronJobs(ctx, clientset, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get cronjobs: %w", err)
	}
	applications = append(applications, cronJobs...)

	return applications, nil
}

func getDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]ApplicationInfo, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var applications []ApplicationInfo
	for _, deployment := range deployments.Items {
		status := getDeploymentStatus(&deployment)
		app := ApplicationInfo{
			Name:          deployment.Name,
			Type:          "Deployment",
			Namespace:     deployment.Namespace,
			Status:        status,
			Replicas:      *deployment.Spec.Replicas,
			ReadyReplicas: deployment.Status.ReadyReplicas,
			CreationTime:  deployment.CreationTimestamp.Time,
			Labels:        deployment.Labels,
			Conditions:    getDeploymentConditions(&deployment),
		}
		applications = append(applications, app)
	}

	return applications, nil
}

func getDaemonSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]ApplicationInfo, error) {
	daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var applications []ApplicationInfo
	for _, daemonSet := range daemonSets.Items {
		status := getDaemonSetStatus(&daemonSet)
		app := ApplicationInfo{
			Name:          daemonSet.Name,
			Type:          "DaemonSet",
			Namespace:     daemonSet.Namespace,
			Status:        status,
			Replicas:      daemonSet.Status.DesiredNumberScheduled,
			ReadyReplicas: daemonSet.Status.NumberReady,
			CreationTime:  daemonSet.CreationTimestamp.Time,
			Labels:        daemonSet.Labels,
		}
		applications = append(applications, app)
	}

	return applications, nil
}

func getStatefulSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]ApplicationInfo, error) {
	statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var applications []ApplicationInfo
	for _, statefulSet := range statefulSets.Items {
		status := getStatefulSetStatus(&statefulSet)
		app := ApplicationInfo{
			Name:          statefulSet.Name,
			Type:          "StatefulSet",
			Namespace:     statefulSet.Namespace,
			Status:        status,
			Replicas:      *statefulSet.Spec.Replicas,
			ReadyReplicas: statefulSet.Status.ReadyReplicas,
			CreationTime:  statefulSet.CreationTimestamp.Time,
			Labels:        statefulSet.Labels,
		}
		applications = append(applications, app)
	}

	return applications, nil
}

func getReplicaSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]ApplicationInfo, error) {
	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var applications []ApplicationInfo
	for _, replicaSet := range replicaSets.Items {
		// Skip ReplicaSets that are owned by Deployments
		if isOwnedByDeployment(&replicaSet) {
			continue
		}

		status := getReplicaSetStatus(&replicaSet)
		app := ApplicationInfo{
			Name:          replicaSet.Name,
			Type:          "ReplicaSet",
			Namespace:     replicaSet.Namespace,
			Status:        status,
			Replicas:      *replicaSet.Spec.Replicas,
			ReadyReplicas: replicaSet.Status.ReadyReplicas,
			CreationTime:  replicaSet.CreationTimestamp.Time,
			Labels:        replicaSet.Labels,
		}
		applications = append(applications, app)
	}

	return applications, nil
}

func getJobs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]ApplicationInfo, error) {
	jobs, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var applications []ApplicationInfo
	for _, job := range jobs.Items {
		// Skip Jobs that are owned by CronJobs
		if isOwnedByCronJob(&job) {
			continue
		}

		status := getJobStatus(&job)
		replicas := int32(1)
		if job.Spec.Parallelism != nil {
			replicas = *job.Spec.Parallelism
		}
		
		app := ApplicationInfo{
			Name:          job.Name,
			Type:          "Job",
			Namespace:     job.Namespace,
			Status:        status,
			Replicas:      replicas,
			ReadyReplicas: job.Status.Succeeded,
			CreationTime:  job.CreationTimestamp.Time,
			Labels:        job.Labels,
		}
		applications = append(applications, app)
	}

	return applications, nil
}

func getCronJobs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]ApplicationInfo, error) {
	// Try v1 first, then fall back to v1beta1 for older clusters
	cronJobs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		// Fall back to v1beta1
		cronJobsV1Beta1, err := clientset.BatchV1beta1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		return convertCronJobsV1Beta1(cronJobsV1Beta1), nil
	}

	var applications []ApplicationInfo
	for _, cronJob := range cronJobs.Items {
		status := getCronJobStatus(&cronJob)
		app := ApplicationInfo{
			Name:         cronJob.Name,
			Type:         "CronJob",
			Namespace:    cronJob.Namespace,
			Status:       status,
			Replicas:     1, // CronJobs don't have replicas, use 1 for display
			ReadyReplicas: 1,
			CreationTime: cronJob.CreationTimestamp.Time,
			Labels:       cronJob.Labels,
		}
		applications = append(applications, app)
	}

	return applications, nil
}

// Helper functions for status determination
func getDeploymentStatus(deployment *appsv1.Deployment) string {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == "False" {
			return "Failed"
		}
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == "True" {
			if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
				return "Running"
			}
			return "Progressing"
		}
	}
	return "Pending"
}

func getDaemonSetStatus(daemonSet *appsv1.DaemonSet) string {
	if daemonSet.Status.NumberReady == daemonSet.Status.DesiredNumberScheduled {
		return "Running"
	}
	if daemonSet.Status.NumberReady > 0 {
		return "Progressing"
	}
	return "Pending"
}

func getStatefulSetStatus(statefulSet *appsv1.StatefulSet) string {
	if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
		return "Running"
	}
	if statefulSet.Status.ReadyReplicas > 0 {
		return "Progressing"
	}
	return "Pending"
}

func getReplicaSetStatus(replicaSet *appsv1.ReplicaSet) string {
	if replicaSet.Status.ReadyReplicas == *replicaSet.Spec.Replicas {
		return "Running"
	}
	if replicaSet.Status.ReadyReplicas > 0 {
		return "Progressing"
	}
	return "Pending"
}

func getJobStatus(job *batchv1.Job) string {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == "True" {
			return "Complete"
		}
		if condition.Type == batchv1.JobFailed && condition.Status == "True" {
			return "Failed"
		}
	}
	if job.Status.Active > 0 {
		return "Running"
	}
	return "Pending"
}

func getCronJobStatus(cronJob *batchv1.CronJob) string {
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		return "Suspended"
	}
	if len(cronJob.Status.Active) > 0 {
		return "Running"
	}
	return "Ready"
}

// Helper functions
func isOwnedByDeployment(replicaSet *appsv1.ReplicaSet) bool {
	for _, owner := range replicaSet.OwnerReferences {
		if owner.Kind == "Deployment" {
			return true
		}
	}
	return false
}

func isOwnedByCronJob(job *batchv1.Job) bool {
	for _, owner := range job.OwnerReferences {
		if owner.Kind == "CronJob" {
			return true
		}
	}
	return false
}

func getDeploymentConditions(deployment *appsv1.Deployment) []string {
	var conditions []string
	for _, condition := range deployment.Status.Conditions {
		if condition.Status == "True" {
			conditions = append(conditions, string(condition.Type))
		}
	}
	return conditions
}

func convertCronJobsV1Beta1(cronJobsV1Beta1 *batchv1beta1.CronJobList) []ApplicationInfo {
	var applications []ApplicationInfo
	for _, cronJob := range cronJobsV1Beta1.Items {
		status := "Ready"
		if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
			status = "Suspended"
		}
		if len(cronJob.Status.Active) > 0 {
			status = "Running"
		}

		app := ApplicationInfo{
			Name:          cronJob.Name,
			Type:          "CronJob",
			Namespace:     cronJob.Namespace,
			Status:        status,
			Replicas:      1,
			ReadyReplicas: 1,
			CreationTime:  cronJob.CreationTimestamp.Time,
			Labels:        cronJob.Labels,
		}
		applications = append(applications, app)
	}
	return applications
}