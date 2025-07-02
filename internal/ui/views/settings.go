package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mceck/clickup-tui/internal/clients"
	ui "github.com/mceck/clickup-tui/internal/ui/styles"
)

type SettingsModel struct {
	token           textinput.Model
	teamId          textinput.Model
	viewId          textinput.Model
	timesheetFilter textinput.Model // New field for timesheet filters
	initialView     string          // New field for the initial view ('kanban' or 'timesheet')

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

	viewId := textinput.New()
	viewId.Placeholder = "Inserisci la view ID"
	viewId.CharLimit = 32
	viewId.Width = 60
	viewId.SetValue(config.ViewId)

	timesheetFilter := textinput.New()
	timesheetFilter.Placeholder = "e.g., tags[]=timesheet&assignees[]=123456"
	timesheetFilter.CharLimit = 200
	timesheetFilter.Width = 60
	timesheetFilter.SetValue(config.TimesheetFilter)

	inputs := []textinput.Model{token, teamId, viewId, timesheetFilter}

	initialView := config.InitialView
	if initialView != "kanban" && initialView != "timesheet" {
		initialView = "kanban"
	}

	m := SettingsModel{
		token:           token,
		teamId:          teamId,
		viewId:          viewId,
		timesheetFilter: timesheetFilter,
		initialView:     initialView,
		inputs:          inputs,
		focusIndex:      0,
		width:           100,
		height:          30,
	}

	if len(m.inputs) > 0 {
		m.inputs[m.focusIndex].Focus()
	}

	return m
}

func (m SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, tea.Batch(func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyTab}
			})
		case "enter":
			token := m.token.Value()
			if token == "" {
				return m, nil
			}
			config := clients.GetConfig()
			client := clients.NewClickupClient(config.ClickupToken, config.TeamId)

			teamId := m.teamId.Value()
			if teamId == "" {
				teams, err := client.GetTeams(token)
				if err != nil || len(teams) == 0 {
					return m, nil
				}
				teamId = teams[0].Id
				m.inputs[1].SetValue(teamId)
				if len(teams) > 1 {
					return m, nil
				}
			}
			user, err := client.GetCurrentUser(token)
			if err != nil {
				return m, nil
			}

			userId := ""
			switch v := user.Id.(type) {
			case float64:
				userId = fmt.Sprintf("%.0f", v)
			case string:
				userId = v
			default:
				return m, nil
			}

			c := clients.Config{
				ClickupToken:    token,
				TeamId:          teamId,
				UserId:          userId,
				ViewId:          m.viewId.Value(),
				TimesheetFilter: m.timesheetFilter.Value(),
				InitialView:     m.initialView,
			}
			clients.SaveConfig(c)
			return m, tea.Quit

		case "up", "down":
			if msg.String() == "up" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			totalItems := len(m.inputs) + 1
			if m.focusIndex >= totalItems {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = totalItems - 1
			}

			cmds = append(cmds, m.updateFocus()...)
			return m, tea.Batch(cmds...)

		case " ", "left", "right":
			if m.focusIndex == len(m.inputs) {
				if m.initialView == "kanban" {
					m.initialView = "timesheet"
				} else {
					m.initialView = "kanban"
				}
			}
		}
	}

	cmds = append(cmds, m.processInputs(msg)...)
	return m, tea.Batch(cmds...)
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
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(msg)
		cmds[i] = cmd
	}
	m.syncFields()
	return cmds
}

func (m *SettingsModel) syncFields() {
	m.token = m.inputs[0]
	m.teamId = m.inputs[1]
	m.viewId = m.inputs[2]
	m.timesheetFilter = m.inputs[3]
}

func (m SettingsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := ui.TitleStyle.Render("Impostazioni ClickUp")
	labelWidth := 20 // Increased width for better alignment

	var inputRows []string
	for i, input := range m.inputs {
		label := ui.SubtitleStyle.Width(labelWidth).Render(m.getLabel(i) + ":")
		inputField := input.View()
		inputRows = append(inputRows, lipgloss.JoinHorizontal(lipgloss.Left, label, inputField))
	}

	radioLabel := ui.SubtitleStyle.Width(labelWidth).Render(m.getLabel(len(m.inputs)) + ":")

	kanbanChoice := "( ) Kanban"
	timesheetChoice := "( ) Timesheet"
	if m.initialView == "kanban" {
		kanbanChoice = "(•) Kanban"
	} else {
		timesheetChoice = "(•) Timesheet"
	}

	var radioView string
	if m.focusIndex == len(m.inputs) {
		focusedStyle := lipgloss.NewStyle().Foreground(ui.Highlight)
		radioView = focusedStyle.Render(kanbanChoice + "    " + timesheetChoice)
	} else {
		blurredStyle := lipgloss.NewStyle().Foreground(ui.Subtle)
		radioView = blurredStyle.Render(kanbanChoice + "    " + timesheetChoice)
	}
	radioRow := lipgloss.JoinHorizontal(lipgloss.Left, radioLabel, radioView)

	formContent := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinVertical(lipgloss.Left, inputRows...),
		"\n",
		radioRow,
	)
	formBox := ui.PanelStyle.Width(m.width - 4).Render(formContent)
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("[↑ ← → ↓] Move      [enter] Save and quit     [esc/tab] Go back")

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
	labels := []string{"Token", "Team ID", "View ID", "Timesheet Filter", "Initial View"}
	if index >= 0 && index < len(labels) {
		return labels[index]
	}
	return ""
}
