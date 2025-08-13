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
}

func NewLeftPane(width, height int) *LeftPane {
	return &LeftPane{
		NavItems: models.GetInitialNavItems(),
		Cursor:   0,
		Width:    width,
		Height:   height,
	}
}

func (lp *LeftPane) GetVisibleItems() []models.VisibleItem {
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

func (lp *LeftPane) ToggleExpand() {
	visibleItems := lp.GetVisibleItems()
	if lp.Cursor < len(visibleItems) {
		item := visibleItems[lp.Cursor]
		if item.Parent == nil {
			for i := range lp.NavItems {
				if lp.NavItems[i].Name == item.Name {
					lp.NavItems[i].Expanded = !lp.NavItems[i].Expanded
					break
				}
			}
		} else {
			lp.SelectedItem = item.Parent.Name + " > " + item.Name
		}
	}
}

func (lp *LeftPane) Collapse() {
	visibleItems := lp.GetVisibleItems()
	if lp.Cursor < len(visibleItems) {
		item := visibleItems[lp.Cursor]
		if item.Parent == nil {
			for i := range lp.NavItems {
				if lp.NavItems[i].Name == item.Name {
					lp.NavItems[i].Expanded = false
					break
				}
			}
		}
	}
}

func (lp *LeftPane) Render() string {
	var b strings.Builder
	b.WriteString(styles.HeaderStyle.Render("Kubernetes Resources") + "\n\n")

	visibleItems := lp.GetVisibleItems()
	startIdx := 0
	endIdx := len(visibleItems)

	maxLines := lp.Height - 6
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
			expanded := lp.NavItems[lp.getNavItemIndex(item.Name)].Expanded
			if expanded {
				line = indent + "▼ " + item.Name
			} else {
				line = indent + "▶ " + item.Name
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