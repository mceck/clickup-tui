package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ui "github.com/mceck/clickup-tui/internal/ui/styles"
)

type TimeEntry struct {
	TaskName string
	Hours    map[string]float64 // map of day -> hours
}

type position struct {
	x      int
	y      int
	width  int
	height int
}

// HelpModel è il modello per la vista di aiuto
type HelpModel struct {
	width     int
	height    int
	timesheet []TimeEntry
	weekDays  []string
	cursor    struct {
		row int
		col int
	}
	editing       bool
	editBuffer    string
	cellPositions [][]position // stores positions of each cell
}

// NewHelpModel crea una nuova istanza del modello di aiuto
func NewHelpModel() HelpModel {
	// Mock data
	mockTimesheet := []TimeEntry{
		{
			TaskName: "Task 1",
			Hours: map[string]float64{
				"Mon": 2.5,
				"Tue": 3.0,
				"Wed": 4.0,
				"Thu": 2.0,
				"Fri": 1.5,
			},
		},
		{
			TaskName: "Task 2",
			Hours: map[string]float64{
				"Mon": 4.0,
				"Tue": 3.5,
				"Wed": 2.0,
				"Thu": 3.0,
				"Fri": 2.5,
			},
		},
		{
			TaskName: "Task 3",
			Hours: map[string]float64{
				"Mon": 1.0,
				"Wed": 3.0,
				"Thu": 4.0,
				"Fri": 3.0,
			},
		},
	}

	model := HelpModel{
		width:         80,
		height:        24,
		timesheet:     mockTimesheet,
		weekDays:      []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
		cellPositions: make([][]position, 0),
	}
	model.cursor.row = 0
	model.cursor.col = 1
	return model
}

// Init inizializza il modello
func (m HelpModel) Init() tea.Cmd {
	return nil
}

// Update aggiorna il modello in base ai messaggi ricevuti
func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.editing {
				m.editing = false
				m.editBuffer = ""
				return m, nil
			}
		case "enter":
			if m.editing {
				// Try to parse and save the new value
				var newHours float64
				_, err := fmt.Sscanf(m.editBuffer, "%f", &newHours)
				if err == nil && newHours >= 0 {
					taskIdx := m.cursor.row
					dayKey := m.weekDays[m.cursor.col-1] // -1 because first col is task name
					m.timesheet[taskIdx].Hours[dayKey] = newHours
				}
				m.editing = false
				m.editBuffer = ""
				return m, nil
			} else if m.cursor.col > 0 { // Don't edit task name column
				m.editing = true
				dayKey := m.weekDays[m.cursor.col-1]
				m.editBuffer = fmt.Sprintf("%.1f", m.timesheet[m.cursor.row].Hours[dayKey])
				return m, nil
			}
		case "up":
			if !m.editing && m.cursor.row > 0 {
				m.cursor.row--
			}
		case "down":
			if !m.editing && m.cursor.row < len(m.timesheet)-1 {
				m.cursor.row++
			}
		case "left":
			if !m.editing && m.cursor.col > 0 {
				m.cursor.col--
			}
		case "right":
			if !m.editing && m.cursor.col < len(m.weekDays) {
				m.cursor.col++
			}
		default:
			if m.editing {
				switch msg.String() {
				case "backspace":
					if len(m.editBuffer) > 0 {
						m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
					}
				default:
					// Allow only numbers and decimal point
					if (msg.String() >= "0" && msg.String() <= "9") || msg.String() == "." {
						m.editBuffer += msg.String()
					}
				}
			}
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress {
			// Recalculate cell positions
			m.cellPositions = make([][]position, len(m.timesheet))
			for rowIdx := range m.timesheet {
				m.cellPositions[rowIdx] = make([]position, len(m.weekDays)+1)
			}
			// Calculate cell positions

			// Calculate responsive column widths
			numColumns := len(m.weekDays) + 1 // +1 for task name column
			padding := 2                      // space between columns
			borderWidth := 2                  // left and right border of each cell
			totalBordersWidth := numColumns * borderWidth
			totalPaddingWidth := (numColumns - 1) * padding
			availableWidth := m.width - totalBordersWidth - totalPaddingWidth
			// Task name column gets more space
			taskColWidth := int(float64(availableWidth)*2.0/9.0) + 4
			// Remaining width divided equally among day columns
			dayColWidth := (availableWidth - taskColWidth) / (len(m.weekDays) + 2)
			// Calculate cell positions
			absoluteX := 0
			absoluteY := 3 // Start after title and header (title + newline + header)
			for rowIdx := range m.timesheet {
				absoluteX = 0
				// Task name column position
				m.cellPositions[rowIdx][0] = position{
					x:      absoluteX,
					y:      absoluteY,
					width:  taskColWidth + borderWidth,
					height: 3, // Aumentiamo l'altezza per includere i bordi
				}
				absoluteX += taskColWidth + borderWidth + padding
				// Hour columns positions
				for colIdx := range m.weekDays {
					m.cellPositions[rowIdx][colIdx+1] = position{
						x:      absoluteX,
						y:      absoluteY,
						width:  dayColWidth + borderWidth,
						height: 3, // Aumentiamo l'altezza per includere i bordi
					}
					absoluteX += dayColWidth + borderWidth + padding
				}
				absoluteY += 3 // Incrementiamo di 1 per la prossima riga
			}

			// Find which cell was clicked
			for row := range m.timesheet {
				for col := 1; col <= len(m.weekDays); col++ {
					cell := m.cellPositions[row][col]
					if msg.X >= cell.x && msg.X < cell.x+cell.width &&
						msg.Y >= cell.y && msg.Y < cell.y+cell.height {
						if m.cursor.row == row && m.cursor.col == col {
							// Toggle editing mode
							m.editing = !m.editing
							if m.editing {
								dayKey := m.weekDays[m.cursor.col-1]
								m.editBuffer = fmt.Sprintf("%.1f", m.timesheet[m.cursor.row].Hours[dayKey])
							} else {
								m.editBuffer = ""
							}
						} else {
							m.cursor.row = row
							m.cursor.col = col
							m.editing = false
						}
						return m, nil
					}
				}
			}
			m.editing = false
			m.editBuffer = ""
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renderizza l'interfaccia utente
func (m *HelpModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	title := ui.TitleStyle.Render("Weekly Timesheet")

	// Calculate responsive column widths
	numColumns := len(m.weekDays) + 1 // +1 for task name column
	padding := 2                      // space between columns
	borderWidth := 2                  // left and right border of each cell
	totalBordersWidth := numColumns * borderWidth
	totalPaddingWidth := (numColumns - 1) * padding
	availableWidth := m.width - totalBordersWidth - totalPaddingWidth

	// Task name column gets more space
	taskColWidth := int(float64(availableWidth) * 0.3)
	if taskColWidth > 30 {
		taskColWidth = 30 // max width for task names
	}

	// Remaining width divided equally among day columns
	dayColWidth := (availableWidth - taskColWidth) / len(m.weekDays)
	if dayColWidth < 8 {
		dayColWidth = 8 // minimum width for hour columns
	}

	// Styles
	headerStyle := ui.SubtitleStyle.Width(dayColWidth + 2).Align(lipgloss.Center)
	taskHeaderStyle := headerStyle.Width(taskColWidth + 2)

	cellStyle := lipgloss.NewStyle().
		Width(dayColWidth).
		Align(lipgloss.Center).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	taskCellStyle := cellStyle.
		Width(taskColWidth).
		Align(lipgloss.Left)

	selectedStyle := cellStyle.
		BorderForeground(lipgloss.Color("86")).
		Bold(true)

	selectedTaskStyle := taskCellStyle.
		BorderForeground(lipgloss.Color("86")).
		Bold(true)

	editingStyle := cellStyle.
		BorderForeground(lipgloss.Color("212")).
		Bold(true)

	// Create header row
	headerRow := []string{taskHeaderStyle.Render("Task")}
	for _, h := range m.weekDays {
		headerRow = append(headerRow, headerStyle.Render(h))
	}

	// Reset cell positions tracking
	m.cellPositions = make([][]position, len(m.timesheet))
	currentY := 3 // Start after title and header (title + newline + header)

	// Tracking absolute positions
	absoluteX := 0
	// headerHeight := 1

	// Create task rows
	var rows []string
	for rowIdx, entry := range m.timesheet {
		row := []string{}
		m.cellPositions[rowIdx] = make([]position, len(m.weekDays)+1)
		absoluteX = 0

		// Task name column with ellipsis if too long
		taskName := entry.TaskName
		if len(taskName) > taskColWidth-4 { // -4 for ellipsis and padding
			taskName = taskName[:taskColWidth-4] + "..."
		}

		style := taskCellStyle
		if rowIdx == m.cursor.row && m.cursor.col == 0 {
			style = selectedTaskStyle
		}
		row = append(row, style.Render(taskName))

		// Task name column position
		m.cellPositions[rowIdx][0] = position{
			x:      absoluteX,
			y:      currentY,
			width:  taskColWidth + borderWidth,
			height: 3, // Aumentiamo l'altezza per includere i bordi
		}
		absoluteX += taskColWidth + borderWidth + padding

		// Hour columns
		for colIdx, day := range m.weekDays {
			style := cellStyle
			content := "-"

			if hours, exists := entry.Hours[day]; exists {
				content = fmt.Sprintf("%.1f", hours)
			}

			if rowIdx == m.cursor.row && colIdx+1 == m.cursor.col {
				if m.editing {
					style = editingStyle
					content = m.editBuffer
				} else {
					style = selectedStyle
				}
			}

			row = append(row, style.Render(content))

			// Hour columns positions
			m.cellPositions[rowIdx][colIdx+1] = position{
				x:      absoluteX,
				y:      currentY,
				width:  dayColWidth + borderWidth,
				height: 3, // Aumentiamo l'altezza per includere i bordi
			}
			absoluteX += dayColWidth + borderWidth + padding
		}

		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, row...))
		currentY += 1 // Incrementiamo di 1 per la prossima riga
	}

	// Calculate daily totals
	totals := make([]float64, len(m.weekDays))
	for _, entry := range m.timesheet {
		for i, day := range m.weekDays {
			totals[i] += entry.Hours[day]
		}
	}

	// Create totals row
	totalRow := []string{taskCellStyle.Render("Total")}
	for _, total := range totals {
		totalRow = append(totalRow, cellStyle.Render(fmt.Sprintf("%.1f", total)))
	}

	// Join all parts
	table := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, headerRow...),
		lipgloss.JoinVertical(lipgloss.Left, rows...),
		lipgloss.JoinHorizontal(lipgloss.Center, totalRow...),
	)

	help := "← ↑ → ↓ to move • ENTER to edit • ESC to cancel • ENTER to save"
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		table,
		"\n",
		ui.InfoNotificationStyle.Render(help+" • Click to select and edit"),
	)
}
