# clickup-tui

A modern terminal user interface (TUI) client for [ClickUp](https://clickup.com/), built in Go using the Charmbracelet Bubble Tea ecosystem. Manage your ClickUp tasks, track time, and configure your workspace directly from your terminal.

## Features

- **Kanban Board:** View and manage your ClickUp tasks in a kanban-style board (Home View).
- **Timesheet Tracking:** Log and edit time spent on tasks for each day of the week (Timesheet View).
- **Settings UI:** Configure your ClickUp API credentials and workspace settings from within the TUI (Settings View).
- **Fast and Responsive:** Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lipgloss](https://github.com/charmbracelet/lipgloss).

## Prerequisites

- [Go 1.24.2](https://golang.org/dl/) or higher
- A ClickUp account and API token ([see ClickUp API docs](https://clickup.com/api))

## Installation

You can install the latest version directly using Go:

```sh
go install github.com/mceck/clickup-tui@latest
```

This will place the `clickup-tui` binary in your `$GOBIN` (usually `$HOME/go/bin`). Make sure this directory is in your `PATH`.

[GitHub Repository](https://github.com/mceck/clickup-tui)

## Configuration

On first run, or by selecting the Settings view (`?`), you will be prompted to enter your ClickUp credentials:
- **ClickUp API Token**
- **Team ID**
- **User ID**
- **View ID** (for the kanban board)

These are saved in a JSON file at:
```
$HOME/.config/clickup-tui/config.json
```
Example config:
```json
{
  "clickup_token": "your-token-here",
  "team_id": "your-team-id",
  "user_id": "your-user-id",
  "view_id": "your-view-id"
}
```

## Usage

- **Navigation:**
  - `Tab/Esc`: Switch between Home and Timesheet views
  - `?`: Open Settings view
  - `Ctrl+C` or `q`: Quit
  - `r` refresh
- **Home View:**
  - Arrow keys to move between columns and tasks
  - Enter a View ID if prompted
  - Press Enter to view task details and comments
- **Timesheet View:**
  - Arrow keys to move between tasks and days
  - Enter to edit hours

## Development

- Format/lint: `go fmt ./...`
- Clean build artifacts: `go clean`
