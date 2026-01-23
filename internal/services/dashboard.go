package services

import (
	"ScrumBoard/internal/models"
	"log"
	"os"
)

// DashboardService processes sprint data for the dashboard
type DashboardService struct {
	jira *JiraService
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(jira *JiraService) *DashboardService {
	return &DashboardService{jira: jira}
}

// CalculateStats calculates sprint statistics from stories
func (d *DashboardService) CalculateStats(stories []models.Story) models.SprintStats {
	stats := models.SprintStats{}

	for _, story := range stories {
		stats.TotalStories++
		stats.TotalPoints += story.StoryPoints

		// Use StatusCategory for grouping (more reliable than status name)
		switch story.StatusCategory {
		case "done":
			stats.CompletedPoints += story.StoryPoints
			stats.CompletedStories++
		case "indeterminate":
			stats.InProgressPoints += story.StoryPoints
			stats.InProgressStories++
		default: // "new" = To Do
			stats.RemainingPoints += story.StoryPoints
			stats.TodoStories++
		}
	}

	return stats
}

// CalculateUserStats calculates per-user statistics
func (d *DashboardService) CalculateUserStats(stories []models.Story) []models.UserStats {
	userMap := make(map[string]*models.UserStats)

	for _, story := range stories {
		if story.Assignee.Name == "" {
			continue
		}

		userID := story.Assignee.Email
		if _, exists := userMap[userID]; !exists {
			userMap[userID] = &models.UserStats{
				User: story.Assignee,
			}
		}

		userMap[userID].AssignedStories++
		userMap[userID].AssignedPoints += story.StoryPoints

		if story.StatusCategory == "done" {
			userMap[userID].CompletedPoints += story.StoryPoints
		}
	}

	result := make([]models.UserStats, 0, len(userMap))
	for _, us := range userMap {
		result = append(result, *us)
	}

	return result
}

// GroupStoriesByStatus groups stories into To Do, In Progress, Done using StatusCategory
func (d *DashboardService) GroupStoriesByStatus(stories []models.Story) (todo, inProgress, done []models.Story) {
	for _, story := range stories {
		switch story.StatusCategory {
		case "done":
			done = append(done, story)
		case "indeterminate":
			inProgress = append(inProgress, story)
		default: // "new" = To Do
			todo = append(todo, story)
		}
	}
	return
}

// GetBoardID returns the board ID from environment
func (d *DashboardService) GetBoardID() string {
	boardID := os.Getenv("JIRA_BOARD_ID")
	if boardID == "" {
		boardID = "1"
	}
	return boardID
}

// GetBoards returns all boards
func (d *DashboardService) GetBoards() ([]models.Board, error) {
	return d.jira.GetBoards()
}

// GetAllSprints returns all sprints for a board
func (d *DashboardService) GetAllSprints(boardID string) ([]models.Sprint, error) {
	return d.jira.GetAllSprints(boardID)
}

// GetDashboardData fetches and processes all dashboard data (active sprint)
func (d *DashboardService) GetDashboardData(boardID string) (*models.DashboardData, error) {
	return d.GetDashboardDataForSprint(boardID, 0)
}

// GetDashboardDataForSprint fetches dashboard data for a specific sprint (0 = active sprint)
// Fetches from known AMF boards - fast and reliable
func (d *DashboardService) GetDashboardDataForSprint(boardID string, sprintID int) (*models.DashboardData, error) {
	// For demo, return mock data if Jira is not configured
	if d.jira.BaseURL == "" {
		return d.GetMockData(), nil
	}

	// Known AMF team boards - manually add new boards here when discovered
	amfBoards := []string{
		"6091", // AMF: Avengers (older sprints like NM-2515)
		"6537", // Mixed board
		"6991", // AMF: Interstellar (current sprints)
		"6992", // AMF: Avengers (current sprints like NM-2517)
		"7556", // AMF board for NM-2601 (add more as needed)
	}

	var sprint *models.Sprint
	var err error

	if sprintID == 0 {
		// Get active sprint from first board
		log.Printf("🔍 Fetching ACTIVE sprint")
		sprint, err = d.jira.GetActiveSprint(amfBoards[0])
		if err != nil {
			return nil, err
		}
		log.Printf("✅ Active sprint: %s (ID: %d)", sprint.Name, sprint.ID)
	} else {
		// Get specific sprint by ID
		log.Printf("🔍 Fetching sprint ID %d", sprintID)
		sprintData, err := d.jira.GetSprintByID(sprintID)
		if err != nil {
			return nil, err
		}
		sprint = &sprintData
		log.Printf("✅ Sprint: %s (ID: %d)", sprint.Name, sprint.ID)
	}

	// Fetch AMF team issues using JQL with AMFDEVFT label + subtasks (works for all sprints)
	allStories, err := d.jira.GetSprintIssuesWithSubtasks(sprint.ID)
	if err != nil {
		log.Printf("⚠️  Error fetching stories with subtasks via JQL: %v", err)
		// Fallback to board-based fetching (without subtasks)
		allStories = []models.Story{}
		for _, board := range amfBoards {
			stories, err := d.jira.GetSprintIssues(board, sprint.ID)
			if err != nil {
				log.Printf("⚠️  Error fetching from board %s: %v", board, err)
				continue
			}
			allStories = append(allStories, stories...)
		}
	}

	log.Printf("📊 Total AMF team stories: %d for sprint %s (ID: %d)", len(allStories), sprint.Name, sprint.ID)

	stats := d.CalculateStats(allStories)
	userStats := d.CalculateUserStats(allStories)
	todo, inProgress, done := d.GroupStoriesByStatus(allStories)

	log.Printf("📈 Stats: Total=%d, Completed=%d, InProgress=%d, Todo=%d",
		stats.TotalStories, stats.CompletedStories, stats.InProgressStories, stats.TodoStories)

	return &models.DashboardData{
		Sprint:      *sprint,
		Stats:       stats,
		Stories:     allStories,
		UserStats:   userStats,
		TodoStories: todo,
		InProgress:  inProgress,
		DoneStories: done,
	}, nil
}

// GetMockData returns mock data for demo purposes
func (d *DashboardService) GetMockData() *models.DashboardData {
	users := []models.User{
		{ID: "1", Name: "Rahul Sharma", Email: "rahul@example.com", AvatarURL: ""},
		{ID: "2", Name: "Priya Patel", Email: "priya@example.com", AvatarURL: ""},
		{ID: "3", Name: "Amit Kumar", Email: "amit@example.com", AvatarURL: ""},
		{ID: "4", Name: "Sneha Gupta", Email: "sneha@example.com", AvatarURL: ""},
	}

	stories := []models.Story{
		{ID: "1", Key: "SCRUM-101", Title: "User Authentication Flow", Status: "Done", StoryPoints: 8, Priority: "High", Assignee: users[0]},
		{ID: "2", Key: "SCRUM-102", Title: "Dashboard API Integration", Status: "In Progress", StoryPoints: 5, Priority: "High", Assignee: users[1]},
		{ID: "3", Key: "SCRUM-103", Title: "Sprint Report Generation", Status: "In Progress", StoryPoints: 5, Priority: "Medium", Assignee: users[0]},
		{ID: "4", Key: "SCRUM-104", Title: "User Profile Settings", Status: "To Do", StoryPoints: 3, Priority: "Low", Assignee: users[2]},
		{ID: "5", Key: "SCRUM-105", Title: "Notification System", Status: "To Do", StoryPoints: 5, Priority: "Medium", Assignee: users[3]},
		{ID: "6", Key: "SCRUM-106", Title: "Performance Optimization", Status: "Done", StoryPoints: 8, Priority: "High", Assignee: users[1]},
		{ID: "7", Key: "SCRUM-107", Title: "Mobile Responsive Design", Status: "In Progress", StoryPoints: 3, Priority: "Medium", Assignee: users[2]},
		{ID: "8", Key: "SCRUM-108", Title: "Data Export Feature", Status: "To Do", StoryPoints: 5, Priority: "Low", Assignee: users[3]},
	}

	stats := d.CalculateStats(stories)
	userStats := d.CalculateUserStats(stories)
	todo, inProgress, done := d.GroupStoriesByStatus(stories)

	return &models.DashboardData{
		Sprint: models.Sprint{
			ID:        1,
			Name:      "Sprint 23",
			State:     "active",
			StartDate: "2024-01-15",
			EndDate:   "2024-01-29",
			Goal:      "Complete user authentication and dashboard integration",
		},
		Stats:       stats,
		Stories:     stories,
		UserStats:   userStats,
		TodoStories: todo,
		InProgress:  inProgress,
		DoneStories: done,
	}
}
