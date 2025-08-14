package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationWarning
	NotificationError
	NotificationSuccess
)

type Notification struct {
	ID        string
	Type      NotificationType
	Title     string
	Message   string
	Timestamp time.Time
	Duration  time.Duration // How long to show the notification
}

type NotificationManager struct {
	Notifications []Notification // Export for access by RightPane
	maxVisible    int
	width         int
}

func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		Notifications: []Notification{},
		maxVisible:    3,
		width:         60,
	}
}

func (nm *NotificationManager) AddNotification(notifType NotificationType, title, message string) {
	notification := Notification{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      notifType,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  5 * time.Second, // Default 5 seconds
	}

	// Add to the beginning of the slice (newest first)
	nm.Notifications = append([]Notification{notification}, nm.Notifications...)

	// Limit the number of notifications
	if len(nm.Notifications) > 10 {
		nm.Notifications = nm.Notifications[:10]
	}
}

func (nm *NotificationManager) AddError(title, message string) {
	notification := Notification{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      NotificationError,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  7 * time.Second, // Error notifications last 7 seconds
	}

	// Add to the beginning of the slice (newest first)
	nm.Notifications = append([]Notification{notification}, nm.Notifications...)

	// Limit the number of notifications
	if len(nm.Notifications) > 10 {
		nm.Notifications = nm.Notifications[:10]
	}
}

func (nm *NotificationManager) AddWarning(title, message string) {
	nm.AddNotification(NotificationWarning, title, message)
}

func (nm *NotificationManager) AddInfo(title, message string) {
	nm.AddNotification(NotificationInfo, title, message)
}

func (nm *NotificationManager) AddSuccess(title, message string) {
	nm.AddNotification(NotificationSuccess, title, message)
}

func (nm *NotificationManager) CleanExpired() {
	now := time.Now()
	var active []Notification

	for _, notif := range nm.Notifications {
		if now.Sub(notif.Timestamp) < notif.Duration {
			active = append(active, notif)
		}
	}

	nm.Notifications = active
}

func (nm *NotificationManager) Clear() {
	nm.Notifications = []Notification{}
}

func (nm *NotificationManager) HasNotifications() bool {
	nm.CleanExpired()
	return len(nm.Notifications) > 0
}

func (nm *NotificationManager) Render(screenWidth, screenHeight int) string {
	nm.CleanExpired()

	if len(nm.Notifications) == 0 {
		return ""
	}

	var renderedNotifications []string
	visibleCount := min(len(nm.Notifications), nm.maxVisible)

	for i := 0; i < visibleCount; i++ {
		notif := nm.Notifications[i]
		rendered := nm.renderNotification(notif)
		renderedNotifications = append(renderedNotifications, rendered)
	}

	// Stack notifications vertically
	combined := lipgloss.JoinVertical(
		lipgloss.Right,
		renderedNotifications...,
	)

	// Place notifications in top-right corner
	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Right,
		lipgloss.Top,
		combined,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)
}

func (nm *NotificationManager) renderNotification(notif Notification) string {
	// Choose colors based on notification type
	var borderColor, iconColor lipgloss.Color
	var icon string

	switch notif.Type {
	case NotificationError:
		borderColor = lipgloss.Color("196") // Red
		iconColor = lipgloss.Color("196")
		icon = "✗"
	case NotificationWarning:
		borderColor = lipgloss.Color("214") // Orange
		iconColor = lipgloss.Color("214")
		icon = "⚠"
	case NotificationSuccess:
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
		Width(nm.width).
		Padding(0, 1).
		Background(lipgloss.Color("235")).
		MarginBottom(1)

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
	timeAgo := nm.formatTimeAgo(notif.Timestamp)

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
		wrapped := nm.wrapText(notif.Message, nm.width-4)
		content.WriteString(messageStyle.Render(wrapped))
	}

	return boxStyle.Render(content.String())
}

func (nm *NotificationManager) formatTimeAgo(t time.Time) string {
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

func (nm *NotificationManager) wrapText(text string, width int) string {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
