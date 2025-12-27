# ScrumBoard - AI Context

## What is this project?
A Jira Sprint Dashboard that shows real-time sprint data with a beautiful UI.

## Current Jira Configuration
- **Board ID**: 6991 (PMOB project)
- **Base URL**: https://miggbo.atlassian.net
- **Story Points Field**: customfield_10028

## Key Files to Know

### Models (internal/models/sprint.go)
- `Story`: Issue with Key, Title, Status, StatusCategory, StoryPoints, Assignee
- `User`: Name, Email, AvatarURL
- `Sprint`: ID, Name, State, StartDate, EndDate
- `SprintStats`: TotalStories, TotalPoints, Completed, InProgress, Remaining
- `UserStats`: Per-user assigned/completed stories and points

### Jira Service (internal/services/jira.go)
- `GetActiveSprint(boardID)` - Fetches active sprint from board
- `GetSprintIssues(boardID, sprintID)` - Fetches all issues for board
- Uses Basic Auth with email:api_token
- Parses customfield_10028 for story points
- Parses statusCategory.key for done/indeterminate/new grouping

### Dashboard Service (internal/services/dashboard.go)
- `CalculateSprintStats(stories)` - Calculates totals and completion
- `CalculateUserStats(stories)` - Per-user breakdown
- `GroupStoriesByStatus(stories)` - Groups into todo, inProgress, done
- Uses StatusCategory (not Status name) for grouping

### Handler (internal/handlers/dashboard.go)
- GET `/` - Main dashboard page
- GET `/api/refresh` - HTMX partial refresh

### Templates (templates/)
- `pages/dashboard.templ` - Main layout
- `components/stats_cards.templ` - Sprint statistics cards
- `components/story_card.templ` - Individual story card with status badge
- `components/user_stats.templ` - Team workload section

## Status Mapping
```
StatusCategory "done"          → Completed (Accepted, Done, Closed)
StatusCategory "indeterminate" → In Progress
StatusCategory "new"           → To Do (Backlog, Open)
```

## How to Run
```bash
templ generate
go build -o scrum-dashboard ./cmd/main.go
./scrum-dashboard
# Open http://localhost:3000
```

## Debugging Tips
- Check .env for JIRA_BOARD_ID
- Use curl to test Jira API directly
- Story points field ID varies per Jira instance
