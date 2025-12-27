package services

import (
	"ScrumBoard/internal/models"
	"time"
)

// RiskLevel indicates how at-risk an item is
type RiskLevel string

const (
	RiskHigh   RiskLevel = "high"
	RiskMedium RiskLevel = "medium"
	RiskLow    RiskLevel = "low"
	RiskNone   RiskLevel = "none"
)

// AtRiskItem represents a story/defect that may not complete in sprint
type AtRiskItem struct {
	Key           string    `json:"key"`
	Title         string    `json:"title"`
	Type          string    `json:"type"` // "story" or "defect"
	Status        string    `json:"status"`
	Assignee      string    `json:"assignee"`
	AssigneeEmail string    `json:"assigneeEmail"`
	StoryPoints   int       `json:"storyPoints"`
	RiskLevel     RiskLevel `json:"riskLevel"`
	Reason        string    `json:"reason"`
	DaysStale     int       `json:"daysStale"`
	DaysLeft      int       `json:"daysLeft"`
}

// RiskPredictionService predicts items at risk of not completing
type RiskPredictionService struct{}

// NewRiskPredictionService creates a new service
func NewRiskPredictionService() *RiskPredictionService {
	return &RiskPredictionService{}
}

// AnalyzeSprintRisks analyzes stories and defects for completion risk
func (s *RiskPredictionService) AnalyzeSprintRisks(
	sprint models.Sprint,
	stories []models.Story,
	defects []models.Defect,
) []AtRiskItem {
	risks := []AtRiskItem{}

	// Calculate days left in sprint
	daysLeft := s.getDaysLeft(sprint.EndDate)

	// Analyze stories
	for _, story := range stories {
		if story.StatusCategory == "done" {
			continue // Skip completed
		}

		risk := s.analyzeStory(story, daysLeft)
		if risk != nil {
			risks = append(risks, *risk)
		}
	}

	// Analyze defects
	for _, defect := range defects {
		if defect.StatusCategory == "done" {
			continue // Skip resolved
		}

		risk := s.analyzeDefect(defect, daysLeft)
		if risk != nil {
			risks = append(risks, *risk)
		}
	}

	return risks
}

func (s *RiskPredictionService) getDaysLeft(endDate string) int {
	endTime, err := time.Parse("2006-01-02T15:04:05.000Z", endDate)
	if err != nil {
		endTime, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			return 7 // Default to 7 days if parse fails
		}
	}

	days := int(time.Until(endTime).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func (s *RiskPredictionService) getDaysStale(updatedAt string) int {
	updateTime, err := time.Parse("2006-01-02T15:04:05.000-0700", updatedAt)
	if err != nil {
		updateTime, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return 0
		}
	}
	return int(time.Since(updateTime).Hours() / 24)
}

func (s *RiskPredictionService) analyzeStory(story models.Story, daysLeft int) *AtRiskItem {
	daysStale := s.getDaysStale(story.UpdatedAt)

	// High Risk: Still in backlog/todo with less than 2 days left
	if story.StatusCategory == "new" && daysLeft <= 2 {
		return &AtRiskItem{
			Key:           story.Key,
			Title:         story.Title,
			Type:          "story",
			Status:        story.Status,
			Assignee:      story.Assignee.Name,
			AssigneeEmail: story.Assignee.Email,
			StoryPoints:   story.StoryPoints,
			RiskLevel:     RiskHigh,
			Reason:        "Still in backlog with only " + itoa(daysLeft) + " days left",
			DaysStale:     daysStale,
			DaysLeft:      daysLeft,
		}
	}

	// High Risk: In progress with high points and < 1 day
	if story.StatusCategory == "indeterminate" && story.StoryPoints >= 5 && daysLeft <= 1 {
		return &AtRiskItem{
			Key:           story.Key,
			Title:         story.Title,
			Type:          "story",
			Status:        story.Status,
			Assignee:      story.Assignee.Name,
			AssigneeEmail: story.Assignee.Email,
			StoryPoints:   story.StoryPoints,
			RiskLevel:     RiskHigh,
			Reason:        "High points (" + itoa(story.StoryPoints) + ") with only " + itoa(daysLeft) + " day left",
			DaysStale:     daysStale,
			DaysLeft:      daysLeft,
		}
	}

	// Medium Risk: No update in 3+ days while in progress
	if story.StatusCategory == "indeterminate" && daysStale >= 3 {
		return &AtRiskItem{
			Key:           story.Key,
			Title:         story.Title,
			Type:          "story",
			Status:        story.Status,
			Assignee:      story.Assignee.Name,
			AssigneeEmail: story.Assignee.Email,
			StoryPoints:   story.StoryPoints,
			RiskLevel:     RiskMedium,
			Reason:        "No update for " + itoa(daysStale) + " days",
			DaysStale:     daysStale,
			DaysLeft:      daysLeft,
		}
	}

	return nil
}

func (s *RiskPredictionService) analyzeDefect(defect models.Defect, daysLeft int) *AtRiskItem {
	daysStale := int(time.Since(defect.UpdatedAt).Hours() / 24)

	// High Risk: Critical/Blocker priority still open with < 2 days
	isHighPriority := defect.Priority == "Blocker" || defect.Priority == "Critical" || defect.Priority == "Highest"
	if isHighPriority && defect.StatusCategory == "new" && daysLeft <= 2 {
		return &AtRiskItem{
			Key:           defect.Key,
			Title:         defect.Summary,
			Type:          "defect",
			Status:        defect.Status,
			Assignee:      defect.Assignee.Name,
			AssigneeEmail: defect.Assignee.Email,
			StoryPoints:   0,
			RiskLevel:     RiskHigh,
			Reason:        defect.Priority + " priority bug still open with " + itoa(daysLeft) + " days left",
			DaysStale:     daysStale,
			DaysLeft:      daysLeft,
		}
	}

	// High Risk: Any defect in backlog with < 1 day
	if defect.StatusCategory == "new" && daysLeft <= 1 {
		return &AtRiskItem{
			Key:           defect.Key,
			Title:         defect.Summary,
			Type:          "defect",
			Status:        defect.Status,
			Assignee:      defect.Assignee.Name,
			AssigneeEmail: defect.Assignee.Email,
			StoryPoints:   0,
			RiskLevel:     RiskHigh,
			Reason:        "Still open with only " + itoa(daysLeft) + " day left",
			DaysStale:     daysStale,
			DaysLeft:      daysLeft,
		}
	}

	// Medium Risk: Stale for 5+ days
	if defect.StatusCategory != "done" && daysStale >= 5 {
		return &AtRiskItem{
			Key:           defect.Key,
			Title:         defect.Summary,
			Type:          "defect",
			Status:        defect.Status,
			Assignee:      defect.Assignee.Name,
			AssigneeEmail: defect.Assignee.Email,
			StoryPoints:   0,
			RiskLevel:     RiskMedium,
			Reason:        "No update for " + itoa(daysStale) + " days",
			DaysStale:     daysStale,
			DaysLeft:      daysLeft,
		}
	}

	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
