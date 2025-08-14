package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ContextSelector struct {
	contexts         []string
	filteredContexts []string
	cursor           int
	SearchQuery      string
	isOpen           bool
	selectedContext  string
	originalContext  string
	width            int
	height           int
	isConnecting     bool
	connectionError  string
	spinnerFrame     int
	spinnerFrames    []string
}

func NewContextSelector(contexts []string, currentContext string) *ContextSelector {
	// Find the index of the current context
	cursorPos := 0
	for i, ctx := range contexts {
		if ctx == currentContext {
			cursorPos = i
			break
		}
	}

	return &ContextSelector{
		contexts:         contexts,
		filteredContexts: contexts,
		cursor:           cursorPos,
		SearchQuery:      "",
		isOpen:           true, // Start open on app launch
		selectedContext:  currentContext,
		originalContext:  currentContext,
		width:            60,
		height:           20,
		isConnecting:     false,
		connectionError:  "",
		spinnerFrame:     0,
		spinnerFrames:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

func (cs *ContextSelector) Open() {
	cs.isOpen = true
	cs.SearchQuery = ""
	cs.filteredContexts = cs.contexts
	cs.isConnecting = false // Clear any connecting state
	cs.connectionError = "" // Clear any error state
	// Set cursor to current context
	for i, ctx := range cs.filteredContexts {
		if ctx == cs.selectedContext {
			cs.cursor = i
			break
		}
	}
}

func (cs *ContextSelector) Close() {
	cs.isOpen = false
	cs.SearchQuery = ""
	cs.cursor = 0
}

func (cs *ContextSelector) IsOpen() bool {
	return cs.isOpen
}

func (cs *ContextSelector) GetSelectedContext() string {
	return cs.selectedContext
}

func (cs *ContextSelector) MoveUp() {
	if cs.cursor > 0 {
		cs.cursor--
	}
}

func (cs *ContextSelector) MoveDown() {
	if cs.cursor < len(cs.filteredContexts)-1 {
		cs.cursor++
	}
}

func (cs *ContextSelector) Select() {
	if cs.cursor < len(cs.filteredContexts) {
		cs.selectedContext = cs.filteredContexts[cs.cursor]
		// Don't close immediately - wait for connection validation
	}
}

func (cs *ContextSelector) SetConnecting(connecting bool) {
	cs.isConnecting = connecting
	if connecting {
		cs.connectionError = ""
	}
}

func (cs *ContextSelector) SetConnectionError(err string) {
	cs.connectionError = err
	cs.isConnecting = false
}

func (cs *ContextSelector) ClearError() {
	cs.connectionError = ""
}

func (cs *ContextSelector) UpdateSpinner() {
	if cs.isConnecting {
		cs.spinnerFrame = (cs.spinnerFrame + 1) % len(cs.spinnerFrames)
	}
}

func (cs *ContextSelector) IsConnecting() bool {
	return cs.isConnecting
}

func (cs *ContextSelector) UpdateSearch(query string) {
	cs.SearchQuery = query
	cs.filterContexts()
	cs.cursor = 0
}

func (cs *ContextSelector) filterContexts() {
	if cs.SearchQuery == "" {
		cs.filteredContexts = cs.contexts
		return
	}

	var filtered []string
	query := strings.ToLower(cs.SearchQuery)

	// First, add exact prefix matches
	for _, context := range cs.contexts {
		if strings.HasPrefix(strings.ToLower(context), query) {
			filtered = append(filtered, context)
		}
	}

	// Then add fuzzy matches that weren't already added
	for _, context := range cs.contexts {
		if !strings.HasPrefix(strings.ToLower(context), query) && fuzzyMatchContext(strings.ToLower(context), query) {
			filtered = append(filtered, context)
		}
	}

	cs.filteredContexts = filtered
}

func fuzzyMatchContext(str, pattern string) bool {
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

func (cs *ContextSelector) Render(screenWidth, screenHeight int) string {
	if !cs.isOpen {
		return ""
	}

	// Create modal style
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")). // Orange for context selector
		Width(cs.width).
		Height(cs.height).
		Padding(1).
		Background(lipgloss.Color("235"))

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true).
		MarginBottom(1)

	title := titleStyle.Render("Select Kubernetes Context")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)

	subtitle := subtitleStyle.Render("Press Enter to use current context or select a different one")

	// Search box
	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("237")).
		Padding(0, 1).
		Width(cs.width - 4)

	searchBox := searchStyle.Render("Search: " + cs.SearchQuery + "│")

	// Context list
	var contextList strings.Builder

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true).
		Width(cs.width - 4)

	currentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	maxItems := cs.height - 8 // Account for title, subtitle, search, padding, borders
	startIdx := 0
	endIdx := len(cs.filteredContexts)

	// Scroll the view if cursor is outside visible range
	if cs.cursor >= maxItems {
		startIdx = cs.cursor - maxItems + 1
		endIdx = cs.cursor + 1
	} else if endIdx > maxItems {
		endIdx = maxItems
	}

	for i := startIdx; i < endIdx && i < len(cs.filteredContexts); i++ {
		context := cs.filteredContexts[i]
		line := "  " + context

		// Mark the original/current context
		if context == cs.originalContext {
			line = "◉ " + context + " (current)"
		}

		if i == cs.cursor {
			contextList.WriteString(selectedStyle.Render(line))
		} else if context == cs.originalContext {
			contextList.WriteString(currentStyle.Render(line))
		} else {
			contextList.WriteString(itemStyle.Render(line))
		}

		if i < endIdx-1 && i < len(cs.filteredContexts)-1 {
			contextList.WriteString("\n")
		}
	}

	if len(cs.filteredContexts) == 0 {
		contextList.WriteString(itemStyle.Render("  No matching contexts"))
	}

	// Show loading or error state if applicable
	var statusMessage string
	if cs.isConnecting {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)
		spinner := cs.spinnerFrames[cs.spinnerFrame]
		statusMessage = loadingStyle.Render(spinner + " Connecting to cluster...")
	} else if cs.connectionError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Width(cs.width - 4)
		statusMessage = errorStyle.Render("✗ " + cs.connectionError)
	}

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		MarginTop(1)

	var instructions string
	if cs.isConnecting {
		instructions = instructionStyle.Render("Please wait...")
	} else if cs.connectionError != "" {
		instructions = instructionStyle.Render("↑/↓ Navigate • Enter Retry • Ctrl+Q Quit")
	} else {
		instructions = instructionStyle.Render("↑/↓ Navigate • Enter Select • Ctrl+Q Quit")
	}

	// Combine all elements
	var elements []string
	elements = append(elements, title, subtitle, "", searchBox, "", contextList.String())

	if statusMessage != "" {
		elements = append(elements, "", statusMessage)
	}

	elements = append(elements, "", instructions)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		elements...,
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
