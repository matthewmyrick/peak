package ui

import (
	"strings"

	"peek/src/styles"
)

type RightPane struct {
	SelectedItem string
	Width        int
	Height       int
	SearchMode   bool
}

func NewRightPane(width, height int) *RightPane {
	return &RightPane{
		Width:  width,
		Height: height,
	}
}

func (rp *RightPane) SetSelectedItem(item string) {
	rp.SelectedItem = item
}

func (rp *RightPane) SetSearchMode(searchMode bool) {
	rp.SearchMode = searchMode
}

func (rp *RightPane) Render() string {
	var b strings.Builder
	
	if rp.SelectedItem != "" {
		b.WriteString(styles.HeaderStyle.Render(rp.SelectedItem) + "\n\n")
		b.WriteString(styles.NormalStyle.Render("Content will appear here"))
	} else {
		b.WriteString(styles.HeaderStyle.Render("Welcome to Peek") + "\n\n")
		b.WriteString(styles.NormalStyle.Render("Select an item from the left panel to view details\n\n"))
		b.WriteString(styles.NormalStyle.Render("Navigation:\n"))
		b.WriteString(styles.NormalStyle.Render("  ↑/k     - Move up\n"))
		b.WriteString(styles.NormalStyle.Render("  ↓/j     - Move down\n"))
		b.WriteString(styles.NormalStyle.Render("  →/l/↵   - Expand/Select\n"))
		b.WriteString(styles.NormalStyle.Render("  ←/h     - Collapse\n"))
		b.WriteString(styles.NormalStyle.Render("  /       - Search\n"))
		b.WriteString(styles.NormalStyle.Render("  q       - Quit\n"))
		if rp.SearchMode {
			b.WriteString(styles.NormalStyle.Render("\nSearch Mode:\n"))
			b.WriteString(styles.NormalStyle.Render("  Type to search\n"))
			b.WriteString(styles.NormalStyle.Render("  Esc     - Exit search\n"))
			b.WriteString(styles.NormalStyle.Render("  Backspace - Delete character\n"))
		}
	}
	
	return b.String()
}