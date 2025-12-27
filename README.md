# ScrumBoard - Jira Sprint Dashboard

A beautiful, real-time Sprint Dashboard built with Go, Templ, and TailwindCSS that connects to Jira.

## Features

### 📊 Sprint Dashboard
- 📊 Real-time sprint statistics (total stories, points, completed, remaining)
- 👥 Team workload visualization per assignee
- 📋 Kanban-style board with To Do, In Progress, Done columns
- 🔄 Auto-refresh from Jira API
- 🎨 Modern glassmorphism UI with animations

### 🐛 Defect Dashboard (NEW!)
- 🐛 Comprehensive defect tracking for current sprint
- 📊 Defect statistics (total, open, in-progress, resolved, closed)
- 🎯 Severity breakdown (Critical, High, Medium, Low)
- ⚡ Priority distribution (Blocker, Critical, Major, Minor, Trivial)
- 👥 Defects by assignee with workload metrics
- 📈 Resolution metrics (avg resolution time, aging defects)
- 🔴 Defect board with Open → In Progress → Resolved columns
- 🌍 Environment tracking (Production, Staging, Dev)
- ⏱️ Age tracking and resolution time analysis

## Tech Stack
- **Backend**: Go + Fiber v2 (web framework)
- **Templates**: Templ (type-safe Go templates)
- **Styling**: TailwindCSS (compiled to static CSS)
- **API**: Jira REST API (Agile + v3)

## Project Structure
```
ScrumBoard/
├── cmd/main.go                      # Entry point
├── internal/
│   ├── handlers/                    # HTTP route handlers
│   │   ├── dashboard.go             # Sprint dashboard handler
│   │   └── defect.go                # Defect dashboard handler
│   ├── models/                      # Data models
│   │   ├── sprint.go                # Story, User, Sprint, SprintStats, UserStats
│   │   └── defect.go                # Defect, DefectStats, DefectDashboardData
│   └── services/                    # Business logic
│       ├── jira.go                  # Jira API client (stories + defects)
│       ├── dashboard.go             # Sprint stats calculation
│       └── defect_dashboard.go      # Defect stats calculation
├── templates/
│   ├── components/                  # Reusable UI components
│   │   ├── stats_card.templ         # Stats cards
│   │   ├── story_card.templ         # Story cards
│   │   ├── defect_card.templ        # Defect cards
│   │   ├── defect_stats.templ       # Defect statistics
│   │   ├── defect_board.templ       # Defect kanban board
│   │   └── ...
│   ├── dashboard.templ              # Sprint dashboard page
│   ├── defect_dashboard.templ       # Defect dashboard page
│   └── layout.templ                 # Base layout
├── static/css/styles.css            # TailwindCSS output
├── .env                             # Environment config
└── go.mod
```

## Environment Variables (.env)
```
JIRA_BASE_URL=https://your-domain.atlassian.net
JIRA_EMAIL=your-email@company.com
JIRA_API_TOKEN=your-jira-api-token
JIRA_BOARD_ID=your-board-id
PORT=3000
```

## Jira Configuration
- **Board ID**: Found in Jira board URL (/boards/6991)
- **API Token**: Generate from https://id.atlassian.com/manage-profile/security/api-tokens
- **Custom Field**: Story Points uses customfield_10028 (may vary per Jira instance)

## Status Categories (from Jira)
- `done` → Completed (green) - Accepted, Done, Closed
- `indeterminate` → In Progress (blue)
- `new` → To Do (gray) - Backlog, Open

## Running the Application
```bash
templ generate                           # Generate templates
go build -o scrum-dashboard ./cmd/main.go  # Build
./scrum-dashboard                         # Run at http://localhost:3000
```

## Common Issues
1. **Story Points 0**: Check customfield_10028 is correct for your Jira
2. **Wrong issues**: Verify JIRA_BOARD_ID matches your board
3. **401 Error**: Check JIRA_EMAIL and JIRA_API_TOKEN in .env

## Application Routes
- `/` - Sprint Dashboard (main page)
- `/defects` - Defect Dashboard
- `/api/refresh` - Refresh sprint data (HTMX)
- `/defects/api/refresh` - Refresh defect data (HTMX)

## Jira API Endpoints Used
- GET /rest/agile/1.0/board/{boardID}/sprint?state=active
- GET /rest/agile/1.0/board/{boardID}/issue?fields=... (for stories)
- GET /rest/agile/1.0/board/{boardID}/issue?jql=type=Bug&fields=... (for defects)
