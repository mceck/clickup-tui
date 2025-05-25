package clients

type TimeEntry struct {
	Id       string      `json:"id"`
	Task     interface{} `json:"task"`
	Duration string      `json:"duration"`
	Start    string      `json:"start"`
	End      string      `json:"end"`
}

type Status struct {
	Id         string `json:"id"`
	Status     string `json:"status"`
	Color      string `json:"color"`
	Orderindex int    `json:"orderindex"`
}

type User struct {
	Id             any    `json:"id"`
	Username       string `json:"username"`
	Initials       string `json:"initials"`
	Color          string `json:"color"`
	ProfilePicture string `json:"profilePicture"`
}

type List struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Tag struct {
	Name  string `json:"name"`
	TagBg string `json:"tag_bg"`
	TagFg string `json:"tag_fg"`
}

type Task struct {
	Id            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"markdown_description"`
	Status        Status `json:"status"`
	Assignees     []User `json:"assignees"`
	List          List   `json:"list"`
	Tags          []Tag  `json:"tags"`
	SubTasksCount int    `json:"subtasks_count"`
}

// CommentText represents a part of a comment, with text and attributes.
type CommentText struct {
	Text       string                 `json:"text"`
	Type       string                 `json:"type,omitempty"`
	Bookmark   map[string]interface{} `json:"bookmark,omitempty"`
	Attributes map[string]interface{} `json:"attributes"`
}

// Comment represents a comment on a task.
type Comment struct {
	Id            string        `json:"id"`
	Comment       []CommentText `json:"comment"`
	CommentText   string        `json:"comment_text"`
	User          User          `json:"user"`
	Assignee      interface{}   `json:"assignee"`
	GroupAssignee interface{}   `json:"group_assignee"`
	Reactions     []interface{} `json:"reactions"`
	Date          string        `json:"date"`
	ReplyCount    int           `json:"reply_count"`
}
