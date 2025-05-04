package app

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mceck/clickup-tui/internal/ui/views"
)

// KeyMap definisce i tasti per le funzionalità dell'applicazione
type KeyMap struct {
	Quit     key.Binding
	Help     key.Binding
	Back     key.Binding
	Enter    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
}

// ShortHelp restituisce i tasti di scelta rapida più importanti
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp restituisce tutti i tasti di scelta rapida
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help, k.Quit},
		{k.Back, k.Enter},
		{k.Tab, k.ShiftTab},
	}
}

var DefaultKeyMap = KeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "esci"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "aiuto"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "indietro"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "seleziona"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "avanti"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "indietro"),
	),
}

// Page rappresenta le diverse pagine dell'applicazione
type Page int

const (
	HomeView Page = iota
	SettingsView
	HelpView
)

// Model è il modello principale dell'applicazione
type Model struct {
	keys         KeyMap
	help         help.Model
	currentPage  Page
	homeView     views.HomeModel
	settingsView views.SettingsModel
	helpView     views.HelpModel
	width        int
	height       int
}

// New crea una nuova istanza del modello principale
func New() Model {
	h := help.New()
	h.ShowAll = false

	return Model{
		keys:         DefaultKeyMap,
		help:         h,
		currentPage:  HomeView,
		homeView:     views.NewHomeModel(),
		settingsView: views.NewSettingsModel(),
		helpView:     views.NewHelpModel(),
	}
}

// Init inizializza il modello
func (m Model) Init() tea.Cmd {
	return nil
}

// Update aggiorna il modello in base ai messaggi ricevuti
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.currentPage = HelpView
			return m, nil
		case key.Matches(msg, m.keys.Back):
			m.currentPage = HomeView
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
	}

	// Gestisce gli aggiornamenti in base alla pagina corrente
	switch m.currentPage {
	case HomeView:
		m.homeView, cmd = m.homeView.Update(msg)
		cmds = append(cmds, cmd)
	case SettingsView:
		m.settingsView, cmd = m.settingsView.Update(msg)
		cmds = append(cmds, cmd)
	case HelpView:
		m.helpView, cmd = m.helpView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renderizza l'interfaccia utente
func (m Model) View() string {
	switch m.currentPage {
	case HomeView:
		return m.homeView.View() + "\n" + m.help.View(m.keys)
	case SettingsView:
		return m.settingsView.View() + "\n" + m.help.View(m.keys)
	case HelpView:
		return m.helpView.View() + "\n" + m.help.View(m.keys)
	default:
		return "Pagina non trovata"
	}
}

// NewProgram crea una nuova istanza dell'applicazione
func NewProgram() *tea.Program {
	model := New()
	return tea.NewProgram(model, tea.WithMouseCellMotion(), tea.WithAltScreen())
}
