package clients

import "time"

type ClickupCache struct {
	TimesheetTasks   []Task               `json:"timesheet_tasks"`
	TimeEntries      []TimeEntry          `json:"time_entries"`
	ViewTasks        []Task               `json:"view_tasks"`
	TaskByID         map[string]Task      `json:"task_by_id"`
	CommentsByTaskID map[string][]Comment `json:"comments_by_task_id"`
	ExpiredAt        int64                `json:"expired_at"`
}

func (c *ClickupCache) IsExpired() bool {
	if c.ExpiredAt == 0 {
		return true
	}
	return c.ExpiredAt < time.Now().Unix()
}

func (c *ClickupCache) BumpExpiry() {
	c.ExpiredAt = time.Now().Add(time.Hour * 1).Unix()
}

func (c *ClickupCache) Clear() {
	c.TimesheetTasks = nil
	c.TimeEntries = nil
	c.ViewTasks = nil
	c.TaskByID = make(map[string]Task)
	c.CommentsByTaskID = make(map[string][]Comment)
}
