package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar creates a horizontal progress bar with percentage
func CreateProgressBar(current, total int64, width int, color string) string {
	if total == 0 {
		return strings.Repeat("─", width) + " 0%"
	}

	percentage := float64(current) / float64(total) * 100
	filledWidth := int(float64(width) * percentage / 100)

	// Create the bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("─", width-filledWidth)
	bar := filled + empty

	// Style the bar
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	percentText := fmt.Sprintf(" %.1f%%", percentage)

	return barStyle.Render(bar) + percentText
}

// CreateUsageBar creates a usage bar with labels
func CreateUsageBar(used, total int64, width int, label string, color string) string {
	if total == 0 {
		return fmt.Sprintf("%-12s │%s│ 0%% (0/0)", label, strings.Repeat("─", width))
	}

	percentage := float64(used) / float64(total) * 100
	filledWidth := int(float64(width) * percentage / 100)

	// Create the bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("─", width-filledWidth)
	bar := filled + empty

	// Style the bar based on usage level
	var barColor string
	switch {
	case percentage >= 90:
		barColor = "196" // Red
	case percentage >= 75:
		barColor = "214" // Orange
	case percentage >= 50:
		barColor = "226" // Yellow
	default:
		barColor = color // Default color (usually green)
	}

	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(12)
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(barColor))
	percentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	return fmt.Sprintf("%s │%s│ %s",
		labelStyle.Render(label),
		barStyle.Render(bar),
		percentStyle.Render(fmt.Sprintf("%.1f%% (%d/%d)", percentage, used, total)))
}

// CreateSimpleChart creates a simple horizontal bar chart
func CreateSimpleChart(data []ChartData, width int) string {
	if len(data) == 0 {
		return "No data available"
	}

	// Find the maximum value for scaling
	var maxValue int64
	for _, item := range data {
		if item.Value > maxValue {
			maxValue = item.Value
		}
	}

	if maxValue == 0 {
		maxValue = 1
	}

	var lines []string
	for _, item := range data {
		// Calculate bar width
		barWidth := int(float64(width) * float64(item.Value) / float64(maxValue))
		if barWidth == 0 && item.Value > 0 {
			barWidth = 1 // Show at least one character for non-zero values
		}

		// Create the bar
		bar := strings.Repeat("█", barWidth)
		if barWidth < width {
			bar += strings.Repeat("─", width-barWidth)
		}

		// Style the bar
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(item.Color))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(15)

		line := fmt.Sprintf("%s │%s│ %d",
			labelStyle.Render(item.Label),
			barStyle.Render(bar),
			item.Value)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// CreateDistributionChart creates a pie-chart-like distribution using text
func CreateDistributionChart(data []ChartData, width int) string {
	if len(data) == 0 {
		return "No data available"
	}

	// Calculate total
	var total int64
	for _, item := range data {
		total += item.Value
	}

	if total == 0 {
		return "No data available"
	}

	var lines []string
	for _, item := range data {
		percentage := float64(item.Value) / float64(total) * 100

		// Create visual representation
		barWidth := int(float64(width) * percentage / 100)
		if barWidth == 0 && item.Value > 0 {
			barWidth = 1
		}

		bar := strings.Repeat("█", barWidth)
		if barWidth < width {
			bar += strings.Repeat("─", width-barWidth)
		}

		// Style
		barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(item.Color))
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(12)
		valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

		line := fmt.Sprintf("%s │%s│ %s",
			labelStyle.Render(item.Label),
			barStyle.Render(bar),
			valueStyle.Render(fmt.Sprintf("%.1f%% (%d)", percentage, item.Value)))
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// ChartData represents data for charts
type ChartData struct {
	Label string
	Value int64
	Color string
}

// CreateSparkline creates a mini sparkline chart
func CreateSparkline(values []int64, width int) string {
	if len(values) == 0 || width == 0 {
		return strings.Repeat("─", width)
	}

	// Find min and max for scaling
	var minVal, maxVal int64
	minVal = values[0]
	maxVal = values[0]

	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// If all values are the same, show a flat line
	if maxVal == minVal {
		return strings.Repeat("─", width)
	}

	// Sparkline characters (from bottom to top)
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	var result strings.Builder
	step := len(values) / width
	if step == 0 {
		step = 1
	}

	for i := 0; i < width && i*step < len(values); i++ {
		value := values[i*step]

		// Normalize to 0-7 range for character selection
		normalized := int(float64(value-minVal) / float64(maxVal-minVal) * 7)
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 7 {
			normalized = 7
		}

		result.WriteString(chars[normalized])
	}

	return result.String()
}
