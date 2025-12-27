package services

import (
	"ScrumBoard/internal/models"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
)

// DefectDashboardService processes defect data for the dashboard
type DefectDashboardService struct {
	jira *JiraService
}

// NewDefectDashboardService creates a new defect dashboard service
func NewDefectDashboardService(jira *JiraService) *DefectDashboardService {
	return &DefectDashboardService{jira: jira}
}

// CalculateDefectStats calculates defect statistics
func (d *DefectDashboardService) CalculateDefectStats(defects []models.Defect) models.DefectStats {
	stats := models.DefectStats{}

	var totalResolutionTime int
	var resolvedCount int
	var oldestAge int

	for _, defect := range defects {
		stats.TotalDefects++

		// By Status Category
		switch defect.StatusCategory {
		case "done":
			if defect.Status == "Closed" {
				stats.ClosedDefects++
			} else {
				stats.ResolvedDefects++
			}
		case "indeterminate":
			stats.InProgressDefects++
		default: // "new"
			stats.OpenDefects++
		}

		// By Severity
		switch defect.Severity {
		case "Critical":
			stats.CriticalDefects++
		case "High":
			stats.HighDefects++
		case "Medium":
			stats.MediumDefects++
		case "Low":
			stats.LowDefects++
		}

		// By Priority
		switch defect.Priority {
		case "Blocker", "Highest":
			stats.BlockerDefects++
		case "Major":
			stats.MajorDefects++
		case "Minor":
			stats.MinorDefects++
		case "Trivial", "Lowest":
			stats.TrivialDefects++
		}

		// Resolution time metrics
		if defect.ResolvedAt != nil && defect.ResolutionTime > 0 {
			totalResolutionTime += defect.ResolutionTime
			resolvedCount++
		}

		// Age metrics
		if defect.AgeInDays > oldestAge {
			oldestAge = defect.AgeInDays
		}

		if defect.StatusCategory != "done" {
			if defect.AgeInDays > 7 {
				stats.DefectsAgedOver7Days++
			}
			if defect.AgeInDays > 30 {
				stats.DefectsAgedOver30Days++
			}
		}
	}

	// Calculate average resolution time
	if resolvedCount > 0 {
		stats.AvgResolutionTimeHours = float64(totalResolutionTime) / float64(resolvedCount)
	}
	stats.OldestDefectDays = oldestAge

	return stats
}

// CalculateDefectByAssignee calculates defect statistics per assignee
func (d *DefectDashboardService) CalculateDefectByAssignee(defects []models.Defect) []models.DefectByAssignee {
	assigneeMap := make(map[string]*models.DefectByAssignee)

	for _, defect := range defects {
		if defect.Assignee.Name == "" {
			continue
		}

		userID := defect.Assignee.Email
		if _, exists := assigneeMap[userID]; !exists {
			assigneeMap[userID] = &models.DefectByAssignee{
				User: defect.Assignee,
			}
		}

		assigneeMap[userID].TotalDefects++

		if defect.StatusCategory == "done" {
			assigneeMap[userID].ResolvedDefects++
		} else if defect.StatusCategory == "new" {
			assigneeMap[userID].OpenDefects++
		}

		if defect.Severity == "Critical" {
			assigneeMap[userID].CriticalDefects++
		} else if defect.Severity == "High" {
			assigneeMap[userID].HighDefects++
		}
	}

	result := make([]models.DefectByAssignee, 0, len(assigneeMap))
	for _, stats := range assigneeMap {
		result = append(result, *stats)
	}

	// Sort by total defects descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalDefects > result[j].TotalDefects
	})

	return result
}

// CalculateSeverityBreakdown calculates defect breakdown by severity
func (d *DefectDashboardService) CalculateSeverityBreakdown(defects []models.Defect) []models.DefectBySeverity {
	severityMap := make(map[string]*models.DefectBySeverity)
	severities := []string{"Critical", "High", "Medium", "Low"}

	// Initialize
	for _, sev := range severities {
		severityMap[sev] = &models.DefectBySeverity{Severity: sev}
	}

	for _, defect := range defects {
		if breakdown, exists := severityMap[defect.Severity]; exists {
			breakdown.Count++
			if defect.StatusCategory == "new" || defect.StatusCategory == "indeterminate" {
				breakdown.Open++
			} else {
				breakdown.Resolved++
			}
		}
	}

	result := make([]models.DefectBySeverity, 0, len(severities))
	for _, sev := range severities {
		result = append(result, *severityMap[sev])
	}

	return result
}

// CalculatePriorityBreakdown calculates defect breakdown by priority
func (d *DefectDashboardService) CalculatePriorityBreakdown(defects []models.Defect) []models.DefectByPriority {
	priorityMap := make(map[string]*models.DefectByPriority)

	for _, defect := range defects {
		priority := defect.Priority
		if _, exists := priorityMap[priority]; !exists {
			priorityMap[priority] = &models.DefectByPriority{Priority: priority}
		}

		priorityMap[priority].Count++
		if defect.StatusCategory == "new" || defect.StatusCategory == "indeterminate" {
			priorityMap[priority].Open++
		} else {
			priorityMap[priority].Resolved++
		}
	}

	result := make([]models.DefectByPriority, 0, len(priorityMap))
	for _, breakdown := range priorityMap {
		result = append(result, *breakdown)
	}

	// Sort by priority order
	priorityOrder := map[string]int{
		"Blocker": 1, "Highest": 1,
		"Critical": 2, "High": 2,
		"Major": 3, "Medium": 3,
		"Minor": 4, "Low": 4,
		"Trivial": 5, "Lowest": 5,
	}
	sort.Slice(result, func(i, j int) bool {
		return priorityOrder[result[i].Priority] < priorityOrder[result[j].Priority]
	})

	return result
}

// GetBoardID returns the board ID from environment
func (d *DefectDashboardService) GetBoardID() string {
	boardID := os.Getenv("JIRA_BOARD_ID")
	if boardID == "" {
		boardID = "1"
	}
	return boardID
}

// GetAllSprints returns all sprints for a board
func (d *DefectDashboardService) GetAllSprints(boardID string) ([]models.Sprint, error) {
	return d.jira.GetAllSprints(boardID)
}

// GroupDefectsByStatus groups defects into Open, In Progress, Resolved
func (d *DefectDashboardService) GroupDefectsByStatus(defects []models.Defect) (open, inProgress, resolved []models.Defect) {
	for _, defect := range defects {
		switch defect.StatusCategory {
		case "done":
			resolved = append(resolved, defect)
		case "indeterminate":
			inProgress = append(inProgress, defect)
		default: // "new"
			open = append(open, defect)
		}
	}
	return
}

// GetDefectDashboardData fetches and processes all defect dashboard data
func (d *DefectDashboardService) GetDefectDashboardData(boardID string) (*models.DefectDashboardData, error) {
	return d.GetDefectDashboardDataForSprint(boardID, 0)
}

// GetDefectDashboardDataForSprint fetches defect data for a specific sprint (0 = active sprint)
func (d *DefectDashboardService) GetDefectDashboardDataForSprint(boardID string, sprintID int) (*models.DefectDashboardData, error) {
	var sprint *models.Sprint
	var err error

	// If sprintID is 0, get active sprint
	if sprintID == 0 {
		sprint, err = d.jira.GetActiveSprint(boardID)
		if err != nil {
			// Return mock data if Jira is not configured
			return d.GetMockDefectData(), nil
		}
	} else {
		// Get all sprints and find the requested one
		sprints, err := d.jira.GetAllSprints(boardID)
		if err != nil {
			return nil, err
		}
		for _, s := range sprints {
			if s.ID == sprintID {
				sprint = &s
				break
			}
		}
		if sprint == nil {
			return nil, fmt.Errorf("sprint %d not found", sprintID)
		}
	}

	// Fetch defects
	defects, err := d.jira.GetSprintDefects(boardID, sprint.ID)
	if err != nil {
		return nil, err
	}

	// If no defects found, return mock data for demo
	if len(defects) == 0 {
		return d.GetMockDefectData(), nil
	}

	// Debug: Log defect count and assignees
	log.Printf("📊 Found %d defects in sprint %s", len(defects), sprint.Name)
	for i, d := range defects {
		log.Printf("  Bug %d: %s - Assignee: %s (%s)", i+1, d.Key, d.Assignee.Name, d.Assignee.Email)
	}

	// Calculate statistics
	stats := d.CalculateDefectStats(defects)
	assigneeStats := d.CalculateDefectByAssignee(defects)
	severityBreakdown := d.CalculateSeverityBreakdown(defects)
	priorityBreakdown := d.CalculatePriorityBreakdown(defects)
	open, inProgress, resolved := d.GroupDefectsByStatus(defects)

	return &models.DefectDashboardData{
		Sprint:            *sprint,
		Stats:             stats,
		Defects:           defects,
		AssigneeStats:     assigneeStats,
		SeverityBreakdown: severityBreakdown,
		PriorityBreakdown: priorityBreakdown,
		OpenDefects:       open,
		InProgressDefects: inProgress,
		ResolvedDefects:   resolved,
	}, nil
}

// GetMockDefectData returns mock defect data for demo purposes
func (d *DefectDashboardService) GetMockDefectData() *models.DefectDashboardData {
	now := time.Now()
	users := []models.User{
		{ID: "1", Name: "Rahul Sharma", Email: "rahul@example.com"},
		{ID: "2", Name: "Priya Patel", Email: "priya@example.com"},
		{ID: "3", Name: "Amit Kumar", Email: "amit@example.com"},
		{ID: "4", Name: "Sneha Gupta", Email: "sneha@example.com"},
	}

	resolved1 := now.Add(-24 * time.Hour)
	resolved2 := now.Add(-48 * time.Hour)

	defects := []models.Defect{
		{
			ID: "1", Key: "BUG-101", Summary: "Login page crashes on invalid credentials",
			Status: "Open", StatusCategory: "new", Priority: "Blocker", Severity: "Critical",
			Assignee: users[0], Reporter: users[1], CreatedAt: now.Add(-72 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour), AgeInDays: 3, Environment: "Production",
		},
		{
			ID: "2", Key: "BUG-102", Summary: "Dashboard loading slow for large datasets",
			Status: "In Progress", StatusCategory: "indeterminate", Priority: "High", Severity: "High",
			Assignee: users[1], Reporter: users[0], CreatedAt: now.Add(-120 * time.Hour),
			UpdatedAt: now.Add(-1 * time.Hour), AgeInDays: 5, Environment: "Production",
		},
		{
			ID: "3", Key: "BUG-103", Summary: "Export to CSV missing some columns",
			Status: "Resolved", StatusCategory: "done", Priority: "Medium", Severity: "Medium",
			Assignee: users[2], Reporter: users[3], CreatedAt: now.Add(-168 * time.Hour),
			UpdatedAt: now.Add(-24 * time.Hour), ResolvedAt: &resolved1, AgeInDays: 7,
			ResolutionTime: 144, Environment: "Staging",
		},
		{
			ID: "4", Key: "BUG-104", Summary: "Typo in notification message",
			Status: "Closed", StatusCategory: "done", Priority: "Low", Severity: "Low",
			Assignee: users[3], Reporter: users[2], CreatedAt: now.Add(-240 * time.Hour),
			UpdatedAt: now.Add(-48 * time.Hour), ResolvedAt: &resolved2, AgeInDays: 10,
			ResolutionTime: 192, Environment: "Production",
		},
		{
			ID: "5", Key: "BUG-105", Summary: "API timeout on bulk operations",
			Status: "Open", StatusCategory: "new", Priority: "Critical", Severity: "High",
			Assignee: users[0], Reporter: users[1], CreatedAt: now.Add(-48 * time.Hour),
			UpdatedAt: now.Add(-12 * time.Hour), AgeInDays: 2, Environment: "Production",
		},
		{
			ID: "6", Key: "BUG-106", Summary: "Mobile view alignment issues",
			Status: "In Progress", StatusCategory: "indeterminate", Priority: "Medium", Severity: "Medium",
			Assignee: users[2], Reporter: users[0], CreatedAt: now.Add(-96 * time.Hour),
			UpdatedAt: now.Add(-6 * time.Hour), AgeInDays: 4, Environment: "Staging",
		},
		{
			ID: "7", Key: "BUG-107", Summary: "Memory leak in background service",
			Status: "Open", StatusCategory: "new", Priority: "High", Severity: "Critical",
			Assignee: users[1], Reporter: users[3], CreatedAt: now.Add(-192 * time.Hour),
			UpdatedAt: now.Add(-24 * time.Hour), AgeInDays: 8, Environment: "Production",
		},
	}

	stats := d.CalculateDefectStats(defects)
	assigneeStats := d.CalculateDefectByAssignee(defects)
	severityBreakdown := d.CalculateSeverityBreakdown(defects)
	priorityBreakdown := d.CalculatePriorityBreakdown(defects)
	open, inProgress, resolved := d.GroupDefectsByStatus(defects)

	return &models.DefectDashboardData{
		Sprint: models.Sprint{
			ID: 1, Name: "Sprint 23", State: "active",
			StartDate: "2024-01-15", EndDate: "2024-01-29",
			Goal: "Complete user authentication and dashboard integration",
		},
		Stats:             stats,
		Defects:           defects,
		AssigneeStats:     assigneeStats,
		SeverityBreakdown: severityBreakdown,
		PriorityBreakdown: priorityBreakdown,
		OpenDefects:       open,
		InProgressDefects: inProgress,
		ResolvedDefects:   resolved,
	}
}
