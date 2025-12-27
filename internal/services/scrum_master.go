package services

import (
	"ScrumBoard/internal/models"
	"fmt"
	"log"
	"sort"
	"time"
)

// ScrumMasterService provides scrum master dashboard functionality
type ScrumMasterService struct {
	jiraService      *JiraService
	dashboardService *DashboardService
}

// NewScrumMasterService creates a new scrum master service
func NewScrumMasterService(jira *JiraService, dashboard *DashboardService) *ScrumMasterService {
	return &ScrumMasterService{
		jiraService:      jira,
		dashboardService: dashboard,
	}
}

// GetScrumMasterDashboard fetches all data for scrum master dashboard
func (s *ScrumMasterService) GetScrumMasterDashboard(sprintID int) (*models.ScrumMasterDashboardData, error) {
	log.Printf("📊 Fetching Scrum Master dashboard for sprint %d", sprintID)

	// Get current sprint data
	boardID := s.dashboardService.GetBoardID()
	data, err := s.dashboardService.GetDashboardDataForSprint(boardID, sprintID)
	if err != nil {
		return nil, err
	}

	// Get velocity data from past sprints
	velocityData := s.calculateVelocityData(data.Sprint)

	// Calculate team health
	teamHealth := s.calculateTeamHealth(data.Stats, data.UserStats, data.Sprint)

	// Get blockers
	blockers := s.identifyBlockers(data.Stories)

	// Get sprint history
	sprintHistory := s.getSprintHistory(boardID)

	// Identify risks
	risks := s.identifyRisks(data.Stats, velocityData, blockers, teamHealth)

	return &models.ScrumMasterDashboardData{
		Sprint:         data.Sprint,
		Stats:          data.Stats,
		UserStats:      data.UserStats,
		VelocityData:   velocityData,
		TeamHealth:     teamHealth,
		Blockers:       blockers,
		SprintHistory:  sprintHistory,
		RiskIndicators: risks,
	}, nil
}

// calculateVelocityData calculates velocity metrics
func (s *ScrumMasterService) calculateVelocityData(currentSprint models.Sprint) models.VelocityData {
	// Get past 6 sprints for velocity trend
	boardID := s.dashboardService.GetBoardID()
	sprints, _ := s.jiraService.GetAllSprints(boardID)

	var sprintVelocities []models.SprintVelocity
	totalCompleted := 0
	count := 0

	// Find closed sprints
	for _, sprint := range sprints {
		if sprint.State == "closed" && count < 6 {
			data, err := s.dashboardService.GetDashboardDataForSprint(boardID, sprint.ID)
			if err != nil {
				continue
			}
			sv := models.SprintVelocity{
				SprintName: sprint.Name,
				SprintID:   sprint.ID,
				Committed:  data.Stats.TotalPoints,
				Completed:  data.Stats.CompletedPoints,
				Carryover:  data.Stats.TotalPoints - data.Stats.CompletedPoints,
			}
			sprintVelocities = append(sprintVelocities, sv)
			totalCompleted += data.Stats.CompletedPoints
			count++
		}
	}

	avgVelocity := 0
	if count > 0 {
		avgVelocity = totalCompleted / count
	}

	// Calculate trend
	trend := "stable"
	trendPct := 0
	if len(sprintVelocities) >= 2 {
		latest := sprintVelocities[0].Completed
		previous := sprintVelocities[1].Completed
		if previous > 0 {
			trendPct = ((latest - previous) * 100) / previous
			if trendPct > 10 {
				trend = "up"
			} else if trendPct < -10 {
				trend = "down"
			}
		}
	}

	// Get current sprint velocity
	currentData, _ := s.dashboardService.GetDashboardDataForSprint(boardID, currentSprint.ID)
	currentVelocity := 0
	if currentData != nil {
		currentVelocity = currentData.Stats.CompletedPoints
	}

	return models.VelocityData{
		CurrentVelocity:  currentVelocity,
		AverageVelocity:  avgVelocity,
		VelocityTrend:    trend,
		TrendPercentage:  trendPct,
		SprintVelocities: sprintVelocities,
		PredictedNext:    avgVelocity,
	}
}

// calculateTeamHealth calculates team health metrics
func (s *ScrumMasterService) calculateTeamHealth(stats models.SprintStats, userStats []models.UserStats, sprint models.Sprint) models.TeamHealth {
	// Calculate capacity balance
	var overloaded, underutilized []models.TeamMember
	avgLoad := 0
	if len(userStats) > 0 {
		totalPoints := 0
		for _, us := range userStats {
			totalPoints += us.AssignedPoints
		}
		avgLoad = totalPoints / len(userStats)
	}

	for _, us := range userStats {
		loadPct := 0.0
		if avgLoad > 0 {
			loadPct = float64(us.AssignedPoints) / float64(avgLoad) * 100
		}
		member := models.TeamMember{
			Name:           us.User.Name,
			Email:          us.User.Email,
			AssignedPoints: us.AssignedPoints,
			LoadPercentage: loadPct,
		}
		if loadPct > 130 {
			overloaded = append(overloaded, member)
		} else if loadPct < 70 && us.AssignedPoints > 0 {
			underutilized = append(underutilized, member)
		}
	}

	// Calculate commitment ratio
	commitmentRatio := 0.0
	if stats.TotalPoints > 0 {
		commitmentRatio = float64(stats.CompletedPoints) / float64(stats.TotalPoints)
	}

	// Calculate burndown (ideal line)
	burndown := s.calculateBurndown(sprint, stats)

	// Calculate overall health score
	score := s.calculateHealthScore(stats, len(overloaded), commitmentRatio)
	status := "healthy"
	if score < 50 {
		status = "critical"
	} else if score < 70 {
		status = "at-risk"
	}

	return models.TeamHealth{
		OverallScore:    score,
		HealthStatus:    status,
		CommitmentRatio: commitmentRatio,
		SprintBurndown:  burndown,
		CapacityBalance: models.CapacityBalance{
			TotalCapacity:     len(userStats) * 10, // Assume 10 pts per person
			UsedCapacity:      stats.TotalPoints,
			OverloadedMembers: overloaded,
			UnderutilizedMbrs: underutilized,
		},
	}
}

// calculateBurndown creates burndown chart data
func (s *ScrumMasterService) calculateBurndown(sprint models.Sprint, stats models.SprintStats) []models.BurndownPoint {
	var burndown []models.BurndownPoint

	// Parse dates
	startDate, _ := time.Parse("2006-01-02", sprint.StartDate[:10])
	endDate, _ := time.Parse("2006-01-02", sprint.EndDate[:10])

	totalDays := int(endDate.Sub(startDate).Hours() / 24)
	if totalDays <= 0 {
		totalDays = 10 // Default 2 weeks
	}

	dailyIdealBurn := float64(stats.TotalPoints) / float64(totalDays)
	remaining := stats.TotalPoints

	for day := 0; day <= totalDays; day++ {
		currentDate := startDate.AddDate(0, 0, day)
		idealRemaining := stats.TotalPoints - int(dailyIdealBurn*float64(day))

		// Simulate actual burndown (completed work)
		if day > 0 {
			remaining = stats.RemainingPoints + stats.InProgressPoints
		}

		burndown = append(burndown, models.BurndownPoint{
			Date:            currentDate.Format("Jan 02"),
			Day:             day,
			RemainingPoints: remaining,
			IdealRemaining:  idealRemaining,
		})
	}

	return burndown
}

// calculateHealthScore calculates overall sprint health
func (s *ScrumMasterService) calculateHealthScore(stats models.SprintStats, overloadedCount int, commitmentRatio float64) int {
	score := 100

	// Deduct for low completion
	if commitmentRatio < 0.5 {
		score -= 30
	} else if commitmentRatio < 0.7 {
		score -= 15
	}

	// Deduct for overloaded members
	score -= overloadedCount * 10

	// Deduct for too much WIP
	if stats.InProgressStories > stats.CompletedStories {
		score -= 10
	}

	if score < 0 {
		score = 0
	}
	return score
}

// identifyBlockers finds blocked items
func (s *ScrumMasterService) identifyBlockers(stories []models.Story) []models.Blocker {
	var blockers []models.Blocker

	for _, story := range stories {
		// Check for blocker labels or specific statuses
		for _, label := range story.Labels {
			if label == "blocker" || label == "blocked" || label == "impediment" {
				blockers = append(blockers, models.Blocker{
					ID:         story.ID,
					Key:        story.Key,
					Title:      story.Title,
					Priority:   story.Priority,
					Status:     story.Status,
					AssignedTo: story.Assignee.Name,
					CreatedAt:  story.CreatedAt,
				})
				break
			}
		}
	}

	return blockers
}

// getSprintHistory returns past sprint summaries
func (s *ScrumMasterService) getSprintHistory(boardID string) []models.SprintSummary {
	var history []models.SprintSummary

	sprints, _ := s.jiraService.GetAllSprints(boardID)
	count := 0

	for _, sprint := range sprints {
		if sprint.State == "closed" && count < 6 {
			data, err := s.dashboardService.GetDashboardDataForSprint(boardID, sprint.ID)
			if err != nil {
				continue
			}

			completionPct := 0.0
			if data.Stats.TotalPoints > 0 {
				completionPct = float64(data.Stats.CompletedPoints) / float64(data.Stats.TotalPoints) * 100
			}

			history = append(history, models.SprintSummary{
				SprintID:      sprint.ID,
				SprintName:    sprint.Name,
				StartDate:     sprint.StartDate,
				EndDate:       sprint.EndDate,
				Committed:     data.Stats.TotalPoints,
				Completed:     data.Stats.CompletedPoints,
				Carryover:     data.Stats.TotalPoints - data.Stats.CompletedPoints,
				HealthScore:   80, // Placeholder
				CompletionPct: completionPct,
			})
			count++
		}
	}

	return history
}

// identifyRisks identifies sprint risks
func (s *ScrumMasterService) identifyRisks(stats models.SprintStats, velocity models.VelocityData, blockers []models.Blocker, health models.TeamHealth) []models.RiskIndicator {
	var risks []models.RiskIndicator

	// Check velocity drop
	if velocity.VelocityTrend == "down" && velocity.TrendPercentage < -20 {
		risks = append(risks, models.RiskIndicator{
			Type:        "velocity-drop",
			Severity:    "high",
			Title:       "Velocity Dropping",
			Description: "Team velocity has dropped significantly",
			Metric:      fmt.Sprintf("%d%%", velocity.TrendPercentage),
		})
	}

	// Check blockers
	if len(blockers) > 0 {
		severity := "medium"
		if len(blockers) > 3 {
			severity = "high"
		}
		risks = append(risks, models.RiskIndicator{
			Type:        "blocker",
			Severity:    severity,
			Title:       "Active Blockers",
			Description: fmt.Sprintf("%d blockers need attention", len(blockers)),
		})
	}

	// Check capacity
	if len(health.CapacityBalance.OverloadedMembers) > 2 {
		risks = append(risks, models.RiskIndicator{
			Type:        "capacity",
			Severity:    "medium",
			Title:       "Capacity Imbalance",
			Description: "Multiple team members are overloaded",
		})
	}

	// Check low completion
	if health.CommitmentRatio < 0.5 {
		risks = append(risks, models.RiskIndicator{
			Type:        "scope-creep",
			Severity:    "high",
			Title:       "Low Completion Rate",
			Description: "Sprint completion is below 50%",
		})
	}

	// Sort by severity
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].Severity == "high" && risks[j].Severity != "high"
	})

	return risks
}
