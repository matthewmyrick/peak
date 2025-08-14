package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type NamespaceSelector struct {
	namespaces         []string
	filteredNamespaces []string
	cursor             int
	SearchQuery        string
	isOpen             bool
	selectedNamespace  string
	width              int
	height             int
}

func NewNamespaceSelector(namespaces []string, currentNamespace string) *NamespaceSelector {
	if len(namespaces) == 0 {
		// Fallback namespaces if none provided
		namespaces = []string{
			"default",
			"kube-system",
			"kube-public",
			"kube-node-lease",
		}
	}

	// Add "All namespaces" option at the beginning
	namespacesWithAll := []string{"All namespaces"}
	namespacesWithAll = append(namespacesWithAll, namespaces...)

	if currentNamespace == "" {
		currentNamespace = "default"
	}

	return &NamespaceSelector{
		namespaces:         namespacesWithAll,
		filteredNamespaces: namespacesWithAll,
		cursor:             0,
		SearchQuery:        "",
		isOpen:             false,
		selectedNamespace:  currentNamespace,
		width:              50,
		height:             15,
	}
}

func (ns *NamespaceSelector) UpdateNamespaces(namespaces []string, currentNamespace string) {
	// Add "All namespaces" option at the beginning
	namespacesWithAll := []string{"All namespaces"}
	namespacesWithAll = append(namespacesWithAll, namespaces...)
	
	ns.namespaces = namespacesWithAll
	ns.filteredNamespaces = namespacesWithAll
	ns.selectedNamespace = currentNamespace
	ns.cursor = 0
	ns.SearchQuery = ""
}

func (ns *NamespaceSelector) Open() {
	ns.isOpen = true
	ns.SearchQuery = ""
	ns.cursor = 0
	ns.filteredNamespaces = ns.namespaces
}

func (ns *NamespaceSelector) Close() {
	ns.isOpen = false
	ns.SearchQuery = ""
	ns.cursor = 0
}

func (ns *NamespaceSelector) IsOpen() bool {
	return ns.isOpen
}

func (ns *NamespaceSelector) GetSelectedNamespace() string {
	if ns.selectedNamespace == "" {
		return "All namespaces"
	}
	return ns.selectedNamespace
}

func (ns *NamespaceSelector) GetSelectedNamespaceRaw() string {
	// Returns empty string for "All namespaces", actual namespace otherwise
	return ns.selectedNamespace
}

func (ns *NamespaceSelector) MoveUp() {
	if ns.cursor > 0 {
		ns.cursor--
	}
}

func (ns *NamespaceSelector) MoveDown() {
	if ns.cursor < len(ns.filteredNamespaces)-1 {
		ns.cursor++
	}
}

func (ns *NamespaceSelector) Select() {
	if ns.cursor < len(ns.filteredNamespaces) {
		selectedOption := ns.filteredNamespaces[ns.cursor]
		if selectedOption == "All namespaces" {
			ns.selectedNamespace = "" // Empty string means all namespaces
		} else {
			ns.selectedNamespace = selectedOption
		}
		ns.Close()
	}
}

func (ns *NamespaceSelector) UpdateSearch(query string) {
	ns.SearchQuery = query
	ns.filterNamespaces()
	ns.cursor = 0
}

func (ns *NamespaceSelector) filterNamespaces() {
	if ns.SearchQuery == "" {
		ns.filteredNamespaces = ns.namespaces
		return
	}

	var filtered []string
	query := strings.ToLower(ns.SearchQuery)

	// First, add exact prefix matches
	for _, namespace := range ns.namespaces {
		if strings.HasPrefix(strings.ToLower(namespace), query) {
			filtered = append(filtered, namespace)
		}
	}

	// Then add fuzzy matches that weren't already added
	for _, namespace := range ns.namespaces {
		if !strings.HasPrefix(strings.ToLower(namespace), query) && fuzzyMatch(strings.ToLower(namespace), query) {
			filtered = append(filtered, namespace)
		}
	}

	ns.filteredNamespaces = filtered
}

func fuzzyMatch(str, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternIdx := 0
	for i := 0; i < len(str) && patternIdx < len(pattern); i++ {
		if str[i] == pattern[patternIdx] {
			patternIdx++
		}
	}

	return patternIdx == len(pattern)
}

func (ns *NamespaceSelector) Render(screenWidth, screenHeight int) string {
	if !ns.isOpen {
		return ""
	}

	// Create modal style
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Width(ns.width).
		Height(ns.height).
		Padding(1).
		Background(lipgloss.Color("235"))

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true).
		MarginBottom(1)

	title := titleStyle.Render("Select Namespace")

	// Search box
	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("237")).
		Padding(0, 1).
		Width(ns.width - 4)

	searchBox := searchStyle.Render("Search: " + ns.SearchQuery + "│")

	// Namespace list
	var namespaceList strings.Builder

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true).
		Width(ns.width - 4)

	currentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	maxItems := ns.height - 6 // Account for title, search, padding, borders
	startIdx := 0
	endIdx := len(ns.filteredNamespaces)

	// Scroll the view if cursor is outside visible range
	if ns.cursor >= maxItems {
		startIdx = ns.cursor - maxItems + 1
		endIdx = ns.cursor + 1
	} else if endIdx > maxItems {
		endIdx = maxItems
	}

	for i := startIdx; i < endIdx && i < len(ns.filteredNamespaces); i++ {
		namespace := ns.filteredNamespaces[i]
		line := "  " + namespace

		if namespace == ns.selectedNamespace {
			line = "◉ " + namespace
		}

		if i == ns.cursor {
			namespaceList.WriteString(selectedStyle.Render(line))
		} else if namespace == ns.selectedNamespace {
			namespaceList.WriteString(currentStyle.Render(line))
		} else {
			namespaceList.WriteString(itemStyle.Render(line))
		}

		if i < endIdx-1 && i < len(ns.filteredNamespaces)-1 {
			namespaceList.WriteString("\n")
		}
	}

	if len(ns.filteredNamespaces) == 0 {
		namespaceList.WriteString(itemStyle.Render("  No matching namespaces"))
	}

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		searchBox,
		"",
		namespaceList.String(),
	)

	modalContent := modalStyle.Render(content)

	// Center the modal
	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		modalContent,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)
}
