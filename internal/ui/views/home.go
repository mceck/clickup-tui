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

const (
	columnWidth  = 36
	headerHeight = 5
	footerHeight = 1
	taskHeight   = 10
	columnGap    = 1
)

type KColumn struct {
	offsetY int
	tasks   []clients.Task
}

type LoadMsg struct{}

type taskLoadedMsg struct {
	tasks []clients.Task
	err   error
}

type HomeModel struct {
	width            int
	height           int
	wndX             int
	wndY             int
	offsetX          int
	selectedColumn   int
	selectedTask     int
	states           []string
	columns          map[string]KColumn
	loading          bool
	spinner          spinner.Model
	inputActive      bool
	viewInput        textinput.Model
	showModal        bool
	modalTask        *clients.Task
	contentViewport  viewport.Model
	commentsViewport viewport.Model
}

func calculateWindowDimensions(width, height int) (wndX, wndY int) {
	wndX = (width - columnGap) / (columnWidth + columnGap)
	if wndX < 1 {
		wndX = 1
	}
	wndY = (height - headerHeight - footerHeight) / taskHeight
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
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	if width == 0 || height == 0 {
		width, height = 80, 24
	}

	config := clients.GetConfig()
	if config.ViewId == "" {
		input := textinput.New()
		input.Placeholder = "Inserisci il View ID..."
		input.Focus()
		input.CharLimit = 50
		input.Width = 40
		return HomeModel{width: width, height: height, viewInput: input, inputActive: true}
	}

	wndX, wndY := calculateWindowDimensions(width, height)
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return HomeModel{
		width:   width,
		height:  height,
		wndX:    wndX,
		wndY:    wndY,
		loading: true,
		spinner: s,
		columns: make(map[string]KColumn),
	}
}

func (m HomeModel) Init() tea.Cmd {
	if m.loading {
		return tea.Batch(fetchTasks, m.spinner.Tick)
	}
	return nil
}

func (m *HomeModel) processTasks(tasks []clients.Task) {

	stateMap := make(map[string]int)
	for _, t := range tasks {
		if existingOrder, ok := stateMap[t.Status.Status]; !ok || t.Status.Orderindex < existingOrder {
			stateMap[t.Status.Status] = t.Status.Orderindex
		}
	}

	states := make([]string, 0, len(stateMap))
	for name := range stateMap {
		states = append(states, name)
	}
	sort.Slice(states, func(i, j int) bool { return stateMap[states[i]] < stateMap[states[j]] })

	m.states = states
	m.columns = make(map[string]KColumn)
	for _, state := range states {
		m.columns[state] = KColumn{tasks: []clients.Task{}}
	}
	for _, task := range tasks {
		col := m.columns[task.Status.Status]
		col.tasks = append(col.tasks, task)
		m.columns[task.Status.Status] = col
	}
}

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.inputActive {
		switch kmsg := msg.(type) {
		case tea.KeyMsg:
			return m.handleInputActiveEvent(kmsg)
		default:
			// While input is active, we only care about KeyMsgs.
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case LoadMsg:
		return m.handleLoadTasksEvent()
	case tea.WindowSizeMsg:
		return m.handleWindowSizeEvent(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTickEvent(msg)
	case taskLoadedMsg:
		return m.handleTasksLoadedEvent(msg)
	case tea.KeyMsg:
		return m.handleKeyEvent(msg)
	}

	return m, nil
}

func (m HomeModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}
	if m.inputActive {
		return m.viewInputScreen()
	}
	if m.loading {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#874BFD")).MarginLeft(2).Render("Caricamento tasks... ") + m.spinner.View()
	}

	var mainView string
	if m.showModal && m.modalTask != nil {
		mainView = m.viewModal()
	} else {
		mainView = m.viewBoard()
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Height(1)
	var helpText string
	if m.showModal {
		helpText = helpStyle.Render("\n[↑/↓] scroll content    [j/k] scroll comments    [enter/esc] close")
	} else {
		helpText = helpStyle.Render("[← → ↑ ↓] navigate    [enter] Task Details    [tab] Timesheet    [r] refresh    [q] quit")
	}

	paddingHeight := m.height - lipgloss.Height(mainView)
	if paddingHeight < 0 {
		paddingHeight = 0
	}
	return mainView + strings.Repeat("\n", paddingHeight-1) + helpText
}

func (m HomeModel) viewInputScreen() string {
	formStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#874BFD")).Padding(1, 2)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#874BFD")).MarginBottom(1)
	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render("Configurazione ClickUp"),
		"Inserisci il View ID:",
		m.viewInput.View(),
		"",
		"Premi Enter per salvare, Ctrl+C per uscire",
	)
	form := formStyle.Render(content)
	return lipgloss.Place(m.width, m.height-1, lipgloss.Center, lipgloss.Center, form)
}

func (m HomeModel) viewBoard() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#874BFD")).MarginBottom(1)
	titleView := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title.Render("ClickUp View"))

	renderedColumns := make([]string, 0)
	visibleStates := m.states[m.offsetX:min(m.offsetX+m.wndX, len(m.states))]

	for i, state := range visibleStates {
		isSelectedCol := m.selectedColumn == i+m.offsetX
		renderedColumns = append(renderedColumns, m.renderColumn(state, isSelectedCol))
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, renderedColumns...)
	return lipgloss.JoinVertical(lipgloss.Left, titleView, board)
}

func (m HomeModel) renderColumn(state string, isSelected bool) string {
	colData := m.columns[state]
	color := "#874BFD"
	if len(colData.tasks) > 0 {
		color = colData.tasks[0].Status.Color
	}

	headerStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(color)).Padding(0, 1).Width(columnWidth - 4).Align(lipgloss.Center)
	headerText := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color)).Render(fmt.Sprintf("%s (%d)", strings.ToUpper(state), len(colData.tasks)))
	header := headerStyle.Render(headerText)

	renderedTasks := make([]string, 0)
	visibleTasks := colData.tasks[colData.offsetY:min(colData.offsetY+m.wndY, len(colData.tasks))]
	for j, task := range visibleTasks {
		isTaskSelected := isSelected && m.selectedTask == j+colData.offsetY
		renderedTasks = append(renderedTasks, m.renderTask(task, isTaskSelected, color))
	}

	colStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(color)).Padding(0, 1).Width(columnWidth)
	if isSelected {
		colStyle = colStyle.BorderStyle(lipgloss.DoubleBorder())
	}
	return colStyle.Render(lipgloss.JoinVertical(lipgloss.Left, header, lipgloss.JoinVertical(lipgloss.Left, renderedTasks...)))
}

func (m HomeModel) renderTask(task clients.Task, isSelected bool, statusColor string) string {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#4A4A4A")).Padding(0, 1).Width(columnWidth - 4)
	if isSelected {
		highlightColor := shared.LightenColor(statusColor, 0.8)
		style = style.BorderForeground(lipgloss.Color(highlightColor)).BorderStyle(lipgloss.DoubleBorder())
	}

	var assignees []string
	for _, a := range task.Assignees {
		bgColor := a.Color
		if bgColor == "" {
			bgColor = "#888888"
		}
		assignees = append(assignees, lipgloss.NewStyle().Bold(true).Background(lipgloss.Color(bgColor)).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).MarginRight(1).Render(a.Initials))
	}

	var tags []string
	for _, t := range task.Tags {
		fgColor := t.TagFg
		if fgColor == "" || fgColor == t.TagBg {
			fgColor = "#ffffff"
		}
		tags = append(tags, lipgloss.NewStyle().Bold(true).Background(lipgloss.Color(t.TagBg)).Foreground(lipgloss.Color(fgColor)).Padding(0, 1).MarginRight(1).Render(t.Name))
	}

	subtaskInfo := ""
	if task.SubTasksCount > 0 {
		subtaskInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MarginRight(1).Render(fmt.Sprintf("📋 %d", task.SubTasksCount))
	}

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Left, subtaskInfo, lipgloss.NewStyle().Width(columnWidth-8-lipgloss.Width(subtaskInfo)).Align(lipgloss.Right).Render(lipgloss.JoinHorizontal(lipgloss.Right, tags...)))
	wrappedTaskName := lipgloss.NewStyle().Width(columnWidth - 8).Height(3).Render(task.Name)
	taskNameStyle := lipgloss.NewStyle().Bold(true).MaxWidth(columnWidth - 5)
	listNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MaxWidth(columnWidth - 5)

	content := lipgloss.JoinVertical(lipgloss.Left, lipgloss.JoinHorizontal(lipgloss.Left, assignees...), listNameStyle.Render("📁 "+task.List.Name), taskNameStyle.Render(wrappedTaskName), bottomRow)
	return style.Render(content)
}

func (m HomeModel) viewModal() string {
	task := m.modalTask
	statusColor := task.Status.Color
	modalStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(statusColor)).Padding(1, 2).Width(m.width - 3).Align(lipgloss.Left)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(statusColor)).Width(m.width - 6).Align(lipgloss.Left)
	listStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MarginBottom(1)
	header := lipgloss.JoinVertical(lipgloss.Left, titleStyle.Render(task.Name), listStyle.Render("📁 "+task.List.Name))

	var metaInfo []string
	metaInfo = append(metaInfo, lipgloss.NewStyle().Background(lipgloss.Color(statusColor)).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).MarginRight(2).Render(task.Status.Status))
	for _, a := range task.Assignees {
		bgColor := a.Color
		if bgColor == "" {
			bgColor = "#888888"
		}
		metaInfo = append(metaInfo, lipgloss.NewStyle().Bold(true).Background(lipgloss.Color(bgColor)).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).MarginRight(1).Render(a.Initials))
	}
	metaRow := lipgloss.NewStyle().Width(m.width - 6).Render(lipgloss.JoinHorizontal(lipgloss.Left, metaInfo...))

	commentsSectionStyle := lipgloss.NewStyle().Width(m.width-9).Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#666666")).Padding(0, 1)
	commentHeaderStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#888888")).MarginBottom(1)
	comments := commentsSectionStyle.Render(lipgloss.JoinVertical(lipgloss.Left, commentHeaderStyle.Render("Comments"), m.commentsViewport.View()))

	modalContent := lipgloss.JoinVertical(lipgloss.Left, header, metaRow, m.contentViewport.View(), comments)
	return modalStyle.Render(modalContent)
}

func RenderCommentText(comment []clients.CommentText) string {
	var sb strings.Builder
	for _, part := range comment {
		switch part.Type {
		case "bookmark":
			url, _ := part.Bookmark["url"].(string)
			title := url
			if raw, ok := part.Attributes["raw"].(string); ok && raw != "" {
				if decoded, err := decodeBookmarkRaw(raw); err == nil && decoded.Title != "" {
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
			if badge, ok := part.Attributes["badge-class"].(string); ok && badge != "" {
				badgeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fff")).Background(lipgloss.Color("#FF5F87")).Padding(0, 1)
				sb.WriteString(badgeStyle.Render(part.Text) + " ")
			} else {
				sb.WriteString(lipgloss.NewStyle().Render(part.Text))
			}
		default:
			sb.WriteString(lipgloss.NewStyle().Render(part.Text))
		}
	}
	return sb.String()
}

type bookmarkRaw struct {
	Title string `json:"title"`
}

func decodeBookmarkRaw(raw string) (bookmarkRaw, error) {
	decoded := bookmarkRaw{}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return decoded, err
	}
	var m map[string]interface{}
	if json.Unmarshal(data, &m) == nil {
		if preview, ok := m["preview"].(map[string]interface{}); ok {
			if title, ok := preview["title"].(string); ok {
				decoded.Title = title
			}
		}
	}
	return decoded, nil
}

func (m HomeModel) handleInputActiveEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var updatedViewInput textinput.Model
	var cmd tea.Cmd
	if msg.Type == tea.KeyEnter && m.viewInput.Value() != "" {
		config := clients.GetConfig()
		config.ViewId = m.viewInput.Value()
		if err := clients.SaveConfig(config); err != nil {
			return m, tea.Quit
		}
		clients.ClearCache()
		return NewHomeModel(), nil
	}
	updatedViewInput, cmd = m.viewInput.Update(msg)
	m.viewInput = updatedViewInput
	return m, cmd
}
func (m HomeModel) handleLoadTasksEvent() (tea.Model, tea.Cmd) {
	m.loading = true
	return m, tea.Batch(fetchTasks, m.spinner.Tick)
}

func (m HomeModel) handleWindowSizeEvent(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width, m.height = msg.Width, msg.Height
	m.wndX, m.wndY = calculateWindowDimensions(m.width, m.height)
	if m.showModal {
		m.contentViewport.Width = m.width - 9
		m.contentViewport.Height = m.height - 20
		m.commentsViewport.Width = m.width - 11
		m.commentsViewport.Height = 6
	}
	return m, nil
}

func (m HomeModel) handleSpinnerTickEvent(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var updatedSpinner spinner.Model
	var cmd tea.Cmd
	if m.loading {
		updatedSpinner, cmd = m.spinner.Update(msg)
		m.spinner = updatedSpinner
		return m, cmd
	}
	return m, cmd
}

func (m HomeModel) handleTasksLoadedEvent(msg taskLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, tea.Quit
	}
	m.loading = false
	m.processTasks(msg.tasks)
	return m, nil
}

func (m HomeModel) handleKeyEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showModal {
		return m.handleKeyModalEvent(msg)
	}
	return m.handleKeyMainEvent(msg)
}

func (m HomeModel) handleKeyModalEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "enter":
		m.showModal = false
		m.modalTask = nil
	case "up":
		m.contentViewport.ScrollUp(1)
	case "down":
		m.contentViewport.ScrollDown(1)
	case "j":
		m.commentsViewport.ScrollUp(1)
	case "k":
		m.commentsViewport.ScrollDown(1)
	}
	return m, nil
}

func (m HomeModel) handleKeyMainEvent(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "r":
		clients.ClearViewTasksCache()
		m.loading = true
		return m, tea.Batch(fetchTasks, m.spinner.Tick)
	case "enter":
		if col, ok := m.columns[m.states[m.selectedColumn]]; ok && m.selectedTask < len(col.tasks) {
			task := col.tasks[m.selectedTask]
			config := clients.GetConfig()
			client := clients.NewClickupClient(config.ClickupToken, config.TeamId)
			t, err := client.GetTask(task.Id)
			if err != nil {
				return m, nil
			}
			comments, err := client.GetTaskComments(task.Id)
			if err != nil {
				comments = []clients.Comment{}
			}
			t.Comments = comments
			m.modalTask = &t
			m.showModal = true

			m.contentViewport = viewport.New(m.width-9, m.height-20)
			m.commentsViewport = viewport.New(m.width-11, 6)

			var renderedMarkdown string
			if m.modalTask.Description != "" {
				rendered, err := glamour.Render(m.modalTask.Description, "dark")
				if err == nil {
					renderedMarkdown = rendered
				} else {
					renderedMarkdown = m.modalTask.Description
				}
			}
			m.contentViewport.SetContent(lipgloss.NewStyle().Width(m.width - 9).Render(renderedMarkdown))

			var commentsContent []string
			for _, comment := range m.modalTask.Comments {
				color := comment.User.Color
				if color == "" {
					color = "#888888"
				}
				commentHeader := lipgloss.JoinHorizontal(lipgloss.Left,
					lipgloss.NewStyle().Bold(true).Background(lipgloss.Color(color)).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).MarginRight(1).Render(comment.User.Initials),
					lipgloss.NewStyle().Width(m.width-23).Foreground(lipgloss.Color("#666666")).Render(comment.User.Username),
					lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(shared.ToElapsedTime(comment.Date)),
				)
				commentLine := lipgloss.NewStyle().Width(m.width - 16).Render(RenderCommentText(comment.Comment))
				commentsContent = append(commentsContent, commentHeader, commentLine)
			}
			m.commentsViewport.SetContent(lipgloss.JoinVertical(lipgloss.Left, commentsContent...))
		}
	case "left":
		if m.selectedColumn > 0 {
			m.selectedColumn--
			if m.selectedColumn < m.offsetX {
				m.offsetX--
			}
			if _, ok := m.columns[m.states[m.selectedColumn]]; ok {
				m.selectedTask = m.columns[m.states[m.selectedColumn]].offsetY
			} else {
				m.selectedTask = 0
			}
		}
	case "right":
		if m.selectedColumn < len(m.states)-1 {
			m.selectedColumn++
			if m.selectedColumn >= m.offsetX+m.wndX {
				m.offsetX++
			}
			if _, ok := m.columns[m.states[m.selectedColumn]]; ok {
				m.selectedTask = m.columns[m.states[m.selectedColumn]].offsetY
			} else {
				m.selectedTask = 0
			}
		}
	case "up":
		if m.selectedTask > 0 {
			m.selectedTask--
			if col, ok := m.columns[m.states[m.selectedColumn]]; ok {
				if m.selectedTask < col.offsetY {
					col.offsetY--
					m.columns[m.states[m.selectedColumn]] = col
				}
			}
		}
	case "down":
		if col, ok := m.columns[m.states[m.selectedColumn]]; ok {
			if m.selectedTask < len(col.tasks)-1 {
				m.selectedTask++
				if m.selectedTask >= col.offsetY+m.wndY {
					col.offsetY++
					m.columns[m.states[m.selectedColumn]] = col
				}
			}
		}
	}
	return m, nil
}
