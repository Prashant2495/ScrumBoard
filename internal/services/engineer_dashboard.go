package services

import (
	"ScrumBoard/internal/models"
	"log"
)

// EngineerDashboardService handles engineer dashboard operations
type EngineerDashboardService struct {
	jiraService *JiraService
}

// NewEngineerDashboardService creates a new engineer dashboard service
func NewEngineerDashboardService(jiraService *JiraService) *EngineerDashboardService {
	return &EngineerDashboardService{
		jiraService: jiraService,
	}
}

// GetEngineerDashboard fetches all data for engineer dashboard
func (s *EngineerDashboardService) GetEngineerDashboard(engineerEmail string, sprintID int) (*models.EngineerDashboardData, error) {
	log.Printf("📊 Fetching engineer dashboard for %s in sprint %d", engineerEmail, sprintID)

	// Fetch sprint details
	sprint, err := s.jiraService.GetSprintByID(sprintID)
	if err != nil {
		log.Printf("❌ Error fetching sprint: %v", err)
		return nil, err
	}

	// Fetch stories assigned to engineer in this sprint
	stories, err := s.jiraService.GetStoriesByEngineerAndSprint(engineerEmail, sprintID)
	if err != nil {
		log.Printf("❌ Error fetching stories: %v", err)
		return nil, err
	}

	// Fetch defects assigned to engineer in this sprint
	defects, err := s.jiraService.GetDefectsByEngineerAndSprint(engineerEmail, sprintID)
	if err != nil {
		log.Printf("❌ Error fetching defects: %v", err)
		return nil, err
	}

	// Calculate stats
	stats := s.calculateEngineerStats(stories, defects)
	storyStats := s.calculateStoryStats(stories)
	defectStats := s.calculateDefectStats(defects)

	// Get engineer details
	engineer := s.getEngineerDetails(engineerEmail, stories, defects)

	log.Printf("✅ Engineer dashboard: %d stories, %d defects", len(stories), len(defects))

	return &models.EngineerDashboardData{
		Engineer:    engineer,
		Sprint:      sprint,
		Stories:     stories,
		Defects:     defects,
		Stats:       stats,
		StoryStats:  storyStats,
		DefectStats: defectStats,
	}, nil
}

func (s *EngineerDashboardService) getEngineerDetails(email string, stories []models.Story, defects []models.Defect) models.Engineer {
	// Try to get name from stories or defects
	name := email
	if len(stories) > 0 && stories[0].Assignee.Name != "" {
		name = stories[0].Assignee.Name
	} else if len(defects) > 0 && defects[0].Assignee.Name != "" {
		name = defects[0].Assignee.Name
	}

	return models.Engineer{
		UserID:      email,
		Name:        name,
		Email:       email,
		AvatarColor: getAvatarColor(email),
	}
}

func (s *EngineerDashboardService) calculateEngineerStats(stories []models.Story, defects []models.Defect) models.EngineerStats {
	stats := models.EngineerStats{}

	// Story stats
	for _, story := range stories {
		stats.TotalStories++
		stats.StoryPoints += story.StoryPoints

		switch story.StatusCategory {
		case "done":
			stats.CompletedStories++
			stats.CompletedPoints += story.StoryPoints
		case "indeterminate":
			stats.InProgressStories++
		case "new":
			stats.TodoStories++
		}
	}

	// Defect stats
	for _, defect := range defects {
		stats.TotalDefects++

		switch defect.StatusCategory {
		case "done":
			stats.ResolvedDefects++
		case "indeterminate":
			stats.InProgressDefects++
		case "new":
			stats.OpenDefects++
		}
	}

	return stats
}

func (s *EngineerDashboardService) calculateStoryStats(stories []models.Story) models.StoryStatusStats {
	stats := models.StoryStatusStats{}

	for _, story := range stories {
		switch story.StatusCategory {
		case "done":
			stats.Done++
		case "indeterminate":
			stats.InProgress++
		case "new":
			stats.Todo++
		}
	}

	return stats
}

func (s *EngineerDashboardService) calculateDefectStats(defects []models.Defect) models.DefectStatusStats {
	stats := models.DefectStatusStats{}

	for _, defect := range defects {
		switch defect.StatusCategory {
		case "done":
			if defect.Status == "Closed" {
				stats.Closed++
			} else {
				stats.Resolved++
			}
		case "indeterminate":
			stats.InProgress++
		case "new":
			stats.Open++
		}
	}

	return stats
}

// getAvatarColor returns a color class for avatar based on email
func getAvatarColor(email string) string {
	colors := []string{
		"bg-gradient-to-br from-blue-500 to-blue-600",
		"bg-gradient-to-br from-green-500 to-green-600",
		"bg-gradient-to-br from-purple-500 to-purple-600",
		"bg-gradient-to-br from-pink-500 to-pink-600",
		"bg-gradient-to-br from-indigo-500 to-indigo-600",
		"bg-gradient-to-br from-red-500 to-red-600",
		"bg-gradient-to-br from-yellow-500 to-yellow-600",
		"bg-gradient-to-br from-teal-500 to-teal-600",
	}

	// Simple hash based on email length and first character
	hash := len(email)
	if len(email) > 0 {
		hash += int(email[0])
	}

	return colors[hash%len(colors)]
}
