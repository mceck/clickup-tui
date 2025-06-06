package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mceck/clickup-tui/internal/clients"
	"github.com/mceck/clickup-tui/internal/shared"
	ui "github.com/mceck/clickup-tui/internal/ui/styles"
	"golang.org/x/term"
)

type TimeEntryR struct {
	TaskId   string
	TaskName string
	Hours    map[string]float64 // map of day -> hours
}

type position struct {
	x      int
	y      int
	width  int
	height int
}

type TimesheetModel struct {
	width      int
	height     int
	wndwSize   int
	wndwOffset int
	timesheet  []TimeEntryR
	weekDays   []string
	weekFrom   time.Time
	cursor     struct {
		row int
		col int
	}
	editing       bool
	editBuffer    string
	firstEdit     bool
	cursorPos     int          // Add cursor position in text
	cellPositions [][]position // stores screen positions of each cell
	searchMode    bool
	searchQuery   string
	searched      []TimeEntryR
	loading       bool
	spinner       spinner.Model
}

type loadedTimesheetMsg struct {
	timesheet []TimeEntryR
	err       error
}

var DAYS = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

func SortTimesheetEntries(entries []TimeEntryR, week time.Time) []TimeEntryR {
	newEntries := make([]TimeEntryR, len(entries))
	copy(newEntries, entries)
	sort.Slice(newEntries, func(i, j int) bool {
		hoursI := 0.0
		for k, h := range newEntries[i].Hours {
			if week.AddDate(0, 0, 7).Format("2006-01-02") >= k && k >= week.Format("2006-01-02") {
				hoursI += h
			}
		}
		hoursJ := 0.0
		for k, h := range newEntries[j].Hours {
			if week.AddDate(0, 0, 7).Format("2006-01-02") >= k && k >= week.Format("2006-01-02") {
				hoursJ += h
			}
		}
		if hoursI > 0 && hoursJ == 0 {
			return true
		}
		if hoursI == 0 && hoursJ > 0 {
			return false
		}
		return newEntries[i].TaskName < newEntries[j].TaskName
	},
	)
	return newEntries
}

func FilterTs(entries []TimeEntryR, query string) []TimeEntryR {
	if len(query) > 0 {
		searched := make([]TimeEntryR, 0)
		for i := range entries {
			if strings.Contains(strings.ToLower(entries[i].TaskName), strings.ToLower(query)) {
				searched = append(searched, entries[i])
			}
		}
		return searched
	} else {
		return entries
	}
}

func calculateWindowSize(height int) int {
	availableHeight := height - 11
	wndwSize := availableHeight / 3

	if wndwSize < 1 {
		return 1
	}
	return wndwSize
}

func fetchTimesheetEntries() tea.Msg {
	config := clients.GetConfig()
	client := clients.NewClickupClient(config.ClickupToken, config.TeamId)
	userId := config.UserId

	tasks, err := client.GetTimesheetTasks()
	if err != nil {
		fmt.Println("Error fetching tasks:", err)
		return loadedTimesheetMsg{
			timesheet: nil,
			err:       err,
		}
	}
	trackings, err := client.GetTimesheetsEntries(userId)
	if err != nil {
		fmt.Println("Error fetching timesheets:", err)
		return loadedTimesheetMsg{
			timesheet: nil,
			err:       err,
		}
	}

	datats := make([]TimeEntryR, len(tasks))
	for i, task := range tasks {
		datats[i] = TimeEntryR{
			TaskId:   task.Id,
			TaskName: task.Name,
			Hours:    make(map[string]float64),
		}
		for _, tracking := range trackings {
			taskId, ok := tracking.Task.(map[string]interface{})
			if ok {
				taskIdStr, ok := taskId["id"].(string)
				if ok && taskIdStr == task.Id {
					day := shared.ToDateString(tracking.Start)
					// Convert duration to hours
					hours := shared.ToHours(tracking.Duration)
					datats[i].Hours[day] += hours // hours
				}
			}

		}
	}

	return loadedTimesheetMsg{
		timesheet: datats,
		err:       nil,
	}
}

func NewTimesheetModel() TimesheetModel {
	weekFrom := time.Now().AddDate(0, 0, -int(time.Now().Weekday())+1)
	// Get initial terminal size
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 120 // fallback width
		height = 24 // fallback height
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	model := TimesheetModel{
		width:         width,
		height:        height, // usa le dimensioni effettive del terminale
		wndwSize:      calculateWindowSize(height),
		wndwOffset:    0,
		timesheet:     []TimeEntryR{},
		weekFrom:      weekFrom,
		weekDays:      []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
		cellPositions: make([][]position, 0),
		firstEdit:     false,
		loading:       true,
		spinner:       s,
		cursorPos:     0,
	}
	model.cursor.row = 0
	model.cursor.col = int(time.Now().Weekday())
	return model
}

// Init inizializza il modello
func (m TimesheetModel) Init() tea.Cmd {
	if m.loading {
		return tea.Batch(fetchTimesheetEntries, m.spinner.Tick)
	}
	return nil
}

// Update aggiorna il modello in base ai messaggi ricevuti
func (m TimesheetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.loading {
		switch msg := msg.(type) {
		case loadedTimesheetMsg:
			if msg.err != nil {
				return m, tea.Quit
			}
			m.loading = false
			m.timesheet = SortTimesheetEntries(msg.timesheet, m.weekFrom)
			m.searched = FilterTs(m.timesheet, m.searchQuery)
			return m, nil
		case spinner.TickMsg:
			var spinnerCmd tea.Cmd
			m.spinner, spinnerCmd = m.spinner.Update(msg)
			return m, spinnerCmd
		}
	}

	switch msg := msg.(type) {
	case LoadMsg:
		return m, tea.Batch(fetchTimesheetEntries, m.spinner.Tick)
	case tea.KeyMsg:
		if msg.String() == "q" && !m.editing && !m.searchMode {
			return m, tea.Quit
		}
		if msg.String() == "r" && !m.editing && !m.searchMode {
			clients.ClearTimesheetTasksCache()
			clients.ClearTimeentriesCache()

			m.loading = true
			return m, tea.Batch(fetchTimesheetEntries, m.spinner.Tick)
		}
		switch msg.String() {
		case "esc":
			if m.editing {
				m.editing = false
				m.editBuffer = ""
				m.firstEdit = false
				m.cursorPos = 0
				return m, func() tea.Msg { return "stopPropagation" }
			}
			if m.searchMode {
				m.searchMode = false
				m.searchQuery = ""
				return m, func() tea.Msg { return "stopPropagation" }
			}
			return m, nil
		case "enter":
			if m.editing {
				// Try to parse and save the new value
				var newHours float64
				_, err := fmt.Sscanf(m.editBuffer, "%f", &newHours)
				if err == nil && newHours >= 0 {
					taskIdx := m.cursor.row
					day := m.weekFrom.AddDate(0, 0, m.cursor.col-1)
					dayKey := day.Format("2006-01-02")
					config := clients.GetConfig()
					client := clients.NewClickupClient(config.ClickupToken, config.TeamId)
					var taskId string
					if m.searchMode {
						taskId = m.searched[taskIdx].TaskId
					} else {
						taskId = m.timesheet[taskIdx].TaskId
					}
					err := client.UpdateTracking(config.UserId, taskId, day, newHours)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error updating tracking for task %s on day %s: %v\n", taskId, dayKey, err)
					} else {
						updated := false
						for i := range m.timesheet {
							if m.timesheet[i].TaskId == taskId {
								if m.timesheet[i].Hours == nil { // Ensure the map is initialized
									m.timesheet[i].Hours = make(map[string]float64)
								}
								m.timesheet[i].Hours[dayKey] = newHours
								updated = true
								break
							}
						}
						if updated {
							m.timesheet = SortTimesheetEntries(m.timesheet, m.weekFrom)
							m.searched = FilterTs(m.timesheet, m.searchQuery)
						} else {
							fmt.Fprintf(os.Stderr, "Error: TaskId %s not found in local timesheet after successful API update for day %s\n", taskId, dayKey)
						}
					}
				}
				m.editing = false
				m.editBuffer = ""
				m.firstEdit = false
				m.cursorPos = 0
				return m, nil
			} else if m.cursor.col > 0 { // Don't edit task name column
				m.editing = true
				m.firstEdit = true
				dayKey := m.weekFrom.AddDate(0, 0, m.cursor.col-1).Format("2006-01-02")
				if m.searchMode {
					m.editBuffer = fmt.Sprintf("%.2f", m.searched[m.cursor.row].Hours[dayKey])
				} else {
					m.editBuffer = fmt.Sprintf("%.2f", m.timesheet[m.cursor.row].Hours[dayKey])
				}
				m.cursorPos = len(m.editBuffer)
				return m, nil
			}
		case "left", "right":
			m.firstEdit = false
			if m.editing {
				if msg.String() == "left" && m.cursorPos > 0 {
					m.cursorPos--
					return m, nil
				} else if msg.String() == "right" && m.cursorPos < len(m.editBuffer) {
					m.cursorPos++
					return m, nil
				}
			}
			if !m.editing {
				if msg.String() == "left" {
					if m.cursor.col > 1 {
						m.cursor.col--
					} else {
						m.cursor.col = 5
						m.weekFrom = m.weekFrom.AddDate(0, 0, -7)
						m.timesheet = SortTimesheetEntries(m.timesheet, m.weekFrom)
					}
				} else if msg.String() == "right" {
					if m.cursor.col < len(m.weekDays) {
						m.cursor.col++
					} else {
						m.cursor.col = 1
						m.weekFrom = m.weekFrom.AddDate(0, 0, 7)
						m.timesheet = SortTimesheetEntries(m.timesheet, m.weekFrom)
					}
				}
			}
		case "up", "down":
			if !m.editing {
				if msg.String() == "up" && m.cursor.row > 0 {
					m.cursor.row--
					if m.cursor.row < m.wndwOffset {
						m.wndwOffset--
					}
				} else if msg.String() == "down" && m.cursor.row < len(m.timesheet)-1 {
					m.cursor.row++
					if m.cursor.row >= m.wndwOffset+m.wndwSize {
						m.wndwOffset++
					}
				}
			}
		case "backspace":
			if m.editing {
				if m.firstEdit {
					m.editBuffer = ""
					m.firstEdit = false
					m.cursorPos = 0
				} else if m.cursorPos > 0 {
					// Remove character before cursor
					m.editBuffer = m.editBuffer[:m.cursorPos-1] + m.editBuffer[m.cursorPos:]
					m.cursorPos--
				}
			}
			if m.searchMode && len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				m.cursor.row = 0
				m.wndwOffset = 0
				m.searched = FilterTs(m.timesheet, m.searchQuery)
			}
		case "ctrl+left":
			if !m.editing {
				m.weekFrom = m.weekFrom.AddDate(0, 0, -7)
				m.timesheet = SortTimesheetEntries(m.timesheet, m.weekFrom)
			}
		case "ctrl+right":
			if !m.editing {
				m.weekFrom = m.weekFrom.AddDate(0, 0, 7)
				m.timesheet = SortTimesheetEntries(m.timesheet, m.weekFrom)
			}
		case "/":
			if !m.editing {
				m.searchMode = !m.searchMode
				m.searchQuery = ""
				m.searched = m.timesheet
			}
		default:
			if !m.editing && m.searchMode {
				m.searchQuery += msg.String()
				m.cursor.row = 0
				m.wndwOffset = 0
				m.searched = FilterTs(m.timesheet, m.searchQuery)
			}
			if m.editing && len(msg.String()) == 1 {
				if m.firstEdit {
					m.editBuffer = msg.String()
					m.cursorPos = 1
					m.firstEdit = false
				} else {
					// Insert character at cursor position
					if m.cursorPos == len(m.editBuffer) {
						m.editBuffer += msg.String()
					} else {
						m.editBuffer = m.editBuffer[:m.cursorPos] + msg.String() + m.editBuffer[m.cursorPos:]
					}
					m.cursorPos++
				}
			}
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Recalculate cell positions
			m.cellPositions = make([][]position, m.wndwSize)
			for rowIdx := 0; rowIdx < m.wndwSize; rowIdx++ {
				m.cellPositions[rowIdx] = make([]position, len(m.weekDays)+1)
			}
			// Calculate cell positions

			// Calculate responsive column widths
			numColumns := len(m.weekDays) + 1 // +1 for task name column
			padding := 2                      // space between columns
			borderWidth := 0                  // left and right border of each cell
			totalBordersWidth := numColumns * borderWidth
			totalPaddingWidth := (numColumns - 1) * padding
			availableWidth := m.width - totalBordersWidth - totalPaddingWidth
			// Task name column gets more space
			taskColWidth := availableWidth * 4 / 9
			// Remaining width divided equally among day columns
			dayColWidth := availableWidth / 9
			// Calculate cell positions
			absoluteX := 0
			absoluteY := 6 // Start after title (1) + header (3) + totals row (3)
			for rowIdx := 0; rowIdx < m.wndwSize; rowIdx++ {
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
						width:  dayColWidth + borderWidth + 1,
						height: 3, // Aumentiamo l'altezza per includere i bordi
					}
					absoluteX += dayColWidth + borderWidth + padding
				}
				absoluteY += 3 // Incrementiamo di 1 per la prossima riga
			}

			// Find which cell was clicked
			for row := 0; row < m.wndwSize; row++ {
				for col := 1; col <= len(m.weekDays); col++ {
					cell := m.cellPositions[row][col]
					if msg.X >= cell.x && msg.X < cell.x+cell.width &&
						msg.Y >= cell.y && msg.Y < cell.y+cell.height {
						if m.cursor.row == row && m.cursor.col == col {
							// Toggle editing mode
							m.editing = !m.editing
							if m.editing {
								m.firstEdit = true
								dayKey := m.weekFrom.AddDate(0, 0, m.cursor.col-1).Format("2006-01-02")
								if m.searchMode {
									m.editBuffer = fmt.Sprintf("%.2f", m.searched[m.cursor.row+m.wndwOffset].Hours[dayKey])
								} else {
									m.editBuffer = fmt.Sprintf("%.2f", m.timesheet[m.cursor.row+m.wndwOffset].Hours[dayKey])
								}
								m.cursorPos = len(m.editBuffer)
							} else {
								m.editBuffer = ""
								m.firstEdit = false
								m.cursorPos = 0
							}
						} else {
							m.cursor.row = row + m.wndwOffset
							m.cursor.col = col
							m.editing = false
						}
						return m, nil
					}
				}
			}
			m.editing = false
			m.editBuffer = ""
		} else if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelUp {
			if !m.editing && m.cursor.row > 0 {
				m.cursor.row--
				if m.cursor.row < m.wndwOffset {
					m.wndwOffset--
				}
			}
		} else if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown {
			if !m.editing && m.cursor.row < len(m.timesheet)-1 {
				m.cursor.row++
				if m.cursor.row >= m.wndwOffset+m.wndwSize {
					m.wndwOffset++
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.wndwSize = calculateWindowSize(msg.Height)
		// Ensure wndwOffset is still valid with new window size
		if m.cursor.row >= m.wndwOffset+m.wndwSize {
			m.wndwOffset = m.cursor.row - m.wndwSize + 1
		}
	}
	return m, cmd
}

// View renderizza l'interfaccia utente
func (m TimesheetModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#874BFD")).
			MarginLeft(2)

		content := loadingStyle.Render("Caricamento timesheet... ") + m.spinner.View()

		return content
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Padding(0, 1)

	title := ui.TitleStyle.Render("Weekly Timesheet")
	title = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title)
	for i := 1; i < 6; i++ {
		m.weekDays[i-1] = m.weekFrom.AddDate(0, 0, i-1).Format("Mon 2")
	}

	// Calculate responsive column widths
	numColumns := len(m.weekDays) + 1 // +1 for task name column
	padding := 2                      // space between columns
	borderWidth := 0                  // left and right border of each cell
	totalBordersWidth := numColumns * borderWidth
	totalPaddingWidth := (numColumns - 1) * padding
	availableWidth := m.width - totalBordersWidth - totalPaddingWidth
	// Task name column gets more space
	taskColWidth := availableWidth * 4 / 9
	// Remaining width divided equally among day columns
	dayColWidth := availableWidth / 9.0
	// Calculate cell positions
	absoluteX := 0
	absoluteY := 7 // Start after title (1) + header (3) + totals row (3)

	// Styles
	headerStyle := ui.SubtitleStyle.Width(dayColWidth + 2).Align(lipgloss.Center)
	taskHeaderStyle := headerStyle.Width(taskColWidth + 2).Foreground(lipgloss.Color("212"))

	cellStyle := lipgloss.NewStyle().
		Width(dayColWidth).
		Align(lipgloss.Center).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	taskCellStyle := cellStyle.
		Inline(true).
		Width(taskColWidth).
		Align(lipgloss.Left)

	selectedStyle := cellStyle.
		BorderForeground(lipgloss.Color("86")).
		Bold(true)

	editingStyle := cellStyle.
		BorderForeground(lipgloss.Color("212")).
		Bold(true)

	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("55"))

	// Add a selected text style
	selectedTextStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("212")).
		Foreground(lipgloss.Color("0"))

	// Add cursor style
	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("212")).
		Foreground(lipgloss.Color("0"))

	// Create header row
	headerRow := []string{taskHeaderStyle.Render(m.weekFrom.Format("January 2006"))}
	for _, h := range m.weekDays {
		headerRow = append(headerRow, headerStyle.Render(h))
	}

	// Reset cell positions tracking
	m.cellPositions = make([][]position, m.wndwSize)

	// Create task rows
	var rows []string
	var wndw []TimeEntryR
	if m.searchMode {
		wndw = m.searched[m.wndwOffset:min(m.wndwOffset+m.wndwSize, len(m.searched))]
	} else {
		wndw = m.timesheet[m.wndwOffset:min(m.wndwOffset+m.wndwSize, len(m.timesheet))]
	}
	for rowIdx, entry := range wndw {
		row := []string{}
		m.cellPositions[rowIdx] = make([]position, len(m.weekDays)+1)
		absoluteX = 0

		// Task name column with ellipsis if too long
		taskName := entry.TaskName
		if m.searchMode && len(m.searchQuery) > 0 {
			lowerTask := strings.ToLower(taskName)
			lowerQuery := strings.ToLower(m.searchQuery)
			if idx := strings.Index(lowerTask, lowerQuery); idx >= 0 {
				before := taskName[:idx]
				match := taskName[idx : idx+len(m.searchQuery)]
				after := taskName[idx+len(m.searchQuery):]

				// Renderizza con l'highlight
				taskName = before + highlightStyle.Render(match) + after
				// Applica l'ellipsis solo se necessario
				fullLength := len(before) + len(match) + len(after)
				if fullLength > taskColWidth-8 && taskColWidth > 8 {
					hw := (taskColWidth - 8) / 2
					taskName = before[:min(len(before), hw)] + ".." + highlightStyle.Render(match[:min(len(match), 9)]) + ".." + after[max(0, len(after)-hw):]
				}

			}
		} else if len(taskName) > taskColWidth-4 {
			taskName = taskName[:taskColWidth-4] + "..."
		}

		row = append(row, taskCellStyle.Render(taskName))

		// Task name column position
		m.cellPositions[rowIdx][0] = position{
			x:      absoluteX,
			y:      absoluteY,
			width:  taskColWidth + borderWidth,
			height: 3, // Aumentiamo l'altezza per includere i bordi
		}
		absoluteX += taskColWidth + borderWidth + padding

		// Hour columns
		for colIdx := range m.weekDays {
			style := cellStyle
			content := "-"
			day := m.weekFrom.AddDate(0, 0, colIdx).Format("2006-01-02")

			if hours, exists := entry.Hours[day]; exists {
				content = fmt.Sprintf("%.1f", hours)
			}

			if rowIdx == m.cursor.row-m.wndwOffset && colIdx+1 == m.cursor.col {
				if m.editing {
					style = editingStyle
					content = m.editBuffer
					if m.firstEdit {
						content = selectedTextStyle.Render(content)
					} else {
						// Highlight the character at cursor position using background color
						if m.cursorPos < len(content) {
							// Highlight current character
							content = content[:m.cursorPos] + cursorStyle.Render(string(content[m.cursorPos])) + content[m.cursorPos+1:]
						} else {
							// Show a space at the end if cursor is at the end
							content = content + cursorStyle.Render(" ")
						}
					}
				} else {
					style = selectedStyle
				}
			}

			row = append(row, style.Render(content))

			// Hour columns positions
			m.cellPositions[rowIdx][colIdx+1] = position{
				x:      absoluteX,
				y:      absoluteY,
				width:  dayColWidth + borderWidth,
				height: 3, // Aumentiamo l'altezza per includere i bordi
			}
			absoluteX += dayColWidth + borderWidth + padding
		}

		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, row...))
		absoluteY += 3 // Incrementiamo di 1 per la prossima riga
	}

	// Calculate daily totals
	totals := make([]float64, len(m.weekDays))
	for _, entry := range m.timesheet {
		for i := range m.weekDays {
			day := m.weekFrom.AddDate(0, 0, i).Format("2006-01-02")
			totals[i] += entry.Hours[day]
		}
	}

	// Create totals row
	totalRow := []string{taskCellStyle.Foreground(lipgloss.Color("72")).Render("Total")}
	for _, total := range totals {
		totalCellStyle := cellStyle.Foreground(lipgloss.Color("208"))
		if total == 8 {
			totalCellStyle = totalCellStyle.Foreground(lipgloss.Color("72"))
		} else if total > 8 {
			totalCellStyle = totalCellStyle.Foreground(lipgloss.Color("134"))
		}
		totalRow = append(totalRow, totalCellStyle.Render(fmt.Sprintf("%.1f", total)))
	}

	// Join all parts
	table := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, headerRow...),
		lipgloss.JoinHorizontal(lipgloss.Center, totalRow...),
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)

	var help string
	if m.searchMode {
		help = "Search: " + m.searchQuery
	} else {
		help = "[← ↑ → ↓] navigate    [enter/click] Edit/Save    [/] Search    [esc] Tasks View"
	}

	// Calcola lo spazio disponibile e aggiungi padding per centrare verticalmente
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		table,
		"\n",
	)

	// Aggiungi spazi vuoti per spingere l'help in fondo
	contentHeight := strings.Count(content, "\n") + 1
	paddingHeight := m.height - contentHeight - 2 // -2 per l'help e un margine
	if paddingHeight > 0 {
		padding := strings.Repeat("\n", paddingHeight)
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			content,
			padding,
		)
	}

	// Aggiungi l'help in fondo
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		helpStyle.Render(help),
	)
}
