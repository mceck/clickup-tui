package models

// AppState contiene lo stato globale dell'applicazione
type AppState struct {
	// Aggiungi qui i dati condivisi tra le diverse viste
	Username      string
	IsLoggedIn    bool
	ActiveTheme   string
	Notifications []Notification
}

// Notification rappresenta una notifica nell'applicazione
type Notification struct {
	ID      int
	Message string
	Type    NotificationType
	Read    bool
}

// NotificationType rappresenta il tipo di notifica
type NotificationType int

const (
	Info NotificationType = iota
	Warning
	Error
	Success
)

// NewAppState crea una nuova istanza dello stato dell'applicazione
func NewAppState() *AppState {
	return &AppState{
		Username:      "",
		IsLoggedIn:    false,
		ActiveTheme:   "default",
		Notifications: []Notification{},
	}
}

// AddNotification aggiunge una nuova notifica
func (s *AppState) AddNotification(message string, nType NotificationType) {
	id := len(s.Notifications) + 1
	notification := Notification{
		ID:      id,
		Message: message,
		Type:    nType,
		Read:    false,
	}
	s.Notifications = append(s.Notifications, notification)
}

// MarkNotificationAsRead marca una notifica come letta
func (s *AppState) MarkNotificationAsRead(id int) {
	for i, notification := range s.Notifications {
		if notification.ID == id {
			s.Notifications[i].Read = true
			break
		}
	}
}
