package clients

import (
	"encoding/json"
	"os"
)

type Config struct {
	ClickupToken    string `json:"clickup_token"`
	TeamId          string `json:"team_id"`
	UserId          string `json:"user_id"`
	ViewId          string `json:"view_id"`
	InitialView     string `json:"initial_view"` // "kanban", "timesheet"
	TimesheetFilter string `json:"timesheet_filter"`
}

var config *Config

func GetConfig() Config {
	if config != nil {
		return *config
	}
	// read from file
	dirPath := os.ExpandEnv("$HOME/.config/clickup-tui")
	file, err := os.ReadFile(dirPath + "/config.json")
	c := Config{}
	if err == nil {
		err = json.Unmarshal(file, &c)
		if err == nil {
		}
	}
	config = &c
	return c
}

func SaveConfig(c Config) error {
	file, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}
	// create directory if it doesn't exist
	dirPath := os.ExpandEnv("$HOME/.config/clickup-tui")
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(dirPath+"/config.json", file, 0644)
	if err != nil {
		return err
	}
	config = &c
	return nil
}
