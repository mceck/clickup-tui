package views

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Task struct {
	Title       string
	Description string
	Status      string
}

type HomeModel struct {
	tasks          []Task
	viewport       viewport.Model
	width          int
	height         int
	offsetX        int
	maxOffsetX     int
	columnWidth    int
	selectedColumn int
	selectedTask   int
	states         []string // Keep track of states order
	columns        map[string][]Task
}

func NewHomeModel() HomeModel {
	// Sample tasks
	tasks := []Task{
		{Title: "Task 1", Description: "Description 1", Status: "Todo"},
		{Title: "Task 2", Description: "Description 2", Status: "In Progress"},
		{Title: "Task 3", Description: "Description 3", Status: "Done"},
		{Title: "Task 3", Description: "Description 3", Status: "Done"},
		{Title: "Task 3", Description: "Description 3", Status: "Done"},
		{Title: "Task 3", Description: "Description 3", Status: "Deploy"},
		{Title: "Task 3", Description: "Description 3", Status: "Boh"},
		{Title: "Task 3", Description: "Description 3", Status: "Boh"},
		{Title: "Task 3", Description: "Description 3", Status: "Complete"},
		{Title: "Task 3", Description: "Description 3", Status: "Complete"},
		{Title: "Task 3", Description: "Description 3", Status: "Complete"},
		// Add more tasks...
	}
	statesMap := make(map[string]bool)
	for _, t := range tasks {
		statesMap[t.Status] = true
	}

	states := make([]string, 0, len(tasks))
	for state := range statesMap {
		states = append(states, state)
	}
	columns := make(map[string][]Task)
	for _, task := range tasks {
		columns[task.Status] = append(columns[task.Status], task)
	}

	return HomeModel{
		tasks:          tasks,
		columnWidth:    30,
		selectedColumn: 0,
		selectedTask:   0,
		states:         states,
		columns:        columns,
	}
}

func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = viewport.New(m.width, m.height-4)

		// Add spacing between columns and account for borders
		spacing := 2
		totalWidth := len(m.states)*(m.columnWidth+spacing) - spacing // subtract last spacing
		if totalWidth > m.width {
			m.maxOffsetX = totalWidth - m.width + 2 // +2 for edge cases
		} else {
			m.maxOffsetX = 0
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			if m.selectedColumn > 0 {
				m.selectedColumn--
				// Adjust scroll if needed
				if m.offsetX > m.selectedColumn*(m.columnWidth+2) {
					m.offsetX = m.selectedColumn * (m.columnWidth + 2)
				}
			}
		case "right":
			if m.selectedColumn < len(m.states)-1 {
				m.selectedColumn++
				// Adjust scroll if needed
				if (m.selectedColumn+1)*(m.columnWidth+2) > m.width+m.offsetX {
					m.offsetX = min(m.maxOffsetX, m.selectedColumn*(m.columnWidth+2))
				}
			}
		case "up":
			if m.selectedTask > 0 {
				m.selectedTask--
			}
		case "down":
			currentTasks := len(m.columns[m.states[m.selectedColumn]])
			if m.selectedTask < currentTasks-1 {
				m.selectedTask++
			}
		}
	}

	return m, nil
}

func (m HomeModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Stili per le colonne e i task
	columnStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1).
		Width(m.columnWidth)

	selectedColumnStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF00FF")). // Bright color for selected column
		Padding(1).
		Width(m.columnWidth)

	taskStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#5B4B8A")).
		Padding(0, 1)

	selectedTaskStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF00FF")). // Bright color for selected task
		Padding(0, 1)

	// Renderizza le colonne
	renderedColumns := make([]string, 0)
	for i, state := range m.states {
		tasks := m.columns[state]

		// Renderizza i task nella colonna
		renderedTasks := make([]string, 0)
		for j, task := range tasks {
			style := taskStyle
			if i == m.selectedColumn && j == m.selectedTask {
				style = selectedTaskStyle
			}

			taskStr := lipgloss.JoinVertical(lipgloss.Left,
				style.Copy().Bold(true).Render(task.Title),
				style.Copy().Faint(true).Render(task.Description),
			)
			renderedTasks = append(renderedTasks, taskStr)
		}

		// Use selected style for active column
		style := columnStyle
		if i == m.selectedColumn {
			style = selectedColumnStyle
		}

		column := style.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Bold(true).Render(state),
				lipgloss.JoinVertical(lipgloss.Left, renderedTasks...),
			),
		)
		renderedColumns = append(renderedColumns, column)
	}

	// Unisci tutte le colonne orizzontalmente con offset
	finalView := lipgloss.JoinHorizontal(lipgloss.Top, renderedColumns...)

	return lipgloss.JoinVertical(lipgloss.Left,
		"← →  colonne   ↑ ↓  task",
		finalView,
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
