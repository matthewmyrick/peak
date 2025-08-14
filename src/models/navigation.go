package models

import (
	"encoding/json"
	"os"
)

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

type NavigationConfig struct {
	Navigation []struct {
		Name  string   `json:"name"`
		Items []string `json:"items"`
	} `json:"navigation"`
}

func GetInitialNavItems() []NavItem {
	// Try to load from JSON file first
	if items := loadFromJSON(); items != nil {
		return items
	}

	// Fallback to hardcoded values
	return []NavItem{
		{Name: "Overview", Items: []string{
			"Cluster Info", "Namespaces", "Resource Usage", "Events",
		}, Expanded: true, Level: 0},
		{Name: "Applications", Items: []string{}, Expanded: false, Level: 0},
		{Name: "Nodes", Items: []string{}, Expanded: false, Level: 0},
		{Name: "Workloads", Items: []string{
			"Overview", "Pods", "Deployments", "DaemonSets", "StatefulSets",
			"ReplicaSets", "ReplicationControllers", "Jobs", "CronJobs",
		}, Expanded: false, Level: 0},
	}
}

func loadFromJSON() []NavItem {
	data, err := os.ReadFile("src/config/navigation.json")
	if err != nil {
		return nil
	}

	var config NavigationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil
	}

	var items []NavItem
	for _, item := range config.Navigation {
		items = append(items, NavItem{
			Name:     item.Name,
			Items:    item.Items,
			Expanded: false,
			Level:    0,
		})
	}

	return items
}
