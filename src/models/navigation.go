package models

type NavItem struct {
	Name     string
	Items    []string
	Expanded bool
	Parent   *NavItem
	Level    int
}

type VisibleItem struct {
	Name     string
	Parent   *NavItem
	IsFolder bool
	Level    int
}

func GetInitialNavItems() []NavItem {
	return []NavItem{
		{Name: "Overview", Items: []string{}, Expanded: false, Level: 0},
		{Name: "Applications", Items: []string{}, Expanded: false, Level: 0},
		{Name: "Nodes", Items: []string{}, Expanded: false, Level: 0},
		{Name: "Workloads", Items: []string{
			"Overview",
			"Pods",
			"Deployments",
			"DaemonSets",
			"StatefulSets",
			"ReplicaSets",
			"ReplicationControllers",
			"Jobs",
			"CronJobs",
		}, Expanded: false, Level: 0},
		{Name: "Config", Items: []string{
			"ConfigMaps",
			"Secrets",
			"ResourceQuotas",
			"LimitRanges",
			"HorizontalPodAutoscalers",
			"PodDisruptionBudgets",
			"PriorityClasses",
			"RuntimeClasses",
			"Leases",
			"MutatingWebhookConfigurations",
			"ValidatingWebhookConfigurations",
		}, Expanded: false, Level: 0},
		{Name: "Network", Items: []string{
			"Services",
			"Endpoints",
			"Ingresses",
			"IngressControllers",
			"NetworkPolicies",
			"PortForwarding",
		}, Expanded: false, Level: 0},
		{Name: "Storage", Items: []string{
			"PersistentVolumeClaims",
			"PersistentVolumes",
			"StorageClasses",
		}, Expanded: false, Level: 0},
		{Name: "Namespaces", Items: []string{}, Expanded: false, Level: 0},
		{Name: "Events", Items: []string{}, Expanded: false, Level: 0},
		{Name: "Helm", Items: []string{
			"Charts",
			"Releases",
		}, Expanded: false, Level: 0},
		{Name: "AccessControl", Items: []string{
			"ServiceAccounts",
			"ClusterRoles",
			"Roles",
			"ClusterRoleBindings",
			"RoleBindings",
		}, Expanded: false, Level: 0},
		{Name: "CustomResources", Items: []string{
			"Definitions",
		}, Expanded: false, Level: 0},
	}
}