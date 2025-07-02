package views

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

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
	Hours    map[string]float64
}

type loadedTimesheetMsg struct {
	timesheet []TimeEntryR
	err       error
}

type position struct {
	x, y, width, height int
}

type tsStyles struct {
	headerStyle       lipgloss.Style
	taskHeaderStyle   lipgloss.Style
	cellStyle         lipgloss.Style
	taskCellStyle     lipgloss.Style
	selectedRowStyle  lipgloss.Style
	selectedStyle     lipgloss.Style
	editingStyle      lipgloss.Style
	highlightStyle    lipgloss.Style
	selectedTextStyle lipgloss.Style
	cursorStyle       lipgloss.Style
	totalStyle        lipgloss.Style
	totalOkStyle      lipgloss.Style
	totalOverStyle    lipgloss.Style
	helpStyle         lipgloss.Style
	loadingStyle      lipgloss.Style
}

type TimesheetModel struct {
	width        int
	height       int
	wndwSize     int
	wndwOffset   int
	timesheet    []TimeEntryR
	filtered     []TimeEntryR
	weekFrom     time.Time
	weekDays     [5]string
	cursorRow    int
	cursorCol    int
	editing      bool
	editBuffer   string
	firstEdit    bool
	cursorPos    int
	searchMode   bool
	searchQuery  string
	loading      bool
	spinner      spinner.Model
	styles       tsStyles
	taskColWidth int
	dayColWidth  int
}

const (
	colTask = iota
	colMon
	colTue
	colWed
	colThu
	colFri
)

func fetchTimesheetEntries() tea.Msg {
	config := clients.GetConfig()
	client := clients.NewClickupClient(config.ClickupToken, config.TeamId)
	userId := config.UserId
	filter := config.TimesheetFilter
	if filter == "" {
		filter = "tags[]=timesheet"
	}
	tasks, err := client.GetTimesheetTasks(filter)
	if err != nil {
		return loadedTimesheetMsg{err: err}
	}
	trackings, err := client.GetTimesheetsEntries(userId)
	if err != nil {
		return loadedTimesheetMsg{err: err}
	}

	datats := make([]TimeEntryR, len(tasks))
	for i, task := range tasks {
		datats[i] = TimeEntryR{
			TaskId:   task.Id,
			TaskName: task.Name,
			Hours:    make(map[string]float64),
		}
		for _, tracking := range trackings {
			if t, ok := tracking.Task.(map[string]interface{}); ok {
				if t["id"] == task.Id {
					day := shared.ToDateString(tracking.Start)
					datats[i].Hours[day] += shared.ToHours(tracking.Duration)
				}
			}
		}
	}

	return loadedTimesheetMsg{timesheet: datats}
}

func sortTimesheetEntries(entries []TimeEntryR, weekStart time.Time) []TimeEntryR {
	sorted := make([]TimeEntryR, len(entries))
	copy(sorted, entries)
	weekEndStr := weekStart.AddDate(0, 0, 7).Format("2006-01-02")
	weekStartStr := weekStart.Format("2006-01-02")

	sort.Slice(sorted, func(i, j int) bool {
		var hoursI, hoursJ float64
		for date, h := range sorted[i].Hours {
			if date >= weekStartStr && date < weekEndStr {
				hoursI += h
			}
		}
		for date, h := range sorted[j].Hours {
			if date >= weekStartStr && date < weekEndStr {
				hoursJ += h
			}
		}
		if hoursI > 0 && hoursJ == 0 {
			return true
		}
		if hoursI == 0 && hoursJ > 0 {
			return false
		}
		return sorted[i].TaskName < sorted[j].TaskName
	})
	return sorted
}

func filterTimesheet(entries []TimeEntryR, query string) []TimeEntryR {
	if query == "" {
		return entries
	}
	filtered := make([]TimeEntryR, 0)
	lowerQuery := strings.ToLower(query)
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.TaskName), lowerQuery) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func NewTimesheetModel() TimesheetModel {
	s := spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))))
	now := time.Now()
	weekFrom := now.AddDate(0, 0, -int(now.Weekday())+1)

	m := TimesheetModel{
		timesheet: []TimeEntryR{},
		filtered:  []TimeEntryR{},
		weekFrom:  weekFrom,
		loading:   true,
		spinner:   s,
		cursorCol: int(now.Weekday()),
	}

	if m.cursorCol == 0 || m.cursorCol > 5 {
		m.cursorCol = 1
	}

	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 120, 24
	}
	m.setSize(width, height)

	return m
}

func (m *TimesheetModel) setSize(width, height int) {
	m.width = width
	m.height = height

	availableHeight := height - 11
	m.wndwSize = availableHeight / 3
	if m.wndwSize < 1 {
		m.wndwSize = 1
	}

	numColumns := 6
	padding := 2
	availableWidth := m.width - (numColumns * padding)
	m.taskColWidth = availableWidth * 4 / 9
	m.dayColWidth = availableWidth / 9

	m.styles.headerStyle = ui.SubtitleStyle.Width(m.dayColWidth).Align(lipgloss.Center).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	m.styles.taskHeaderStyle = m.styles.headerStyle.Width(m.taskColWidth).Foreground(lipgloss.Color("212"))
	m.styles.cellStyle = lipgloss.NewStyle().Width(m.dayColWidth).Align(lipgloss.Center).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	m.styles.taskCellStyle = m.styles.cellStyle.Padding(0, 1).Width(m.taskColWidth).Align(lipgloss.Left)
	m.styles.selectedRowStyle = m.styles.taskCellStyle.BorderForeground(lipgloss.Color("250"))
	m.styles.selectedStyle = m.styles.cellStyle.BorderForeground(lipgloss.Color("86")).Bold(true)
	m.styles.editingStyle = m.styles.cellStyle.BorderForeground(lipgloss.Color("212")).Bold(true)
	m.styles.highlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("55"))
	m.styles.selectedTextStyle = lipgloss.NewStyle().Background(lipgloss.Color("212")).Foreground(lipgloss.Color("0"))
	m.styles.cursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("212")).Foreground(lipgloss.Color("0"))
	m.styles.totalStyle = m.styles.cellStyle.Foreground(lipgloss.Color("208"))
	m.styles.totalOkStyle = m.styles.totalStyle.Foreground(lipgloss.Color("72"))
	m.styles.totalOverStyle = m.styles.totalStyle.Foreground(lipgloss.Color("134"))
	m.styles.helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Padding(0, 1)
	m.styles.loadingStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#874BFD")).MarginLeft(2)
}

func (m *TimesheetModel) activeTimesheet() []TimeEntryR {
	if m.searchMode {
		return m.filtered
	}
	return m.timesheet
}

func (m *TimesheetModel) visibleTimesheet() []TimeEntryR {
	active := m.activeTimesheet()
	end := m.wndwOffset + m.wndwSize
	if end > len(active) {
		end = len(active)
	}
	if m.wndwOffset >= len(active) {
		return []TimeEntryR{}
	}
	return active[m.wndwOffset:end]
}

func (m *TimesheetModel) reapplyFiltersAndSort() {
	m.timesheet = sortTimesheetEntries(m.timesheet, m.weekFrom)
	m.filtered = filterTimesheet(m.timesheet, m.searchQuery)
	m.clampCursor()
}

func (m *TimesheetModel) clampCursor() {
	activeLen := len(m.activeTimesheet())
	if activeLen == 0 {
		m.cursorRow = 0
		m.wndwOffset = 0
		return
	}
	if m.cursorRow < 0 {
		m.cursorRow = 0
	}
	if m.cursorRow >= activeLen {
		m.cursorRow = activeLen - 1
	}
	if m.cursorRow < m.wndwOffset {
		m.wndwOffset = m.cursorRow
	}
	if m.cursorRow >= m.wndwOffset+m.wndwSize {
		m.wndwOffset = m.cursorRow - m.wndwSize + 1
	}
}

func (m *TimesheetModel) stopEditing() {
	m.editing = false
	m.editBuffer = ""
	m.firstEdit = false
	m.cursorPos = 0
}

func (m *TimesheetModel) startEditing() {
	if len(m.activeTimesheet()) == 0 {
		return
	}
	m.editing = true
	m.firstEdit = true
	dayKey := m.weekFrom.AddDate(0, 0, m.cursorCol-1).Format("2006-01-02")
	hours := m.activeTimesheet()[m.cursorRow].Hours[dayKey]
	m.editBuffer = fmt.Sprintf("%.2f", hours)
	m.cursorPos = len(m.editBuffer)
}

func (m TimesheetModel) Init() tea.Cmd {
	if m.loading {
		return tea.Batch(fetchTimesheetEntries, m.spinner.Tick)
	}
	return nil
}

func (m TimesheetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case LoadMsg:
		m.loading = true
		cmd = tea.Batch(fetchTimesheetEntries, m.spinner.Tick)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.MouseMsg:
		m.handleMouseInput(msg)
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
	case loadedTimesheetMsg:
		m.loading = false
		if msg.err == nil {
			m.timesheet = msg.timesheet
			m.reapplyFiltersAndSort()
		}
	}

	m.clampCursor()
	return m, cmd
}

func (m *TimesheetModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.editing {
		m.handleEditingInput(msg)
	} else if m.searchMode {
		m.handleSearchInput(msg)
	} else {
		cmd = m.handleNavigationInput(msg)
	}
	m.clampCursor()
	return m, cmd
}

func parseHoursInput(input string) (float64, error) {
	var hours, minutes float64
	input = strings.ToLower(strings.TrimSpace(input))

	if strings.Contains(input, "h") || strings.Contains(input, "m") {
		parts := strings.Fields(input)
		if len(parts) == 0 {
			parts = []string{input}
		}

		for _, part := range parts {
			if strings.Contains(part, "h") && strings.Contains(part, "m") {
				hPart := strings.Split(part, "h")[0]
				mPart := strings.Split(strings.Split(part, "h")[1], "m")[0]

				h, err := strconv.ParseFloat(hPart, 64)
				if err != nil {
					return 0, err
				}
				hours = h

				m, err := strconv.ParseFloat(mPart, 64)
				if err != nil {
					return 0, err
				}
				minutes = m
				continue
			}

			if strings.HasSuffix(part, "h") {
				h, err := strconv.ParseFloat(strings.TrimSuffix(part, "h"), 64)
				if err != nil {
					return 0, err
				}
				hours = h
			}
			if strings.HasSuffix(part, "m") {
				m, err := strconv.ParseFloat(strings.TrimSuffix(part, "m"), 64)
				if err != nil {
					return 0, err
				}
				minutes = m
			}
		}
		return hours + minutes/60.0, nil
	}

	return strconv.ParseFloat(input, 64)
}

func (m *TimesheetModel) handleEditingInput(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEnter:
		newHours, err := parseHoursInput(m.editBuffer)
		if err == nil && newHours >= 0 {
			entry := m.activeTimesheet()[m.cursorRow]
			day := m.weekFrom.AddDate(0, 0, m.cursorCol-1)
			config := clients.GetConfig()
			client := clients.NewClickupClient(config.ClickupToken, config.TeamId)
			if updateErr := client.UpdateTracking(config.UserId, entry.TaskId, day, newHours); updateErr != nil {
				fmt.Fprintf(os.Stderr, "Error updating tracking for task %s: %v\n", entry.TaskId, updateErr)
			} else {
				dayKey := day.Format("2006-01-02")
				for i := range m.timesheet {
					if m.timesheet[i].TaskId == entry.TaskId {
						m.timesheet[i].Hours[dayKey] = newHours
						break
					}
				}
				m.reapplyFiltersAndSort()
			}
		}
		m.stopEditing()
	case tea.KeyEscape:
		m.stopEditing()
	case tea.KeyBackspace:
		if m.firstEdit {
			m.editBuffer, m.cursorPos, m.firstEdit = "", 0, false
		} else if m.cursorPos > 0 {
			m.editBuffer = m.editBuffer[:m.cursorPos-1] + m.editBuffer[m.cursorPos:]
			m.cursorPos--
		}
	case tea.KeyLeft:
		if m.cursorPos > 0 {
			m.cursorPos--
		}
	case tea.KeyRight:
		if m.cursorPos < len(m.editBuffer) {
			m.cursorPos++
		}
	case tea.KeyRunes:
		if m.firstEdit {
			m.editBuffer, m.cursorPos, m.firstEdit = "", 0, false
		}
		m.editBuffer = m.editBuffer[:m.cursorPos] + string(msg.Runes) + m.editBuffer[m.cursorPos:]
		m.cursorPos += len(msg.Runes)
	}
}

func (m *TimesheetModel) handleSearchInput(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEscape:
		m.searchMode, m.searchQuery = false, ""
		m.reapplyFiltersAndSort()
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.filtered = filterTimesheet(m.timesheet, m.searchQuery)
			m.cursorRow = 0
		}
	case tea.KeyRunes:
		m.searchQuery += string(msg.Runes)
		m.filtered = filterTimesheet(m.timesheet, m.searchQuery)
		m.cursorRow = 0
	default:
		m.handleNavigationInput(msg)
	}
}

func (m *TimesheetModel) handleNavigationInput(msg tea.KeyMsg) tea.Cmd {
	switch key := msg.String(); key {
	case "q":
		return tea.Quit
	case "r":
		clients.ClearTimesheetTasksCache()
		clients.ClearTimeentriesCache()
		m.loading = true
		return tea.Batch(fetchTimesheetEntries, m.spinner.Tick)
	case "enter":
		if m.cursorCol > colTask && len(m.activeTimesheet()) > 0 {
			m.startEditing()
		}
	case "up":
		if m.cursorRow > 0 {
			m.cursorRow--
		}
	case "down":
		if m.cursorRow < len(m.activeTimesheet())-1 {
			m.cursorRow++
		}
	case "left":
		if m.cursorCol > colMon {
			m.cursorCol--
		} else {
			m.weekFrom, m.cursorCol = m.weekFrom.AddDate(0, 0, -7), colFri
			m.reapplyFiltersAndSort()
		}
	case "right":
		if m.cursorCol < colFri {
			m.cursorCol++
		} else {
			m.weekFrom, m.cursorCol = m.weekFrom.AddDate(0, 0, 7), colMon
			m.reapplyFiltersAndSort()
		}
	case "ctrl+left":
		m.weekFrom = m.weekFrom.AddDate(0, 0, -7)
		m.reapplyFiltersAndSort()
	case "ctrl+right":
		m.weekFrom = m.weekFrom.AddDate(0, 0, 7)
		m.reapplyFiltersAndSort()
	case "/":
		m.searchMode, m.searchQuery = true, ""
		m.filtered, m.cursorRow = m.timesheet, 0
	}
	return nil
}

func (m *TimesheetModel) calculateCellPositions() [][]position {
	positions := make([][]position, m.wndwSize)
	for i := range positions {
		positions[i] = make([]position, 6)
	}
	startY := 9
	rowHeight := 3
	padding := 2
	for r := 0; r < m.wndwSize; r++ {
		currentX := 0
		positions[r][colTask] = position{x: currentX, y: startY + r*rowHeight, width: m.taskColWidth + padding, height: rowHeight}
		currentX += m.taskColWidth + padding
		for c := colMon; c <= colFri; c++ {
			positions[r][c] = position{x: currentX, y: startY + r*rowHeight, width: m.dayColWidth + padding, height: rowHeight}
			currentX += m.dayColWidth + padding
		}
	}
	return positions
}

func (m *TimesheetModel) handleMouseInput(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.cursorRow > 0 {
			m.cursorRow--
		}
	case tea.MouseButtonWheelDown:
		if m.cursorRow < len(m.activeTimesheet())-1 {
			m.cursorRow++
		}
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionRelease {
			return
		}
		if m.editing {
			m.stopEditing()
			return
		}
		positions := m.calculateCellPositions()
		for r, rowPos := range positions {
			for c, cellPos := range rowPos {
				if c > 0 && msg.X >= cellPos.x && msg.X < cellPos.x+cellPos.width && msg.Y >= cellPos.y && msg.Y < cellPos.y+cellPos.height {
					clickedDataRow := r + m.wndwOffset
					if clickedDataRow >= len(m.activeTimesheet()) {
						return
					}
					isCurrentlySelectedCell := clickedDataRow == m.cursorRow && c == m.cursorCol
					if isCurrentlySelectedCell {
						if c != colTask {
							m.startEditing()
						}
					} else {
						m.cursorRow, m.cursorCol = clickedDataRow, c
						m.stopEditing()
					}
					return
				}
			}
		}
		m.stopEditing()
	}
}

func (m TimesheetModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}
	if m.loading {
		return m.styles.loadingStyle.Render("Loading timesheet... ") + m.spinner.View()
	}

	for i := 0; i < 5; i++ {
		m.weekDays[i] = m.weekFrom.AddDate(0, 0, i).Format("Mon 2")
	}

	title := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, ui.TitleStyle.Render("Weekly Timesheet"))
	table := m.renderTable()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(lipgloss.Left, title, "\n", table)
	if paddingHeight := m.height - lipgloss.Height(content) - lipgloss.Height(help); paddingHeight > 0 {
		content = lipgloss.JoinVertical(lipgloss.Left, content, strings.Repeat("\n", paddingHeight-1))
	}

	return lipgloss.JoinVertical(lipgloss.Left, content, m.styles.helpStyle.Render(help))
}

func (m *TimesheetModel) renderTable() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), m.renderTotalsRow(), m.renderBody())
}

func (m *TimesheetModel) renderHeader() string {
	headers := []string{m.styles.taskHeaderStyle.Render(m.weekFrom.Format("January 2006"))}
	for _, day := range m.weekDays {
		headers = append(headers, m.styles.headerStyle.Render(day))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, headers...)
}

func formatHoursToHM(hours float64) string {
	if hours == 0 {
		return "-"
	}
	h := int(hours)
	m := int((hours - float64(h)) * 60)
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func (m *TimesheetModel) renderTotalsRow() string {
	totals := make([]float64, 5)
	for _, entry := range m.timesheet {
		for i := 0; i < 5; i++ {
			day := m.weekFrom.AddDate(0, 0, i).Format("2006-01-02")
			totals[i] += entry.Hours[day]
		}
	}
	totalCells := []string{m.styles.taskCellStyle.Foreground(lipgloss.Color("72")).Render("Total")}
	for _, total := range totals {
		style := m.styles.totalStyle
		if total == 8 {
			style = m.styles.totalOkStyle
		} else if total > 8 {
			style = m.styles.totalOverStyle
		}
		totalCells = append(totalCells, style.Render(formatHoursToHM(total)))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, totalCells...)
}

func (m *TimesheetModel) renderBody() string {
	var rows []string
	for i, entry := range m.visibleTimesheet() {
		rows = append(rows, m.renderRow(entry, m.wndwOffset+i))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m *TimesheetModel) renderRow(entry TimeEntryR, rowIdx int) string {
	isCursorRow := (rowIdx == m.cursorRow)
	taskCell := m.renderTaskCell(entry.TaskName, isCursorRow)
	dayCells := make([]string, 5)
	for i := 0; i < 5; i++ {
		day := m.weekFrom.AddDate(0, 0, i).Format("2006-01-02")
		dayCells[i] = m.renderDayCell(entry.Hours[day], isCursorRow, i+1)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, append([]string{taskCell}, dayCells...)...)
}

func (m *TimesheetModel) renderTaskCell(taskNameInput string, isCursorRow bool) string {
	var currentStyle lipgloss.Style
	if isCursorRow {
		currentStyle = m.styles.selectedRowStyle
	} else {
		currentStyle = m.styles.taskCellStyle
	}

	finalTextContent := taskNameInput

	if m.searchMode && len(m.searchQuery) > 0 {
		lowerTask := strings.ToLower(taskNameInput)
		lowerQuery := strings.ToLower(m.searchQuery)

		if idx := strings.Index(lowerTask, lowerQuery); idx >= 0 {
			before := taskNameInput[:idx]
			matchText := taskNameInput[idx : idx+len(m.searchQuery)]
			after := taskNameInput[idx+len(m.searchQuery):]

			contentBudgetForSpecialLayout := m.taskColWidth - 8

			if len(taskNameInput) > contentBudgetForSpecialLayout && contentBudgetForSpecialLayout > 0 {
				halfBudget := contentBudgetForSpecialLayout / 2

				bPart := before
				if len(before) > halfBudget {
					bPart = before[:halfBudget]
				}

				mPartText := matchText
				maxMatchHighlightLen := 9
				if len(matchText) > maxMatchHighlightLen {
					mPartText = matchText[:maxMatchHighlightLen]
				}
				highlightedMatchPart := m.styles.highlightStyle.Render(mPartText)

				aPart := after
				startAfterIndex := len(after) - halfBudget
				if startAfterIndex < 0 {
					startAfterIndex = 0
				}
				aPart = after[startAfterIndex:]

				finalTextContent = bPart + ".." + highlightedMatchPart + ".." + aPart
			} else {
				finalTextContent = before + m.styles.highlightStyle.Render(matchText) + after
			}
		}
	} else if maxLen := m.taskColWidth - 6; lipgloss.Width(taskNameInput) > maxLen && maxLen > 0 {
		finalTextContent = taskNameInput[:maxLen] + "..."
	}

	return currentStyle.Render(finalTextContent)
}

func (m *TimesheetModel) renderDayCell(hours float64, isCursorRow bool, colIdx int) string {
	style, content := m.styles.cellStyle, "-"
	if hours > 0 {
		content = formatHoursToHM(hours)
	}
	if isCursorRow && colIdx == m.cursorCol {
		if m.editing {
			style, content = m.styles.editingStyle, m.editBuffer
			if m.firstEdit {
				content = m.styles.selectedTextStyle.Render(content)
			} else if m.cursorPos < len(content) {
				content = content[:m.cursorPos] + m.styles.cursorStyle.Render(string(content[m.cursorPos])) + content[m.cursorPos+1:]
			} else {
				content += m.styles.cursorStyle.Render(" ")
			}
		} else {
			style = m.styles.selectedStyle
		}
	}
	return style.Render(content)
}

func (m *TimesheetModel) renderHelp() string {
	if m.searchMode {
		searchBar := "Search: " + m.searchQuery
		return lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Width(m.width-38).Render(searchBar),
			"[esc] Exit Search    [↑↓] Navigate",
		)
	}
	return "[← ↑ → ↓] Navigate   [enter] Select/Edit   [/] Search   [tab] View   [r] Refresh    [?] Settings   [q] Quit"
}
