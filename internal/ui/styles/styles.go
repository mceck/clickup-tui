package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	Subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	Highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	Special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	Error     = lipgloss.AdaptiveColor{Light: "#FF5F87", Dark: "#FF5F87"}
	Warning   = lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFA500"}
)

var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(Highlight).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Special).
			Bold(true)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Highlight).
			Padding(1, 2)
)
