package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/mceck/clickup-tui/internal/shared"
)

var cache = ClickupCache{}

type ClickupClient struct {
	BaseURL    string
	HTTPClient *http.Client
	APIToken   string
	TeamID     string
}

type TaskResponse struct {
	Tasks    []Task `json:"tasks"`
	LastPage bool   `json:"last_page"`
}

func SaveCache() error {
	cache.BumpExpiry()
	file, err := json.MarshalIndent(cache, "", " ")
	if err != nil {
		return err
	}
	// create directory if it doesn't exist
	dirPath := os.ExpandEnv("$HOME/.config/clickup-tui")
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}
	err = os.WriteFile(dirPath+"/cache.json", file, 0644)
	if err != nil {
		return err
	}
	return nil
}

func NewClickupClient(apiToken string, teamId string) *ClickupClient {
	// read from file
	dirPath := os.ExpandEnv("$HOME/.config/clickup-tui")
	file, err := os.ReadFile(dirPath + "/cache.json")
	if err == nil {
		err = json.Unmarshal(file, &cache)
		if err != nil {
			fmt.Println("Error reading cache file:", err)
		}
	}

	return &ClickupClient{
		BaseURL:    "https://api.clickup.com",
		HTTPClient: &http.Client{},
		APIToken:   apiToken,
		TeamID:     teamId,
	}
}
func (c *ClickupClient) getTasksPage(page int, qs string) (TaskResponse, error) {
	url := fmt.Sprintf("%s/api/v2/team/%s/task?%s&page=%d", c.BaseURL, c.TeamID, qs, page)
	req, err := http.NewRequest("GET", url, nil)
	data := TaskResponse{}
	if err != nil {
		return data, err
	}
	req.Header.Set("Authorization", c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return data, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return data, fmt.Errorf("failed to get tasks: %s", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(responseBody, &data)
	if err != nil {
		return data, err
	}
	return data, nil
}

func (c *ClickupClient) getViewPage(page int, viewId string) (TaskResponse, error) {
	url := fmt.Sprintf("%s/api/v2/view/%s/task?page=%d", c.BaseURL, viewId, page)
	req, err := http.NewRequest("GET", url, nil)
	data := TaskResponse{}
	if err != nil {
		return data, err
	}
	req.Header.Set("Authorization", c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return data, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return data, fmt.Errorf("failed to get tasks: %s", resp.Status)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(responseBody, &data)
	if err != nil {
		return data, err
	}
	return data, nil
}

func (c *ClickupClient) GetTask(taskId string) (Task, error) {
	if cache.IsExpired() {
		cache.Clear()
	}
	if cache.TaskByID != nil {
		if task, ok := cache.TaskByID[taskId]; ok {
			return task, nil
		}
	} else {
		cache.TaskByID = make(map[string]Task)
	}
	url := fmt.Sprintf("%s/api/v2/task/%s?include_markdown_description=true", c.BaseURL, taskId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Task{}, err
	}
	req.Header.Set("Authorization", c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Task{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Task{}, fmt.Errorf("failed to get task: %s", resp.Status)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Task{}, err
	}
	var task Task
	err = json.Unmarshal(responseBody, &task)
	if err != nil {
		return Task{}, err
	}
	cache.TaskByID[taskId] = task
	SaveCache()
	return task, nil
}

// GetTaskComments fetches comments for a given task ID.
func (c *ClickupClient) GetTaskComments(taskId string) ([]Comment, error) {
	if cache.IsExpired() {
		cache.Clear()
	}
	if cache.CommentsByTaskID == nil {
		cache.CommentsByTaskID = make(map[string][]Comment)
	}
	if comments, ok := cache.CommentsByTaskID[taskId]; ok {
		return comments, nil
	}
	url := fmt.Sprintf("%s/api/v2/task/%s/comment", c.BaseURL, taskId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get comments: %s", resp.Status)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data struct {
		Comments []Comment `json:"comments"`
	}
	err = json.Unmarshal(responseBody, &data)
	if err != nil {
		return nil, err
	}
	cache.CommentsByTaskID[taskId] = data.Comments
	SaveCache()
	return data.Comments, nil
}

func (c *ClickupClient) GetTimesheetTasks() ([]Task, error) {
	if cache.IsExpired() {
		cache.Clear()
	}
	if cache.TimesheetTasks != nil {
		return cache.TimesheetTasks, nil
	}
	tasks := []Task{}
	page := 0
	for {
		res, err := c.getTasksPage(page, "tags[]=timesheet")
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, res.Tasks...)
		if res.LastPage {
			break
		}
		page++
	}

	cache.TimesheetTasks = tasks
	SaveCache()
	return tasks, nil
}

func (c *ClickupClient) GetViewTasks(viewId string) ([]Task, error) {
	if cache.IsExpired() {
		cache.Clear()
	}
	if cache.ViewTasks != nil {
		return cache.ViewTasks, nil
	}
	tasks := []Task{}
	page := 0
	for {
		res, err := c.getViewPage(page, viewId)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, res.Tasks...)
		if res.LastPage {
			break
		}
		page++
	}

	cache.ViewTasks = tasks
	SaveCache()
	return tasks, nil
}

type TsResponse struct {
	Data []TimeEntry
}

func (c *ClickupClient) GetTimesheetsEntries(userId string) ([]TimeEntry, error) {
	if cache.IsExpired() {
		cache.Clear()
	}
	if cache.TimeEntries != nil {
		return cache.TimeEntries, nil
	}
	url := fmt.Sprintf("%s/api/v2/team/%s/time_entries?assignee=%s", c.BaseURL, c.TeamID, userId)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err

	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get timesheets: %s", resp.Status)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data := TsResponse{}
	err = json.Unmarshal(responseBody, &data)
	if err != nil {
		return nil, err
	}
	cache.TimeEntries = data.Data
	SaveCache()
	return data.Data, nil
}

func (c *ClickupClient) DeleteTimeEntry(taskId string, entryId string) error {
	url := fmt.Sprintf("%s/api/v2/task/%s/time/%s", c.BaseURL, taskId, entryId)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete timesheet: %s %s", entryId, resp.Status)
	}
	return nil
}

func (c *ClickupClient) CreateTimeEntry(taskId string, start time.Time, duration int, userId string) error {
	url := fmt.Sprintf("%s/api/v2/task/%s/time", c.BaseURL, taskId)
	reqBody := map[string]interface{}{
		"start": start.Unix() * 1000,
		"time":  duration,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, io.NopCloser(bytes.NewBuffer(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create time entry: %s", resp.Status)
	}
	return nil
}

func (c *ClickupClient) UpdateTracking(userId string, taskId string, day time.Time, hours float64) error {
	entries, err := c.GetTimesheetsEntries(userId)
	if err != nil {
		return err
	}
	cache.TimeEntries = nil
	dateEntries := []TimeEntry{}
	for _, entry := range entries {
		if entry.Task.(map[string]interface{})["id"] != taskId {
			continue
		}
		start := shared.ToDate(entry.Start)
		if start.Format("2006-01-02") == day.Format("2006-01-02") {
			dateEntries = append(dateEntries, entry)
		}
	}
	total := 0.0
	toDelete := []TimeEntry{}
	for _, entry := range dateEntries {
		h := shared.ToHours(entry.Duration)
		if total+h > hours {
			toDelete = append(toDelete, entry)
			continue
		}
		total += h
	}
	if len(toDelete) > 0 {
		for _, entry := range toDelete {
			err = c.DeleteTimeEntry(taskId, entry.Id)
			if err != nil {
				return err
			}
		}
	}

	if total == hours {
		return nil
	}

	diff := hours - total
	duration := int(diff * 60 * 60 * 1000)
	start := time.Date(day.Year(), day.Month(), day.Day(), 6, 0, 0, 0, day.Location())
	err = c.CreateTimeEntry(taskId, start, duration, userId)
	if err != nil {
		return err
	}

	c.GetTimesheetsEntries(userId)
	return nil
}

func ClearCache() {
	cache.Clear()
}

func ClearTimeentriesCache() {
	cache.TimeEntries = nil
	SaveCache()
}

func ClearTimesheetTasksCache() {
	cache.TimesheetTasks = nil
	SaveCache()
}

func ClearViewTasksCache() {
	cache.ViewTasks = nil
	SaveCache()
}
