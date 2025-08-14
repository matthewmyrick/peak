package ui

import (
	"strings"

	"peek/src/models"
	"peek/src/styles"
)

type LeftPane struct {
	NavItems      []models.NavItem
	Cursor        int
	Width         int
	Height        int
	SelectedItem  string
	SearchMode    bool
	SearchQuery   string
	FilteredItems []models.VisibleItem
}

func NewLeftPane(width, height int) *LeftPane {
	return &LeftPane{
		NavItems:    models.GetInitialNavItems(),
		Cursor:      0,
		Width:       width,
		Height:      height,
		SearchMode:  false,
		SearchQuery: "",
	}
}

func (lp *LeftPane) GetVisibleItems() []models.VisibleItem {
	if lp.SearchMode && lp.SearchQuery != "" {
		return lp.FilteredItems
	}

	var items []models.VisibleItem
	for i := range lp.NavItems {
		item := &lp.NavItems[i]
		items = append(items, models.VisibleItem{
			Name:     item.Name,
			Parent:   nil,
			IsFolder: len(item.Items) > 0,
			Level:    0,
		})
		if item.Expanded {
			for _, subItem := range item.Items {
				items = append(items, models.VisibleItem{
					Name:     subItem,
					Parent:   item,
					IsFolder: false,
					Level:    1,
				})
			}
		}
	}
	return items
}

func (lp *LeftPane) MoveUp() {
	if lp.Cursor > 0 {
		lp.Cursor--
	}
}

func (lp *LeftPane) MoveDown() {
	visibleItems := lp.GetVisibleItems()
	if lp.Cursor < len(visibleItems)-1 {
		lp.Cursor++
	}
}

func (lp *LeftPane) ToggleExpand() bool {
	visibleItems := lp.GetVisibleItems()
	if lp.Cursor < len(visibleItems) {
		item := visibleItems[lp.Cursor]

		if item.Parent == nil {
			// Check if this item has children
			navItemIndex := lp.getNavItemIndex(item.Name)
			if navItemIndex >= 0 && len(lp.NavItems[navItemIndex].Items) > 0 {
				// Item has children - toggle expansion (don't select)
				itemName := item.Name

				// Toggle the expansion
				lp.NavItems[navItemIndex].Expanded = !lp.NavItems[navItemIndex].Expanded

				// Find the item in the new visible list and update cursor
				newVisibleItems := lp.GetVisibleItems()
				for i, newItem := range newVisibleItems {
					if newItem.Name == itemName && newItem.Parent == nil {
						lp.Cursor = i
						break
					}
				}
				// Return false - we expanded a folder, didn't select a resource
				return false
			} else {
				// Item has no children - select it directly
				lp.SelectedItem = item.Name
				// Return true - we selected a resource
				return true
			}
		} else {
			// Selecting a leaf item - keep cursor on it
			lp.SelectedItem = item.Parent.Name + " > " + item.Name
			// Return true - we selected a resource
			return true
		}
	}
	// Return false if nothing happened
	return false
}

func (lp *LeftPane) Collapse() {
	visibleItems := lp.GetVisibleItems()
	if lp.Cursor < len(visibleItems) {
		item := visibleItems[lp.Cursor]
		if item.Parent == nil {
			// Remember which item we're on
			itemName := item.Name

			// Collapse the item
			for i := range lp.NavItems {
				if lp.NavItems[i].Name == item.Name {
					lp.NavItems[i].Expanded = false
					break
				}
			}

			// Find the item in the new visible list and update cursor
			newVisibleItems := lp.GetVisibleItems()
			for i, newItem := range newVisibleItems {
				if newItem.Name == itemName && newItem.Parent == nil {
					lp.Cursor = i
					break
				}
			}
		}
	}
}

func (lp *LeftPane) Render() string {
	var b strings.Builder
	b.WriteString(styles.HeaderStyle.Render("Kubernetes Resources") + "\n")

	if lp.SearchMode {
		searchPrefix := "Search: "
		searchLine := searchPrefix + lp.SearchQuery + "│"
		b.WriteString(styles.SearchStyle.Render(searchLine) + "\n\n")
	} else {
		b.WriteString("\n")
	}

	visibleItems := lp.GetVisibleItems()
	startIdx := 0
	endIdx := len(visibleItems)

	maxLines := lp.Height - 7
	if lp.SearchMode {
		maxLines = lp.Height - 8
	}

	if endIdx-startIdx > maxLines {
		if lp.Cursor >= maxLines {
			startIdx = lp.Cursor - maxLines + 1
			endIdx = lp.Cursor + 1
		} else {
			endIdx = maxLines
		}
	}

	for i := startIdx; i < endIdx && i < len(visibleItems); i++ {
		item := visibleItems[i]
		line := ""

		indent := strings.Repeat("  ", item.Level)

		if item.Parent == nil && len(lp.NavItems[lp.getNavItemIndex(item.Name)].Items) > 0 {
			if lp.SearchMode {
				// In search mode, always show as expanded if it has children
				line = indent + "▼ " + item.Name
			} else {
				expanded := lp.NavItems[lp.getNavItemIndex(item.Name)].Expanded
				if expanded {
					line = indent + "▼ " + item.Name
				} else {
					line = indent + "▶ " + item.Name
				}
			}
		} else if item.Parent != nil {
			line = indent + "  " + item.Name
		} else {
			line = indent + "  " + item.Name
		}

		if i == lp.Cursor {
			b.WriteString(styles.SelectedStyle.Render(line))
		} else if item.IsFolder {
			b.WriteString(styles.FolderStyle.Render(line))
		} else {
			b.WriteString(styles.ItemStyle.Render(line))
		}

		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (lp *LeftPane) getNavItemIndex(name string) int {
	for i, item := range lp.NavItems {
		if item.Name == name {
			return i
		}
	}
	return -1
}

func (lp *LeftPane) ToggleSearch() {
	// If exiting search mode, try to preserve cursor position
	var currentItemName string
	var currentParentName string

	if lp.SearchMode {
		// Remember what item we're currently on
		visibleItems := lp.GetVisibleItems()
		if lp.Cursor < len(visibleItems) {
			currentItem := visibleItems[lp.Cursor]
			currentItemName = currentItem.Name
			if currentItem.Parent != nil {
				currentParentName = currentItem.Parent.Name
			}
		}
	}

	lp.SearchMode = !lp.SearchMode

	if !lp.SearchMode {
		lp.SearchQuery = ""
		lp.FilteredItems = nil

		// Try to find the same item in the regular view
		if currentItemName != "" {
			newVisibleItems := lp.GetVisibleItems()
			for i, item := range newVisibleItems {
				// Match both name and parent for accuracy
				if item.Name == currentItemName {
					if (currentParentName == "" && item.Parent == nil) ||
						(currentParentName != "" && item.Parent != nil && item.Parent.Name == currentParentName) {
						lp.Cursor = i
						return
					}
				}
			}
		}

		// If we couldn't find the item, reset to 0
		lp.Cursor = 0
	}
}

func (lp *LeftPane) UpdateSearch(query string) {
	lp.SearchQuery = query
	lp.FilteredItems = lp.filterItems(query)
	lp.Cursor = 0
}

func (lp *LeftPane) filterItems(query string) []models.VisibleItem {
	if query == "" {
		return nil
	}

	var filtered []models.VisibleItem
	query = strings.ToLower(query)

	for i := range lp.NavItems {
		item := &lp.NavItems[i]
		parentMatches := strings.Contains(strings.ToLower(item.Name), query)

		// Check if any children match
		hasMatchingChildren := false
		for _, subItem := range item.Items {
			if strings.Contains(strings.ToLower(subItem), query) {
				hasMatchingChildren = true
				break
			}
		}

		// If parent matches or has matching children, include parent
		if parentMatches || hasMatchingChildren {
			filtered = append(filtered, models.VisibleItem{
				Name:     item.Name,
				Parent:   nil,
				IsFolder: len(item.Items) > 0,
				Level:    0,
			})

			// Add all children if parent has matching children or parent matches
			if hasMatchingChildren || parentMatches {
				for _, subItem := range item.Items {
					// Only show children that match, or all children if parent matches
					if parentMatches || strings.Contains(strings.ToLower(subItem), query) {
						filtered = append(filtered, models.VisibleItem{
							Name:     subItem,
							Parent:   item,
							IsFolder: false,
							Level:    1,
						})
					}
				}
			}
		}
	}

	return filtered
}
