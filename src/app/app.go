package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"peek/src/styles"
	"peek/src/ui"
)

type Model struct {
	leftPane       *ui.LeftPane
	rightPane      *ui.RightPane
	width          int
	height         int
	leftPaneWidth  int
	rightPaneWidth int
}

func InitialModel() Model {
	leftPaneWidth := 35
	leftPane := ui.NewLeftPane(leftPaneWidth, 24)
	rightPane := ui.NewRightPane(45, 24)

	return Model{
		leftPane:      leftPane,
		rightPane:     rightPane,
		leftPaneWidth: leftPaneWidth,
		width:         80,
		height:        24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rightPaneWidth = m.width - m.leftPaneWidth - 3
		m.leftPane.Height = m.height
		m.leftPane.Width = m.leftPaneWidth
		m.rightPane.Height = m.height
		m.rightPane.Width = m.rightPaneWidth
		return m, nil

	case tea.KeyMsg:
		if m.leftPane.SearchMode {
			switch {
			case msg.String() == "ctrl+c" || msg.String() == "q":
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
			case msg.String() == "up" || msg.String() == "k":
				m.leftPane.MoveUp()
			case msg.String() == "down" || msg.String() == "j":
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
			case "ctrl+c", "q":
				return m, tea.Quit
			case "/":
				m.leftPane.ToggleSearch()
				m.rightPane.SetSearchMode(m.leftPane.SearchMode)
			case "up", "k":
				m.leftPane.MoveUp()
			case "down", "j":
				m.leftPane.MoveDown()
			case "enter", " ", "right", "l":
				m.leftPane.ToggleExpand()
				m.rightPane.SetSelectedItem(m.leftPane.SelectedItem)
			case "left", "h":
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

	leftPaneContent := m.leftPane.Render()
	rightPaneContent := m.rightPane.Render()

	leftPaneStyled := styles.BorderStyle.
		Width(m.leftPaneWidth).
		Height(m.height - 2).
		Render(leftPaneContent)

	rightPaneStyled := styles.BorderStyle.
		Width(m.rightPaneWidth).
		Height(m.height - 2).
		Render(rightPaneContent)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftPaneStyled,
		rightPaneStyled,
	)
}