package views

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/mceck/clickup-tui/internal/clients"
	"github.com/mceck/clickup-tui/internal/shared"
	"golang.org/x/term"
)

type KColumn struct {
	offsetY int
	tasks   []clients.Task
}

type taskLoadedMsg struct {
	tasks []clients.Task
	err   error
}

type HomeModel struct {
	tasks            []clients.Task
	viewport         viewport.Model
	width            int
	height           int
	offsetX          int
	wndX             int
	wndY             int
	columnWidth      int
	selectedColumn   int
	selectedTask     int
	states           []string // Keep track of states order
	columns          map[string]KColumn
	viewInput        textinput.Model
	inputActive      bool
	loading          bool
	spinner          spinner.Model
	showModal        bool           // Whether to show the task details modal
	modalTask        *clients.Task  // The task to show in the modal
	contentViewport  viewport.Model // Viewport for scrollable content
	commentsViewport viewport.Model // Viewport for scrollable comments
}

func calculateWindowDimensions(width, height, columnWidth int) (wndX, wndY int) {
	const (
		headerHeight = 3 // space for header
		footerHeight = 1 // space for help text
		taskHeight   = 9 // height of each task card including borders
		columnGap    = 1 // space between columns
	)

	// Calculate how many columns can fit
	wndX = (width - columnGap) / (columnWidth + columnGap)
	if wndX < 1 {
		wndX = 1
	}

	// Calculate how many tasks can fit vertically
	availableHeight := height - headerHeight - footerHeight
	wndY = (availableHeight / taskHeight)
	if wndY < 1 {
		wndY = 1
	}

	return wndX, wndY
}

func fetchTasks() tea.Msg {
	config := clients.GetConfig()
	client := clients.NewClickupClient(config.ClickupToken, config.TeamId)
	tasks, err := client.GetViewTasks(config.ViewId)
	return taskLoadedMsg{tasks: tasks, err: err}
}

func NewHomeModel() tea.Model {
	config := clients.GetConfig()
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width, height = 80, 24 // Default size
	}
	if config.ViewId == "" {
		input := textinput.New()
		input.Placeholder = "Inserisci il View ID..."
		input.Focus()
		input.CharLimit = 50
		input.Width = 40

		return HomeModel{
			width:       width,
			height:      height,
			viewInput:   input,
			inputActive: true,
		}
	}

	columnWidth := 30
	wndX, wndY := calculateWindowDimensions(width, height, columnWidth)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return HomeModel{
		width:       width,
		height:      height,
		columnWidth: columnWidth,
		wndX:        wndX,
		wndY:        wndY,
		loading:     true,
		spinner:     s,
		columns:     make(map[string]KColumn),
	}
}

func (m HomeModel) Init() tea.Cmd {
	if m.loading {
		return tea.Batch(fetchTasks, m.spinner.Tick)
	}
	return nil
}

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.inputActive {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				if m.viewInput.Value() != "" {
					config := clients.GetConfig()
					config.ViewId = m.viewInput.Value()
					if err := clients.SaveConfig(config); err != nil {
						return m, tea.Quit
					}
					clients.ClearCache()
					return NewHomeModel(), nil
				}
			}

			m.viewInput, cmd = m.viewInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if m.loading {
		switch msg := msg.(type) {
		case taskLoadedMsg:
			if msg.err != nil {
				fmt.Println("Error fetching tasks:", msg.err)
				return m, tea.Quit
			}
			m.loading = false
			m.tasks = msg.tasks

			stateMap := make(map[string]int)
			for _, t := range m.tasks {
				if existingOrder, ok := stateMap[t.Status.Status]; !ok || t.Status.Orderindex < existingOrder {
					stateMap[t.Status.Status] = t.Status.Orderindex
				}
			}

			states := make([]string, 0, len(stateMap))
			for name := range stateMap {
				states = append(states, name)
			}

			sort.Slice(states, func(i, j int) bool {
				return stateMap[states[i]] < stateMap[states[j]]
			})

			m.states = states
			m.columns = make(map[string]KColumn)
			for _, task := range m.tasks {
				if _, ok := m.columns[task.Status.Status]; !ok {
					m.columns[task.Status.Status] = KColumn{
						offsetY: 0,
						tasks:   []clients.Task{},
					}
				}

				m.columns[task.Status.Status] = KColumn{
					offsetY: 0,
					tasks:   append(m.columns[task.Status.Status].tasks, task),
				}
			}
			return m, nil

		case spinner.TickMsg:
			var spinnerCmd tea.Cmd
			m.spinner, spinnerCmd = m.spinner.Update(msg)
			return m, spinnerCmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.wndX, m.wndY = calculateWindowDimensions(m.width, m.height, m.columnWidth)
		m.viewport = viewport.New(m.width, m.height-3) // 3 is header height

		// Update modal viewports if modal is open
		if m.showModal {
			m.commentsViewport.Width = m.width - 11
			m.commentsViewport.Height = 8
			m.contentViewport.Width = m.width - 9
			m.contentViewport.Height = m.height - 23
		}

		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "r":
			clients.ClearCache()
			m.loading = true
			return m, tea.Batch(fetchTasks, m.spinner.Tick)
		case "enter":
			if !m.showModal {
				// Get the selected task
				if column, ok := m.columns[m.states[m.selectedColumn]]; ok {
					if m.selectedTask < len(column.tasks) {
						task := column.tasks[m.selectedTask]
						config := clients.GetConfig()
						client := clients.NewClickupClient(config.ClickupToken, config.TeamId)
						t, err := client.GetTask(task.Id)
						if err != nil {
							fmt.Println("Error fetching task:", err)
							return m, nil
						}
						m.modalTask = &t
						m.showModal = true
						// Initialize viewports
						contentHeight := m.height - 23 // Same as content style height
						commentsHeight := 8            // Same as comments style height
						m.contentViewport = viewport.New(m.width-9, contentHeight)
						m.contentViewport.YPosition = 0
						m.commentsViewport = viewport.New(m.width-11, commentsHeight-2) // -2 for header
						m.commentsViewport.YPosition = 0

						// Set content in viewports
						contentStyle := lipgloss.NewStyle().
							Width(m.width - 9).
							Align(lipgloss.Left)

						// Render markdown using glamour
						var renderedMarkdown string
						if m.modalTask.Description != "" {
							rendered, err := glamour.Render(m.modalTask.Description, "dark")
							if err == nil {
								renderedMarkdown = rendered
							} else {
								renderedMarkdown = m.modalTask.Description
							}
						} else {
							renderedMarkdown = ""
						}
						m.contentViewport.SetContent(contentStyle.Render(renderedMarkdown))

						// Build comments section
						comments, err := client.GetTaskComments(task.Id)

						if err != nil {
							fmt.Println("Error fetching comments:", err)
							comments = []clients.Comment{}
						}

						var commentsContent []string
						for _, comment := range comments {
							color := comment.User.Color
							if color == "" {
								color = "#888888"
							}
							commentHeader := lipgloss.JoinHorizontal(lipgloss.Left,
								lipgloss.NewStyle().
									Bold(true).
									Background(lipgloss.Color(color)).
									Foreground(lipgloss.Color("#FFFFFF")).
									Padding(0, 1).
									MarginRight(1).
									Render(comment.User.Initials),
								" ",
								lipgloss.NewStyle().Width(m.width-23).Foreground(lipgloss.Color("#666666")).Render(comment.User.Username),
								" ",
								lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(shared.ToElapsedTime(comment.Date)),
							)
							commentLine := lipgloss.NewStyle().Width(m.width - 15).Render(RenderCommentText(comment.Comment))
							commentsContent = append(commentsContent, commentHeader)
							commentsContent = append(commentsContent, commentLine)
						}
						m.commentsViewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, commentsContent...))
					}
				}
				return m, nil
			} else {
				// Close modal if it's open
				m.showModal = false
				m.modalTask = nil
				return m, nil
			}
		case "esc":
			if m.showModal {
				m.showModal = false
				m.modalTask = nil
				return m, nil
			}
		case "up", "down":
			if m.showModal {
				var cmd tea.Cmd
				switch msg.String() {
				case "up":
					m.contentViewport.ScrollUp(1)
				case "down":
					m.contentViewport.ScrollDown(1)
				}
				return m, cmd
			} else {
				// Handle regular navigation when modal is not shown
				switch msg.String() {
				case "up":
					if m.selectedTask > 0 {
						m.selectedTask--
						// Scroll up if necessary
						if m.selectedTask < m.columns[m.states[m.selectedColumn]].offsetY {
							column := m.columns[m.states[m.selectedColumn]]
							column.offsetY--
							m.columns[m.states[m.selectedColumn]] = column
						}
					}
				case "down":
					currentTasks := len(m.columns[m.states[m.selectedColumn]].tasks)
					if m.selectedTask < currentTasks-1 {
						m.selectedTask++
						// Scroll down if necessary
						if m.selectedTask >= m.columns[m.states[m.selectedColumn]].offsetY+m.wndY {
							column := m.columns[m.states[m.selectedColumn]]
							column.offsetY++
							m.columns[m.states[m.selectedColumn]] = column
						}
					}
				}
			}
		case "pgup", "pgdown":
			if m.showModal {
				var cmd tea.Cmd
				switch msg.String() {
				case "pgup":
					m.commentsViewport.ScrollUp(1)
				case "pgdown":
					m.commentsViewport.ScrollDown(1)
				}
				return m, cmd
			}
		case "left":
			if !m.showModal && m.selectedColumn > 0 {
				m.selectedColumn--
				// Scroll left if necessary
				if m.selectedColumn < m.offsetX {
					m.offsetX--
				}
				// Set selection to first visible task in column
				if column, ok := m.columns[m.states[m.selectedColumn]]; ok {
					m.selectedTask = column.offsetY
				}
			}
		case "right":
			if !m.showModal && m.selectedColumn < len(m.states)-1 {
				m.selectedColumn++
				// Scroll right if necessary
				if m.selectedColumn >= m.offsetX+m.wndX {
					m.offsetX++
				}
				// Set selection to first visible task in column
				if column, ok := m.columns[m.states[m.selectedColumn]]; ok {
					m.selectedTask = column.offsetY
				}
			}
		}
	}

	return m, nil
}

func (m HomeModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Inline(true).  // Ensure the help text stays on one line
		MarginLeft(1). // Small left margin instead of padding
		Height(1)      // Force height to be exactly 1 line

	if m.inputActive {
		formStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2)

		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#874BFD")).
			MarginBottom(1)

		content := lipgloss.JoinVertical(lipgloss.Center,
			titleStyle.Render("Configurazione ClickUp"),
			"Inserisci il View ID:",
			m.viewInput.View(),
			"",
			"Premi Enter per salvare, Ctrl+C per uscire",
		)

		renderedForm := formStyle.Render(content)

		// Center the form vertically
		emptyLines := (m.height - lipgloss.Height(renderedForm) - 2) / 2
		if emptyLines > 0 {
			renderedForm = strings.Repeat("\n", emptyLines) + renderedForm
		}

		helpText := helpStyle.Render("← →  colonne   ↑ ↓  task   tab Timesheet   q esci")
		// Fill remaining space to push help to bottom, accounting for exact 1 line help text
		remainingLines := m.height - lipgloss.Height(renderedForm) - 1 - emptyLines - 1
		if remainingLines > 0 {
			renderedForm += strings.Repeat("\n", remainingLines)
		}

		return renderedForm + "\n" + helpText
	}

	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#874BFD")).
			MarginLeft(2)

		content := loadingStyle.Render("Caricamento tasks... ") + m.spinner.View()

		return content
	}

	// Funzione helper per ottenere il colore dello status
	getStatusColor := func(state string) string {
		if len(m.columns[state].tasks) > 0 {
			return m.columns[state].tasks[0].Status.Color
		}
		return "#874BFD"
	}

	// Header style per le colonne
	headerCardStyle := func(state string) lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(getStatusColor(state))).
			Padding(0, 1).
			Width(m.columnWidth - 4).
			Align(lipgloss.Center)
	}

	// Base style per le colonne
	columnStyle := func(state string) lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(getStatusColor(state))).
			Padding(0, 1).
			Width(m.columnWidth)
	}

	selectedColumnStyle := func(state string) lipgloss.Style {
		return columnStyle(state).
			BorderStyle(lipgloss.DoubleBorder())
	}

	// Stili per i task
	taskCardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4A4A4A")).
		Padding(0, 1).
		Width(m.columnWidth - 4)

	selectedTaskCardStyle := taskCardStyle.
		BorderForeground(lipgloss.Color(getStatusColor(""))).
		BorderStyle(lipgloss.DoubleBorder())

	listNameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		MaxWidth(m.columnWidth - 5)

	taskNameStyle := lipgloss.NewStyle().
		Bold(true).
		MaxWidth(m.columnWidth - 5).
		MaxHeight(4).
		Inline(false)

	// Add this new style for tags
	tagStyle := func(bg string, fg string) lipgloss.Style {
		if bg == "" {
			bg = "#888888" // Default color for tags without color
		}
		if fg == "" || fg == bg {
			fg = "#ffffff" // Default color for tags without color
		}
		return lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color(bg)).
			Foreground(lipgloss.Color(fg)).
			Padding(0, 1).
			MarginRight(1)
	}

	// Add this new style near the other style definitions
	subtaskStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		MarginRight(1)

	// Renderizza le colonne
	renderedColumns := make([]string, 0)
	maxColumnHeight := m.height - 2 // reserve space for only help bar and its newline

	for i, state := range m.states[m.offsetX:min(m.offsetX+m.wndX, len(m.states))] {
		tasks := m.columns[state].tasks

		// Renderizza i task nella colonna
		renderedTasks := make([]string, 0)
		visibleTasks := tasks[m.columns[state].offsetY:min(m.columns[state].offsetY+m.wndY, len(tasks))]

		for j, task := range visibleTasks {
			style := taskCardStyle
			if m.selectedColumn == i+m.offsetX && m.selectedTask == j+m.columns[state].offsetY {
				style = selectedTaskCardStyle
			}

			// Create assignees row
			assigneesRow := ""
			if len(task.Assignees) > 0 {
				assigneeCards := make([]string, 0)
				for _, assignee := range task.Assignees {
					color := assignee.Color
					if color == "" {
						// Generate random color if none provided
						colors := []string{"#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FF00FF", "#00FFFF"}
						color = colors[len(task.Assignees)%len(colors)]
					}
					assigneeCards = append(assigneeCards,
						lipgloss.NewStyle().
							Bold(true).
							Background(lipgloss.Color(color)).
							Foreground(lipgloss.Color("#FFFFFF")).
							Padding(0, 1).
							MarginRight(1).
							Render(assignee.Initials))
				}
				assigneesRow = lipgloss.JoinHorizontal(lipgloss.Left, assigneeCards...)
			}

			// Create tags and subtask row
			tagsRow := ""
			subtaskInfo := ""
			if task.SubTasksCount > 0 {
				subtaskInfo = subtaskStyle.Render(fmt.Sprintf("📋 %d", task.SubTasksCount))
			}
			if len(task.Tags) > 0 {
				tagCards := make([]string, 0)
				for _, tag := range task.Tags {
					tagCards = append(tagCards,
						tagStyle(tag.TagBg, tag.TagFg).Render(tag.Name))
				}
				tagsRow = lipgloss.JoinHorizontal(lipgloss.Right, tagCards...)
			}
			// Combine subtask count and tags in one row
			bottomRow := lipgloss.JoinHorizontal(lipgloss.Left,
				subtaskInfo,
				lipgloss.NewStyle().
					Width(m.columnWidth-8-lipgloss.Width(subtaskInfo)).
					AlignHorizontal(lipgloss.Right).
					Render(tagsRow),
			)

			// Wrappa il testo del task name
			wrappedTaskName := lipgloss.NewStyle().
				Width(m.columnWidth - 8).
				Height(3).
				Render(task.Name)

			taskContent := lipgloss.JoinVertical(lipgloss.Left,
				assigneesRow,
				listNameStyle.Render("📁 "+task.List.Name),
				taskNameStyle.Render(wrappedTaskName),
				bottomRow,
			)
			renderedTasks = append(renderedTasks, style.Render(taskContent))
		}

		// Usa lo stile evidenziato per la colonna selezionata
		style := columnStyle(state)
		if m.selectedColumn == i+m.offsetX {
			style = selectedColumnStyle(state)
		}

		// Header della colonna
		headerCard := headerCardStyle(state).Render(
			lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(getStatusColor(state))).
				Render(strings.ToUpper(state)),
		)

		column := style.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				headerCard,
				lipgloss.JoinVertical(lipgloss.Left, renderedTasks...),
			),
		)

		// Ensure column doesn't exceed maximum height
		if lipgloss.Height(column) > maxColumnHeight {
			lines := strings.Split(column, "\n")
			column = strings.Join(lines[:maxColumnHeight], "\n")
		}

		renderedColumns = append(renderedColumns, column)
	}

	finalView := lipgloss.JoinHorizontal(lipgloss.Top, renderedColumns...)

	// Calculate space needed to push help bar to bottom with minimal spacing
	var helpText string
	if m.showModal {
		helpText = helpStyle.Render("[↑/↓] scroll content    [pgup/pgdn] scroll comments    [enter/esc] close")
	} else {
		helpText = helpStyle.Render("[← → ↑ ↓] navigate    [enter] Task Details    [tab] Timesheet    [q] quit")
	}
	contentHeight := lipgloss.Height(finalView)
	paddingHeight := m.height - contentHeight - 1 // -1 for help bar (which includes its own newline)

	if paddingHeight > 0 {
		finalView += strings.Repeat("\n", paddingHeight)
	}

	// Render modal if it's active
	if m.showModal && m.modalTask != nil {
		// Create modal styles
		modalStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(m.modalTask.Status.Color)).
			Padding(1, 2).
			Width(m.width - 3). // Full screen width
			Align(lipgloss.Left)

		// Header styles
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(m.modalTask.Status.Color)).
			Width(m.width - 6). // Account for modal padding and borders
			Align(lipgloss.Left)

		listStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			MarginBottom(1)

		// Status and assignees styles
		metaRowStyle := lipgloss.NewStyle().
			Width(m.width - 6).
			MarginTop(1).
			MarginBottom(1)

		statusStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(m.modalTask.Status.Color)).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			MarginRight(2)

		// Comments section styles
		commentsSectionStyle := lipgloss.NewStyle().
			Width(m.width-9).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Padding(0, 1)

		commentHeaderStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#888888")).
			MarginBottom(1)

		// Build comments section
		comments := commentsSectionStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				commentHeaderStyle.Render("Comments"),
				m.commentsViewport.View(),
			),
		)

		// Build header section
		header := lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(m.modalTask.Name),
			listStyle.Render("📁 "+m.modalTask.List.Name),
		)

		// Build meta info row (status and assignees)
		metaInfo := []string{statusStyle.Render(m.modalTask.Status.Status)}
		if len(m.modalTask.Assignees) > 0 {
			for _, assignee := range m.modalTask.Assignees {
				color := assignee.Color
				if color == "" {
					color = "#888888"
				}
				metaInfo = append(metaInfo, lipgloss.NewStyle().
					Bold(true).
					Background(lipgloss.Color(color)).
					Foreground(lipgloss.Color("#FFFFFF")).
					Padding(0, 1).
					MarginRight(1).
					Render(assignee.Initials))
			}
		}
		metaRow := metaRowStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, metaInfo...))

		// Join all sections
		modalContent := lipgloss.JoinVertical(lipgloss.Left,
			header,
			metaRow,
			m.contentViewport.View(),
			comments,
		)

		// Render full-screen modal
		renderedModal := modalStyle.Render(modalContent)

		// Return the view with the modal
		return lipgloss.JoinVertical(
			lipgloss.Left,
			"DOS - Flusso di lavoro",
			renderedModal,
			helpText,
		)
	}

	return "DOS - Flusso di lavoro\n" + finalView + helpText
}

// RenderCommentText renders a slice of CommentText as a formatted string using lipgloss.
func RenderCommentText(comment []clients.CommentText) string {
	var sb strings.Builder
	for _, part := range comment {
		switch part.Type {
		case "bookmark":
			// Render bookmark as a link (show title or URL)
			url, _ := part.Bookmark["url"].(string)
			title := url
			if raw, ok := part.Attributes["raw"].(string); ok && raw != "" {
				// Try to decode base64 and extract title if possible
				decoded, err := decodeBookmarkRaw(raw)
				if err == nil && decoded.Title != "" {
					title = decoded.Title
				}
			}
			linkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Underline(true)
			if title == url {
				sb.WriteString(linkStyle.Render(title) + " \n")
			} else {
				sb.WriteString(linkStyle.Render(title) + " → " + url + " \n")
			}
		case "":
			// Plain text, check for badge-class
			if badge, ok := part.Attributes["badge-class"].(string); ok && badge != "" {
				badgeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#FF5F87")).Padding(0, 1)
				sb.WriteString(badgeStyle.Render(part.Text) + " ")
			} else {
				textStyle := lipgloss.NewStyle().Render(part.Text)
				sb.WriteString(textStyle)
			}
		default:
			// Fallback: just render text
			textStyle := lipgloss.NewStyle().Render(part.Text)
			sb.WriteString(textStyle)
		}
	}
	return sb.String()
}

// decodeBookmarkRaw decodes the base64-encoded raw attribute for bookmarks.
type bookmarkRaw struct {
	Title string `json:"title"`
}

func decodeBookmarkRaw(raw string) (bookmarkRaw, error) {
	decoded := bookmarkRaw{}
	data, err := decodeBase64(raw)
	if err != nil {
		return decoded, err
	}
	// Try to extract title from JSON
	var m map[string]interface{}
	err = json.Unmarshal(data, &m)
	if err == nil {
		if preview, ok := m["preview"].(map[string]interface{}); ok {
			if title, ok := preview["title"].(string); ok {
				decoded.Title = title
			}
		}
	}
	return decoded, nil
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
