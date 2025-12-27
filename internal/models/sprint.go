package models

// Board represents a Jira board
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // scrum, kanban
}

// Story represents a user story/issue from Jira
type Story struct {
	ID             string   `json:"id"`
	Key            string   `json:"key"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Status         string   `json:"status"`         // Actual status name: Accepted, In Progress, Backlog, etc.
	StatusCategory string   `json:"statusCategory"` // Category: done, indeterminate, new
	StoryPoints    int      `json:"storyPoints"`
	Priority       string   `json:"priority"` // High, Medium, Low
	Assignee       User     `json:"assignee"`
	Labels         []string `json:"labels"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
}

// User represents a team member
type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarUrl"`
}

// Sprint represents a Jira sprint
type Sprint struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"` // active, closed, future
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Goal      string `json:"goal"`
}

// SprintStats holds calculated statistics for a sprint
type SprintStats struct {
	TotalStories      int `json:"totalStories"`
	TotalPoints       int `json:"totalPoints"`
	CompletedPoints   int `json:"completedPoints"`
	InProgressPoints  int `json:"inProgressPoints"`
	RemainingPoints   int `json:"remainingPoints"`
	CompletedStories  int `json:"completedStories"`
	InProgressStories int `json:"inProgressStories"`
	TodoStories       int `json:"todoStories"`
}

// UserStats holds story statistics per user
type UserStats struct {
	User            User `json:"user"`
	AssignedStories int  `json:"assignedStories"`
	AssignedPoints  int  `json:"assignedPoints"`
	CompletedPoints int  `json:"completedPoints"`
}

// DashboardData holds all data needed for the dashboard
type DashboardData struct {
	Sprint      Sprint      `json:"sprint"`
	Stats       SprintStats `json:"stats"`
	Stories     []Story     `json:"stories"`
	UserStats   []UserStats `json:"userStats"`
	TodoStories []Story     `json:"todoStories"`
	InProgress  []Story     `json:"inProgress"`
	DoneStories []Story     `json:"doneStories"`
}
