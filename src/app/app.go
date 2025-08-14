package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"peek/src/k8s"
	"peek/src/styles"
	"peek/src/ui"
)

type FocusedPane int

const (
	FocusLeftPane FocusedPane = iota
	FocusRightPane
)

type Model struct {
	leftPane          *ui.LeftPane
	rightPane         *ui.RightPane
	footer            *ui.Footer
	namespaceSelector *ui.NamespaceSelector
	contextSelector   *ui.ContextSelector
	notifications     *ui.NotificationManager
	kubeConfig        *k8s.KubeConfig
	loadingSpinner    *ui.LoadingSpinner
	timeframeInputPane *ui.TimeframeInput
	width             int
	height            int
	leftPaneWidth     int
	rightPaneWidth    int
	isLoading         bool
	isConnected       bool
	initError         error
	focusedPane       FocusedPane
}

func InitialModel() Model {
	leftPaneWidth := 35
	leftPane := ui.NewLeftPane(leftPaneWidth, 24)
	rightPane := ui.NewRightPane(45, 24)
	footer := ui.NewFooter(80)
	notifications := ui.NewNotificationManager()
	loadingSpinner := ui.NewLoadingSpinner("Connecting to Kubernetes cluster...")
	timeframeInputPane := ui.NewTimeframeInput()

	// Connect notifications to right pane
	rightPane.SetNotifications(notifications)

	return Model{
		leftPane:           leftPane,
		rightPane:          rightPane,
		footer:             footer,
		notifications:      notifications,
		loadingSpinner:     loadingSpinner,
		timeframeInputPane: timeframeInputPane,
		leftPaneWidth:      leftPaneWidth,
		width:              80,
		height:             24,
		isLoading:          true,
		isConnected:        false,
		focusedPane:        FocusLeftPane, // Start with left pane focused
	}
}

func (m Model) Init() tea.Cmd {
	// Start both the connection check and ticker
	return tea.Batch(
		connectToClusterCmd(),
		tickCmd(),
	)
}

type tickMsg time.Time
type connectionResultMsg struct {
	kubeConfig        *k8s.KubeConfig
	namespaceSelector *ui.NamespaceSelector
	contextSelector   *ui.ContextSelector
	err               error
}
type contextConnectionResultMsg struct {
	context          string
	namespaces       []string
	currentNamespace string
	err              error
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func connectToClusterCmd() tea.Cmd {
	return func() tea.Msg {
		// Initialize Kubernetes configuration
		kubeConfig, err := k8s.NewKubeConfig()
		if err != nil {
			return connectionResultMsg{err: fmt.Errorf("Failed to load kubeconfig: %v", err)}
		}

		// Get namespaces for current context (test connectivity)
		namespaces, nsErr := kubeConfig.GetNamespaces(kubeConfig.CurrentContext)
		if nsErr != nil {
			return connectionResultMsg{err: fmt.Errorf("Failed to connect to cluster: %v", nsErr)}
		}

		currentNamespace := kubeConfig.GetCurrentNamespace()
		namespaceSelector := ui.NewNamespaceSelector(namespaces, currentNamespace)
		contextSelector := ui.NewContextSelector(kubeConfig.Contexts, kubeConfig.CurrentContext)

		return connectionResultMsg{
			kubeConfig:        kubeConfig,
			namespaceSelector: namespaceSelector,
			contextSelector:   contextSelector,
			err:               nil,
		}
	}
}

func testContextConnectionCmd(kubeConfig *k8s.KubeConfig, context string) tea.Cmd {
	return func() tea.Msg {
		// Try to connect to the specified context
		namespaces, err := kubeConfig.GetNamespaces(context)
		if err != nil {
			return contextConnectionResultMsg{
				context: context,
				err:     err,
			}
		}

		currentNamespace := kubeConfig.GetCurrentNamespace()

		return contextConnectionResultMsg{
			context:          context,
			namespaces:       namespaces,
			currentNamespace: currentNamespace,
			err:              nil,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connectionResultMsg:
		m.isLoading = false

		if msg.err != nil {
			// Initial connection failed - show context selector with error
			m.kubeConfig, _ = k8s.NewKubeConfig() // Try to at least get kubeconfig structure

			if m.kubeConfig != nil && len(m.kubeConfig.Contexts) > 0 {
				// We have contexts, show selector
				m.contextSelector = ui.NewContextSelector(m.kubeConfig.Contexts, m.kubeConfig.CurrentContext)
				m.contextSelector.SetConnectionError("Failed to connect to current context: " + msg.err.Error())
				m.isConnected = false
			} else {
				// No kubeconfig at all
				m.initError = msg.err
			}
		} else {
			// Connection successful
			m.kubeConfig = msg.kubeConfig
			m.namespaceSelector = msg.namespaceSelector
			m.contextSelector = msg.contextSelector
			m.rightPane.SetKubeConfig(msg.kubeConfig)
			m.isConnected = true

			// Auto-select Overview and focus left pane when connecting
			m.leftPane.SelectedItem = "Overview"
			m.rightPane.SetSelectedItem(m.leftPane.SelectedItem)
			m.focusedPane = FocusLeftPane

			// If initial connection worked, close the context selector
			if m.contextSelector != nil {
				m.contextSelector.Close()
			}
		}
		return m, nil

	case contextConnectionResultMsg:
		// Handle context connection test result
		if m.contextSelector != nil {
			if msg.err != nil {
				// Connection failed - show error in selector
				m.contextSelector.SetConnectionError(msg.err.Error())
			} else {
				// Connection successful - update context and close selector
				m.kubeConfig.SwitchContext(msg.context)
				m.namespaceSelector.UpdateNamespaces(msg.namespaces, msg.currentNamespace)
				m.rightPane.SetKubeConfig(m.kubeConfig)
				m.contextSelector.Close()
				m.isConnected = true
				
				// Auto-select Overview and focus left pane when switching contexts
				m.leftPane.SelectedItem = "Overview"
				m.rightPane.SetSelectedItem(m.leftPane.SelectedItem)
				m.focusedPane = FocusLeftPane
				
				m.notifications.AddSuccess("Context switched", fmt.Sprintf("Now using context: %s", msg.context))
			}
		}
		return m, nil

	case tickMsg:
		// Clean up expired notifications on each tick
		if m.notifications != nil {
			m.notifications.CleanExpired()
		}

		// Update loading spinner if still loading
		if m.isLoading && m.loadingSpinner != nil {
			m.loadingSpinner.Update()
		}

		// Update context selector spinner if connecting
		if m.contextSelector != nil && m.contextSelector.IsOpen() {
			m.contextSelector.UpdateSpinner()
		}

		// Update nodes if nodes view is selected and we're connected
		if m.isConnected && m.rightPane != nil &&
			strings.Contains(strings.ToLower(m.leftPane.SelectedItem), "nodes") {
			m.rightPane.UpdateNodes()
		}

		// Update events if events view is selected and we're connected
		if m.isConnected && m.rightPane != nil &&
			strings.Contains(strings.ToLower(m.leftPane.SelectedItem), "events") {
			m.rightPane.UpdateEvents()
		}

		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rightPaneWidth = m.width - m.leftPaneWidth - 3

		// Adjust heights to account for footer (reserve 1 line for footer)
		paneHeight := m.height - 1
		m.leftPane.Height = paneHeight
		m.leftPane.Width = m.leftPaneWidth
		m.rightPane.Height = paneHeight
		m.rightPane.Width = m.rightPaneWidth
		m.footer.Width = m.width
		return m, nil

	case tea.KeyMsg:
		// Always allow quit, even during loading or error states
		if msg.String() == "ctrl+q" {
			return m, tea.Quit
		}

		// Skip other inputs if loading or error state
		if m.isLoading || m.initError != nil {
			return m, nil
		}

		// Handle context selector first if it's open
		if m.contextSelector != nil && m.contextSelector.IsOpen() {
			// Block all input if connecting
			if m.contextSelector.IsConnecting() {
				return m, nil
			}

			switch {
			case msg.Type == tea.KeyEscape:
				// Don't allow escape if not connected
				if !m.isConnected {
					return m, nil
				}
				m.contextSelector.Close()
			case msg.String() == "enter":
				m.contextSelector.Select()
				newContext := m.contextSelector.GetSelectedContext()

				// Clear any previous error
				m.contextSelector.ClearError()

				// Set connecting state
				m.contextSelector.SetConnecting(true)

				// Test connection to the selected context
				return m, testContextConnectionCmd(m.kubeConfig, newContext)

			case msg.String() == "up":
				m.contextSelector.MoveUp()
			case msg.String() == "down":
				m.contextSelector.MoveDown()
			case msg.Type == tea.KeyBackspace:
				if len(m.contextSelector.SearchQuery) > 0 {
					query := m.contextSelector.SearchQuery[:len(m.contextSelector.SearchQuery)-1]
					m.contextSelector.UpdateSearch(query)
				}
			default:
				if len(msg.String()) == 1 {
					query := m.contextSelector.SearchQuery + msg.String()
					m.contextSelector.UpdateSearch(query)
				}
			}
			return m, nil
		}

		// Handle timeframe input if it's open (highest priority after context selector)
		if m.timeframeInputPane != nil && m.timeframeInputPane.IsOpen() {
			switch {
			case msg.Type == tea.KeyEscape:
				m.timeframeInputPane.Close()
			case msg.String() == "enter":
				input := m.timeframeInputPane.GetInput()
				if input != "" {
					if eventsTable := m.rightPane.GetEventsTable(); eventsTable != nil {
						err := eventsTable.HandleTimeframeInput(input)
						if err != nil && m.notifications != nil {
							m.notifications.AddError("Invalid Input", err.Error())
						} else if m.notifications != nil {
							m.notifications.AddSuccess("Timeframe Updated",
								fmt.Sprintf("Now showing events from the past %s minutes", input))
						}
					}
				}
				m.timeframeInputPane.Close()
			case msg.Type == tea.KeyBackspace:
				m.timeframeInputPane.Backspace()
			default:
				if len(msg.String()) == 1 {
					m.timeframeInputPane.AddChar(msg.String())
				}
			}
			return m, nil
		}

		// Handle namespace selector if it's open
		if m.namespaceSelector != nil && m.namespaceSelector.IsOpen() {
			switch {
			case msg.Type == tea.KeyEscape:
				m.namespaceSelector.Close()
			case msg.String() == "enter":
				previousNamespace := m.namespaceSelector.GetSelectedNamespace()
				m.namespaceSelector.Select()
				newNamespace := m.namespaceSelector.GetSelectedNamespace()
				if newNamespace != previousNamespace && m.notifications != nil {
					m.notifications.AddInfo("Namespace changed", fmt.Sprintf("Now using namespace: %s", newNamespace))
				}
			case msg.String() == "up":
				m.namespaceSelector.MoveUp()
			case msg.String() == "down":
				m.namespaceSelector.MoveDown()
			case msg.Type == tea.KeyBackspace:
				if len(m.namespaceSelector.SearchQuery) > 0 {
					query := m.namespaceSelector.SearchQuery[:len(m.namespaceSelector.SearchQuery)-1]
					m.namespaceSelector.UpdateSearch(query)
				}
			default:
				if len(msg.String()) == 1 {
					query := m.namespaceSelector.SearchQuery + msg.String()
					m.namespaceSelector.UpdateSearch(query)
				}
			}
			return m, nil
		}

		if m.leftPane.SearchMode && m.focusedPane == FocusLeftPane {
			switch {
			case msg.String() == "1":
				m.focusedPane = FocusLeftPane
			case msg.String() == "2":
				m.focusedPane = FocusRightPane
			case msg.Type == tea.KeyEscape:
				m.leftPane.ToggleSearch()
				m.rightPane.SetSearchMode(m.leftPane.SearchMode)
			case msg.String() == "enter":
				resourceSelected := m.leftPane.ToggleExpand()
				m.rightPane.SetSelectedItem(m.leftPane.SelectedItem)
				// Exit search mode when selecting a resource item
				if resourceSelected {
					m.leftPane.ToggleSearch()
					m.rightPane.SetSearchMode(m.leftPane.SearchMode)
					// Auto-focus right pane when selecting a resource
					m.focusedPane = FocusRightPane
				}
			case msg.String() == "up":
				m.leftPane.MoveUp()
			case msg.String() == "down":
				m.leftPane.MoveDown()
			case msg.Type == tea.KeyBackspace:
				if len(m.leftPane.SearchQuery) > 0 {
					query := m.leftPane.SearchQuery[:len(m.leftPane.SearchQuery)-1]
					m.leftPane.UpdateSearch(query)
				}
			default:
				if len(msg.String()) == 1 && msg.String() != "1" && msg.String() != "2" {
					query := m.leftPane.SearchQuery + msg.String()
					m.leftPane.UpdateSearch(query)
				}
			}
		} else {
			switch msg.String() {
			case "1":
				m.focusedPane = FocusLeftPane
			case "2":
				m.focusedPane = FocusRightPane
			case "ctrl+n":
				if m.namespaceSelector != nil {
					m.namespaceSelector.Open()
				}
			case "ctrl+k":
				if m.contextSelector != nil {
					m.contextSelector.Open()
				}
			case "/":
				if m.focusedPane == FocusLeftPane {
					m.leftPane.ToggleSearch()
					m.rightPane.SetSearchMode(m.leftPane.SearchMode)
				}
			case "up":
				if m.focusedPane == FocusLeftPane {
					m.leftPane.MoveUp()
				}
			case "down":
				if m.focusedPane == FocusLeftPane {
					m.leftPane.MoveDown()
				}
			case "t":
				// Handle timeframe adjustment for events view
				if m.focusedPane == FocusRightPane && m.rightPane != nil &&
					strings.Contains(strings.ToLower(m.leftPane.SelectedItem), "events") {
					if m.timeframeInputPane != nil {
						m.timeframeInputPane.Open()
					}
				}
			case "enter":
				if m.focusedPane == FocusLeftPane {
					resourceSelected := m.leftPane.ToggleExpand()
					m.rightPane.SetSelectedItem(m.leftPane.SelectedItem)
					// Auto-focus right pane only when selecting a resource (not expanding folders)
					if resourceSelected {
						m.focusedPane = FocusRightPane
					}
				}
			}
			switch msg.Type {
			case tea.KeyEscape:
				if m.focusedPane == FocusLeftPane {
					m.leftPane.Collapse()
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Show loading spinner while connecting
	if m.isLoading && m.loadingSpinner != nil {
		return m.loadingSpinner.Render(m.width, m.height)
	}

	// Check for initialization error - show error with notifications overlay
	if m.initError != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196")).
			Width(m.width / 2)

		errorBox := lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			errorStyle.Render("Connection Error\n\n"+m.initError.Error()+"\n\nPress Ctrl+Q to quit"),
		)

		// Overlay notifications if any
		if m.notifications != nil && m.notifications.HasNotifications() {
			notificationsOverlay := m.notifications.Render(m.width, m.height)
			return lipgloss.Place(
				m.width,
				m.height,
				lipgloss.Left,
				lipgloss.Top,
				errorBox+notificationsOverlay,
			)
		}

		return errorBox
	}

	// Check if context selector is open (must be shown before main interface)
	if m.contextSelector != nil && m.contextSelector.IsOpen() {
		return m.contextSelector.Render(m.width, m.height)
	}

	// Must be connected to show main interface
	if !m.isConnected {
		// If we have a context selector but it's closed and we're not connected, reopen it
		if m.contextSelector != nil {
			m.contextSelector.Open()
			return m.contextSelector.Render(m.width, m.height)
		}
		return "Initializing..."
	}

	leftPaneContent := m.leftPane.Render()
	rightPaneContent := m.rightPane.Render()

	isNamespaceMode := m.namespaceSelector != nil && m.namespaceSelector.IsOpen()
	footerContent := m.footer.RenderWithMode(m.leftPane.SearchMode, isNamespaceMode)

	// Create context and namespace indicators
	contextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Background(lipgloss.Color("237")).
		Padding(0, 1).
		Bold(true)

	namespaceStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("237")).
		Padding(0, 1).
		Bold(true)

	contextIndicator := ""
	namespaceIndicator := ""

	if m.kubeConfig != nil {
		contextIndicator = contextStyle.Render("Context: " + m.kubeConfig.CurrentContext)
	}

	if m.namespaceSelector != nil {
		namespaceIndicator = namespaceStyle.Render("Namespace: " + m.namespaceSelector.GetSelectedNamespace())
	}

	// Create top bar with context and namespace indicators
	topBarStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	// Combine context and namespace indicators with spacing
	topBarContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		contextIndicator,
		"  ",
		namespaceIndicator,
	)

	topBar := topBarStyle.Render(topBarContent)

	// Adjust pane heights to account for footer and top bar
	paneHeight := m.height - 4 // -2 for borders, -1 for footer, -1 for top bar

	// Create focused and unfocused border styles
	focusedBorderStyle := styles.BorderStyle.
		BorderForeground(lipgloss.Color("39")) // Blue for focused pane

	unfocusedBorderStyle := styles.BorderStyle.
		BorderForeground(lipgloss.Color("240")) // Gray for unfocused pane

	// Apply appropriate border style based on focus
	var leftPaneStyled, rightPaneStyled string
	if m.focusedPane == FocusLeftPane {
		leftPaneStyled = focusedBorderStyle.
			Width(m.leftPaneWidth).
			Height(paneHeight).
			Render(leftPaneContent)
		rightPaneStyled = unfocusedBorderStyle.
			Width(m.rightPaneWidth).
			Height(paneHeight).
			Render(rightPaneContent)
	} else {
		leftPaneStyled = unfocusedBorderStyle.
			Width(m.leftPaneWidth).
			Height(paneHeight).
			Render(leftPaneContent)
		rightPaneStyled = focusedBorderStyle.
			Width(m.rightPaneWidth).
			Height(paneHeight).
			Render(rightPaneContent)
	}

	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPaneStyled,
		rightPaneStyled,
	)

	// Combine all UI elements
	fullUI := lipgloss.JoinVertical(
		lipgloss.Left,
		topBar,
		mainContent,
		footerContent,
	)

	// If namespace selector is open, overlay it
	if m.namespaceSelector != nil && m.namespaceSelector.IsOpen() {
		// Render the namespace selector as an overlay
		selectorOverlay := m.namespaceSelector.Render(m.width, m.height)

		// Place the selector over the main UI
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			selectorOverlay,
		)
	}

	// Check if timeframe input is open (highest priority overlay)
	if m.timeframeInputPane != nil && m.timeframeInputPane.IsOpen() {
		// Render the timeframe input as an overlay over the main UI
		timeframeOverlay := m.timeframeInputPane.Render(m.width, m.height)
		
		// Combine main UI with timeframe input overlay
		return m.renderWithTimeframeInput(fullUI, timeframeOverlay)
	}

	// Check if we have notifications to overlay
	if m.notifications != nil && m.notifications.HasNotifications() {
		// Create a custom overlay that combines main UI with notifications
		return m.renderWithNotifications(fullUI)
	}

	return fullUI
}

func (m Model) renderWithNotifications(mainUI string) string {
	// Get notification content without the full-screen placement
	m.notifications.CleanExpired()
	notifications := m.notifications.Notifications

	if len(notifications) == 0 {
		return mainUI
	}

	// Render notifications directly without full-screen placement
	var renderedNotifications []string
	maxVisible := 3
	visibleCount := len(notifications)
	if visibleCount > maxVisible {
		visibleCount = maxVisible
	}

	for i := 0; i < visibleCount; i++ {
		notif := notifications[i]
		rendered := m.renderSingleNotification(notif)
		renderedNotifications = append(renderedNotifications, rendered)
	}

	// Stack notifications vertically
	notificationStack := lipgloss.JoinVertical(
		lipgloss.Right,
		renderedNotifications...,
	)

	// Split the main UI into lines so we can overlay notifications
	mainUILines := strings.Split(mainUI, "\n")
	notificationLines := strings.Split(notificationStack, "\n")

	// Calculate where to place notifications (top-right)
	rightPadding := 2
	topPadding := 1

	// Overlay notifications on the top-right of main UI
	for i, notifLine := range notificationLines {
		if i+topPadding < len(mainUILines) {
			mainLine := mainUILines[i+topPadding]
			// Calculate position to place notification on the right
			availableWidth := m.width - len(notifLine) - rightPadding
			if availableWidth > 0 && len(mainLine) < availableWidth {
				// Pad the main line to make room for notification
				padding := strings.Repeat(" ", availableWidth-len(mainLine))
				mainUILines[i+topPadding] = mainLine + padding + notifLine
			} else if len(mainLine) >= availableWidth {
				// Truncate main line and add notification
				if availableWidth > 0 {
					mainUILines[i+topPadding] = mainLine[:availableWidth] + notifLine
				}
			}
		}
	}

	return strings.Join(mainUILines, "\n")
}

func (m Model) renderSingleNotification(notif ui.Notification) string {
	// Choose colors based on notification type
	var borderColor, iconColor lipgloss.Color
	var icon string

	switch notif.Type {
	case ui.NotificationError:
		borderColor = lipgloss.Color("196") // Red
		iconColor = lipgloss.Color("196")
		icon = "✗"
	case ui.NotificationWarning:
		borderColor = lipgloss.Color("214") // Orange
		iconColor = lipgloss.Color("214")
		icon = "⚠"
	case ui.NotificationSuccess:
		borderColor = lipgloss.Color("46") // Green
		iconColor = lipgloss.Color("46")
		icon = "✓"
	default: // Info
		borderColor = lipgloss.Color("39") // Blue
		iconColor = lipgloss.Color("39")
		icon = "ℹ"
	}

	// Create notification box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(60).
		Padding(0, 1).
		Background(lipgloss.Color("235"))

	// Icon style
	iconStyle := lipgloss.NewStyle().
		Foreground(iconColor).
		Bold(true)

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Bold(true)

	// Message style
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	// Time ago style
	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	// Format time ago
	timeAgo := m.formatTimeAgo(notif.Timestamp)

	// Build notification content
	var content strings.Builder

	// Title line with icon
	titleLine := fmt.Sprintf("%s %s",
		iconStyle.Render(icon),
		titleStyle.Render(notif.Title),
	)
	content.WriteString(titleLine)

	// Add time on the same line if there's space
	content.WriteString("  ")
	content.WriteString(timeStyle.Render(timeAgo))
	content.WriteString("\n")

	// Message
	if notif.Message != "" {
		// Wrap message text if it's too long
		wrapped := m.wrapText(notif.Message, 56)
		content.WriteString(messageStyle.Render(wrapped))
	}

	return boxStyle.Render(content.String())
}

func (m Model) formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Second {
		return "just now"
	} else if duration < time.Minute {
		seconds := int(duration.Seconds())
		return fmt.Sprintf("%ds ago", seconds)
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	}

	hours := int(duration.Hours())
	return fmt.Sprintf("%dh ago", hours)
}

func (m Model) renderWithTimeframeInput(mainUI, timeframeOverlay string) string {
	// The timeframe input overlay is already positioned with lipgloss.Place
	// We need to combine it with the main UI background
	mainUILines := strings.Split(mainUI, "\n")
	overlayLines := strings.Split(timeframeOverlay, "\n")
	
	// Create a result that preserves the main UI background with the overlay on top
	maxLines := len(mainUILines)
	if len(overlayLines) > maxLines {
		maxLines = len(overlayLines)
	}
	
	result := make([]string, maxLines)
	for i := 0; i < maxLines; i++ {
		if i < len(overlayLines) && strings.TrimSpace(overlayLines[i]) != "" {
			// Use overlay line if it has content
			result[i] = overlayLines[i]
		} else if i < len(mainUILines) {
			// Use main UI line as background
			result[i] = mainUILines[i]
		}
	}
	
	return strings.Join(result, "\n")
}

func (m Model) wrapText(text string, width int) string {
	words := strings.Fields(text)
	var lines []string
	var currentLine []string
	currentLength := 0

	for _, word := range words {
		wordLength := len(word)
		if currentLength > 0 && currentLength+wordLength+1 > width {
			lines = append(lines, strings.Join(currentLine, " "))
			currentLine = []string{word}
			currentLength = wordLength
		} else {
			currentLine = append(currentLine, word)
			if currentLength > 0 {
				currentLength += 1 // space
			}
			currentLength += wordLength
		}
	}

	if len(currentLine) > 0 {
		lines = append(lines, strings.Join(currentLine, " "))
	}

	return strings.Join(lines, "\n")
}
