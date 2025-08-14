package app

import (
	"fmt"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"peek/src/k8s"
	"peek/src/styles"
	"peek/src/ui"
)

type Model struct {
	leftPane          *ui.LeftPane
	rightPane         *ui.RightPane
	footer            *ui.Footer
	namespaceSelector *ui.NamespaceSelector
	contextSelector   *ui.ContextSelector
	notifications     *ui.NotificationManager
	kubeConfig        *k8s.KubeConfig
	width             int
	height            int
	leftPaneWidth     int
	rightPaneWidth    int
	initError         error
}

func InitialModel() Model {
	leftPaneWidth := 35
	leftPane := ui.NewLeftPane(leftPaneWidth, 24)
	rightPane := ui.NewRightPane(45, 24)
	footer := ui.NewFooter(80)
	notifications := ui.NewNotificationManager()
	
	// Connect notifications to right pane
	rightPane.SetNotifications(notifications)
	
	// Initialize Kubernetes configuration
	kubeConfig, err := k8s.NewKubeConfig()
	if err != nil {
		// Create a minimal model with error
		rightPane.SetNotifications(notifications)
		return Model{
			leftPane:      leftPane,
			rightPane:     rightPane,
			footer:        footer,
			notifications: notifications,
			leftPaneWidth: leftPaneWidth,
			width:         80,
			height:        24,
			initError:     fmt.Errorf("Failed to load kubeconfig: %v", err),
		}
	}
	
	// Get namespaces for current context
	namespaces, nsErr := kubeConfig.GetNamespaces(kubeConfig.CurrentContext)
	if nsErr != nil {
		// Add error notification but continue
		notifications.AddError("Connection Error", nsErr.Error())
	}
	
	currentNamespace := kubeConfig.GetCurrentNamespace()
	
	namespaceSelector := ui.NewNamespaceSelector(namespaces, currentNamespace)
	contextSelector := ui.NewContextSelector(kubeConfig.Contexts, kubeConfig.CurrentContext)

	// Connect kubeconfig to right pane for metrics
	rightPane.SetKubeConfig(kubeConfig)

	return Model{
		leftPane:          leftPane,
		rightPane:         rightPane,
		footer:            footer,
		namespaceSelector: namespaceSelector,
		contextSelector:   contextSelector,
		notifications:     notifications,
		kubeConfig:        kubeConfig,
		leftPaneWidth:     leftPaneWidth,
		width:             80,
		height:            24,
	}
}

func (m Model) Init() tea.Cmd {
	// Start a ticker to refresh notifications
	return tickCmd()
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Clean up expired notifications on each tick
		if m.notifications != nil {
			m.notifications.CleanExpired()
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
		// Handle context selector first if it's open
		if m.contextSelector != nil && m.contextSelector.IsOpen() {
			switch {
			case msg.String() == "ctrl+q":
				return m, tea.Quit
			case msg.Type == tea.KeyEscape:
				m.contextSelector.Close()
			case msg.String() == "enter":
				previousContext := m.kubeConfig.CurrentContext
				m.contextSelector.Select()
				newContext := m.contextSelector.GetSelectedContext()
				
				// If context changed, update namespaces
				if newContext != previousContext {
					m.kubeConfig.SwitchContext(newContext)
					// Reload namespaces for new context
					namespaces, nsErr := m.kubeConfig.GetNamespaces(newContext)
					if nsErr != nil {
						// Show error notification
						if m.notifications != nil {
							m.notifications.AddError("Failed to connect to cluster", nsErr.Error())
						}
						// Force context selector to reopen for user to select another cluster
						m.contextSelector.Open()
					} else {
						// Show success notification
						if m.notifications != nil {
							m.notifications.AddSuccess("Context switched", fmt.Sprintf("Now using context: %s", newContext))
						}
						// Update right pane with new kubeconfig
						if m.rightPane != nil {
							m.rightPane.SetKubeConfig(m.kubeConfig)
						}
					}
					currentNamespace := m.kubeConfig.GetCurrentNamespace()
					m.namespaceSelector.UpdateNamespaces(namespaces, currentNamespace)
				}
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
		
		// Handle namespace selector if it's open
		if m.namespaceSelector != nil && m.namespaceSelector.IsOpen() {
			switch {
			case msg.String() == "ctrl+q":
				return m, tea.Quit
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
		
		if m.leftPane.SearchMode {
			switch {
			case msg.String() == "ctrl+q":
				return m, tea.Quit
			case msg.Type == tea.KeyEscape:
				m.leftPane.ToggleSearch()
				m.rightPane.SetSearchMode(m.leftPane.SearchMode)
			case msg.String() == "enter":
				m.leftPane.ToggleExpand()
				m.rightPane.SetSelectedItem(m.leftPane.SelectedItem)
				// Exit search mode when selecting a resource item
				if m.leftPane.SelectedItem != "" {
					m.leftPane.ToggleSearch()
					m.rightPane.SetSearchMode(m.leftPane.SearchMode)
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
				if len(msg.String()) == 1 {
					query := m.leftPane.SearchQuery + msg.String()
					m.leftPane.UpdateSearch(query)
				}
			}
		} else {
			switch msg.String() {
			case "ctrl+q":
				return m, tea.Quit
			case "ctrl+n":
				if m.namespaceSelector != nil {
					m.namespaceSelector.Open()
				}
			case "ctrl+k":
				if m.contextSelector != nil {
					m.contextSelector.Open()
				}
			case "/":
				m.leftPane.ToggleSearch()
				m.rightPane.SetSearchMode(m.leftPane.SearchMode)
			case "up":
				m.leftPane.MoveUp()
			case "down":
				m.leftPane.MoveDown()
			case "enter":
				m.leftPane.ToggleExpand()
				m.rightPane.SetSelectedItem(m.leftPane.SelectedItem)
			}
			switch msg.Type {
			case tea.KeyEscape:
				m.leftPane.Collapse()
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Check for initialization error
	if m.initError != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Padding(2)
		return errorStyle.Render(m.initError.Error())
	}

	// Check if context selector is open (startup or manual)
	if m.contextSelector != nil && m.contextSelector.IsOpen() {
		return m.contextSelector.Render(m.width, m.height)
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

	leftPaneStyled := styles.BorderStyle.
		Width(m.leftPaneWidth).
		Height(paneHeight).
		Render(leftPaneContent)

	rightPaneStyled := styles.BorderStyle.
		Width(m.rightPaneWidth).
		Height(paneHeight).
		Render(rightPaneContent)

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

	return fullUI
}