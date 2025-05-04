package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	ui "github.com/mceck/clickup-tui/internal/ui/styles"
)

// SettingsModel è il modello per la vista impostazioni
type SettingsModel struct {
	username   textinput.Model
	theme      textinput.Model
	focusIndex int
	inputs     []textinput.Model
	width      int
	height     int
}

// NewSettingsModel crea una nuova istanza del modello impostazioni
func NewSettingsModel() SettingsModel {
	// Creazione del campo username
	username := textinput.New()
	username.Placeholder = "Inserisci il tuo nome utente"
	username.Focus()
	username.CharLimit = 32
	username.Width = 30

	// Creazione del campo tema
	theme := textinput.New()
	theme.Placeholder = "Seleziona il tema (default, dark, light)"
	theme.CharLimit = 10
	theme.Width = 30

	// Raccoglie tutti gli input
	inputs := []textinput.Model{username, theme}

	return SettingsModel{
		username:   username,
		theme:      theme,
		inputs:     inputs,
		focusIndex: 0,
	}
}

// Init inizializza il modello
func (m SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update aggiorna il modello in base ai messaggi ricevuti
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "enter", "up", "down":
			// Gestisce la navigazione tra i campi
			s := msg.String()

			// Ciclo attraverso i campi quando si preme tab
			if s == "tab" || s == "enter" || s == "down" {
				m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			} else if s == "shift+tab" || s == "up" {
				m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
			}

			// Aggiorna lo stato di focus per tutti gli input
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					// Questo campo deve essere focalizzato
					cmds = append(cmds, m.inputs[i].Focus())
					m.inputs[i] = m.inputs[i]
					cmds = append(cmds, m.inputs[i].Focus())
				} else {
					// Questo campo deve perdere il focus
					m.inputs[i].Blur()
					m.inputs[i].Blur()
				}
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Aggiorna il campo attualmente in focus
	cmd := m.updateInputs(msg)
	return m, cmd
}

// updateInputs aggiorna l'input correntemente in focus
func (m *SettingsModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	// Aggiorna solo l'input che ha il focus
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)

	// Sincronizza lo stato del modello con gli input aggiornati
	m.username = m.inputs[0]
	m.theme = m.inputs[1]

	return cmd
}

// View renderizza l'interfaccia utente
func (m SettingsModel) View() string {
	if m.width == 0 {
		return "Inizializzazione..."
	}

	// Titolo della sezione
	title := ui.TitleStyle.Render("Impostazioni")

	// Prepara le label
	labelWidth := 15
	usernameLabel := ui.SubtitleStyle.Width(labelWidth).Render("Username:")
	themeLabel := ui.SubtitleStyle.Width(labelWidth).Render("Tema:")

	// Prepara gli input
	usernameInput := m.username.View()
	themeInput := m.theme.View()

	// Formatta ogni riga con label e input
	usernameRow := lipgloss.JoinHorizontal(lipgloss.Left, usernameLabel, usernameInput)
	themeRow := lipgloss.JoinHorizontal(lipgloss.Left, themeLabel, themeInput)

	// Unisci tutte le righe
	formContent := lipgloss.JoinVertical(
		lipgloss.Left,
		usernameRow,
		"",
		themeRow,
	)

	// Crea un box per il form
	formBox := ui.PanelStyle.Width(m.width - 4).Render(formContent)

	// Crea un footer con informazioni di aiuto
	footer := ui.InfoNotificationStyle.Render("Tab/Shift+Tab per navigare • Enter per confermare • Esc per tornare")

	// Unisci tutto il contenuto
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"\n",
		formBox,
		"\n",
		footer,
	)
}
