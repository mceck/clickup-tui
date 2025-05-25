package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// Definizione dei colori
var (
	Subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	Highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	Special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	Error     = lipgloss.AdaptiveColor{Light: "#FF5F87", Dark: "#FF5F87"}
	Warning   = lipgloss.AdaptiveColor{Light: "#FFA500", Dark: "#FFA500"}
)

// Definizione degli stili
var (
	// Stili generali
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Titoli
	TitleStyle = lipgloss.NewStyle().
			Foreground(Highlight).
			Bold(true)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Special).
			Bold(true)

	// Menu
	ActiveMenuItem = lipgloss.NewStyle().
			Foreground(Highlight).
			Background(Subtle).
			Padding(0, 2)

	InactiveMenuItem = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Padding(0, 2)

	// Input
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(Subtle).
			Padding(0, 1)

	FocusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(Highlight).
				Padding(0, 1)

	// Contenitori
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Subtle).
			Padding(1, 2)

	// Pannello
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Highlight).
			Padding(1, 2)

	// Notifiche
	InfoNotificationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Italic(true)

	WarningNotificationStyle = lipgloss.NewStyle().
					Foreground(Warning).
					Italic(true)

	ErrorNotificationStyle = lipgloss.NewStyle().
				Foreground(Error).
				Italic(true)

	SuccessNotificationStyle = lipgloss.NewStyle().
					Foreground(Special).
					Italic(true)
)

// GetNotificationStyle restituisce lo stile appropriato per il tipo di notifica
func GetNotificationStyle(notificationType int) lipgloss.Style {
	switch notificationType {
	case 0: // Info
		return InfoNotificationStyle
	case 1: // Warning
		return WarningNotificationStyle
	case 2: // Error
		return ErrorNotificationStyle
	case 3: // Success
		return SuccessNotificationStyle
	default:
		return InfoNotificationStyle
	}
}
