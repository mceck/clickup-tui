package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mceck/clickup-tui/internal/clients"
	"github.com/mceck/clickup-tui/internal/ui/views"
)

type Page int

const (
	HomeView Page = iota
	SettingsView
	TimesheetView
)

type AppModel struct {
	currentPage Page
	routes      map[Page]tea.Model
	width       int
	height      int
}

func (m AppModel) getCurrentRoute() tea.Model {
	route := m.routes[m.currentPage]
	if route == nil {
		switch m.currentPage {
		case HomeView:
			m.routes[m.currentPage] = views.NewHomeModel()
		case SettingsView:
			m.routes[m.currentPage] = views.NewSettingsModel()
		case TimesheetView:
			m.routes[m.currentPage] = views.NewTimesheetModel()
		}
	}
	return m.routes[m.currentPage]
}

func New() AppModel {
	config := clients.GetConfig()
	var initialPage Page
	if config.InitialView == "timesheet" {
		initialPage = TimesheetView
	} else {
		initialPage = HomeView
	}
	return AppModel{
		currentPage: initialPage,
		routes:      map[Page]tea.Model{},
	}
}

func (m AppModel) Init() tea.Cmd {
	config := clients.GetConfig()
	if config.ClickupToken == "" || config.TeamId == "" {
		m.currentPage = SettingsView
	}
	return m.getCurrentRoute().Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.routes[m.currentPage], cmd = m.getCurrentRoute().Update(msg)
	if cmd != nil {
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.currentPage == HomeView {
				m.currentPage = TimesheetView
			} else {
				m.currentPage = HomeView
			}
			if m.routes[m.currentPage] == nil {
				m.routes[m.currentPage] = m.getCurrentRoute()
				m.routes[m.currentPage], cmd = m.routes[m.currentPage].Update(views.LoadMsg{})
			}
		case "?":
			m.currentPage = SettingsView
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	config := clients.GetConfig()
	if config.ClickupToken == "" {
		m.currentPage = SettingsView
	}

	return m, cmd
}

func (m AppModel) View() string {
	route := m.routes[m.currentPage]
	if route == nil {
		return ""
	}
	return route.View()
}

func NewProgram() *tea.Program {
	model := New()
	return tea.NewProgram(model, tea.WithMouseCellMotion(), tea.WithAltScreen())
}
