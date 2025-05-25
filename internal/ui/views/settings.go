package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mceck/clickup-tui/internal/clients"
	ui "github.com/mceck/clickup-tui/internal/ui/styles"
)

type SettingsModel struct {
	token      textinput.Model
	teamId     textinput.Model
	userId     textinput.Model
	viewId     textinput.Model
	focusIndex int
	inputs     []textinput.Model
	width      int
	height     int
}

func NewSettingsModel() SettingsModel {
	config := clients.GetConfig()
	token := textinput.New()
	token.Placeholder = "Inserisci il tuo token ClickUp"
	token.CharLimit = 150
	token.Width = 60
	token.SetValue(config.ClickupToken)

	teamId := textinput.New()
	teamId.Placeholder = "Inserisci il team ID"
	teamId.CharLimit = 32
	teamId.Width = 60
	teamId.SetValue(config.TeamId)

	userId := textinput.New()
	userId.Placeholder = "Inserisci il tuo user ID"
	userId.CharLimit = 32
	userId.Width = 60
	userId.SetValue(config.UserId)

	viewId := textinput.New()
	viewId.Placeholder = "Inserisci la view ID"
	viewId.CharLimit = 32
	viewId.Width = 60
	viewId.SetValue(config.ViewId)

	inputs := []textinput.Model{token, teamId, userId, viewId}

	m := SettingsModel{
		token:      token,
		teamId:     teamId,
		userId:     userId,
		viewId:     viewId,
		inputs:     inputs,
		focusIndex: 0,
		width:      100,
		height:     30,
	}

	// Imposta il focus iniziale
	m.inputs[m.focusIndex].Focus()
	return m
}

func (m SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			c := clients.Config{
				ClickupToken: m.token.Value(),
				TeamId:       m.teamId.Value(),
				UserId:       m.userId.Value(),
				ViewId:       m.viewId.Value(),
			}
			clients.SaveConfig(c)
			return m, tea.Quit
		case "down":
			m.focusIndex++
		case "up":
			m.focusIndex--
		}

		if m.focusIndex >= len(m.inputs) {
			m.focusIndex = 0
		} else if m.focusIndex < 0 {
			m.focusIndex = len(m.inputs) - 1
		}

		// Aggiorna il focus
		m.updateFocus()
	}

	// Processa gli input
	m.processInputs(msg)

	return m, nil
}

func (m *SettingsModel) updateFocus() []tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	m.syncFields()
	return cmds
}

func (m *SettingsModel) processInputs(msg tea.Msg) []tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	m.syncFields()
	return cmds
}

func (m *SettingsModel) syncFields() {
	m.token = m.inputs[0]
	m.teamId = m.inputs[1]
	m.userId = m.inputs[2]
}

func (m SettingsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := ui.TitleStyle.Render("Impostazioni ClickUp")
	labelWidth := 15

	inputRows := make([]string, len(m.inputs))
	for i, input := range m.inputs {
		label := ui.SubtitleStyle.Width(labelWidth).Render(m.getLabel(i) + ":")
		inputField := input.View()
		inputRows[i] = lipgloss.JoinHorizontal(lipgloss.Left, label, inputField)
	}

	formContent := lipgloss.JoinVertical(lipgloss.Left, inputRows...)
	formBox := ui.PanelStyle.Width(m.width - 4).Render(formContent)
	footer := ui.InfoNotificationStyle.Render("[↑ ↓] move      [enter] save")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"\n",
		formBox,
		"\n",
		footer,
	)
}

func (m SettingsModel) getLabel(index int) string {
	labels := []string{"Token", "Team ID", "User ID", "View ID"}
	if index < len(labels) {
		return labels[index]
	}
	return ""
}
