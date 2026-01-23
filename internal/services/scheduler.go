package services

import (
	"ScrumBoard/internal/models"
	"fmt"
	"log"
	"os"
	"time"
)

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	NotifyEmail  string // Email to send daily report to
	ReportTime   string // Time to run daily report (e.g., "19:00" for 7 PM)
	ActiveSprint int    // Sprint ID to check (0 = active sprint)
	BoardID      string
}

// Scheduler runs automated tasks
type Scheduler struct {
	config       SchedulerConfig
	jiraService  *JiraService
	riskService  *RiskPredictionService
	webexService *WebexService
	stopChan     chan struct{}
	running      bool
}

// NewScheduler creates a new scheduler
func NewScheduler(jira *JiraService, config SchedulerConfig) *Scheduler {
	return &Scheduler{
		config:       config,
		jiraService:  jira,
		riskService:  NewRiskPredictionService(),
		webexService: NewWebexService(),
		stopChan:     make(chan struct{}),
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() {
	if s.running {
		return
	}
	s.running = true

	// Start state tracking goroutine (checks every 5 mins)
	go s.runStateTracker()
	go s.runScheduler()
	log.Printf("📅 Scheduler started - Daily report at %s, notifying %s", s.config.ReportTime, s.config.NotifyEmail)
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.running {
		close(s.stopChan)
		s.running = false
		log.Println("📅 Scheduler stopped")
	}
}

func (s *Scheduler) runScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	lastRunDate := ""

	for {
		select {
		case <-s.stopChan:
			return
		case now := <-ticker.C:
			currentTime := now.Format("15:04")
			currentDate := now.Format("2006-01-02")

			// Run consolidated report at configured time
			if currentTime == s.config.ReportTime && currentDate != lastRunDate {
				log.Printf("⏰ Running daily consolidated report at %s", currentTime)
				s.runConsolidatedReport()
				lastRunDate = currentDate
			}
		}
	}
}

// runStateTracker periodically checks for state changes
func (s *Scheduler) runStateTracker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.trackStateChanges()
		}
	}
}

func (s *Scheduler) trackStateChanges() {
	sprintID := s.config.ActiveSprint
	if sprintID == 0 {
		sprint, err := s.jiraService.GetActiveSprint(s.config.BoardID)
		if err != nil {
			return
		}
		sprintID = sprint.ID
	}

	sprint, _ := s.jiraService.GetSprintByID(sprintID)
	stories, _ := s.jiraService.GetSprintIssuesByJQL(sprintID)
	defects, _ := s.jiraService.GetSprintDefects(s.config.BoardID, sprintID)

	tracker := GetStateTracker()

	for _, story := range stories {
		tracker.CheckAndRecordTransition(story.Key, story.Title, "story", story.Status, story.Assignee.Name, sprint.Name)
	}
	for _, defect := range defects {
		tracker.CheckAndRecordTransition(defect.Key, defect.Summary, "defect", defect.Status, defect.Assignee.Name, sprint.Name)
	}
}

// RunNow runs the consolidated report immediately (for testing)
func (s *Scheduler) RunNow() {
	log.Println("🔄 Running immediate consolidated report...")
	s.trackStateChanges() // Update state tracking first
	s.runConsolidatedReport()
}

func (s *Scheduler) runConsolidatedReport() {
	// Get current sprint based on today's date
	sprint, err := s.getCurrentSprintByDate()
	if err != nil {
		log.Printf("❌ Failed to get current sprint: %v", err)
		return
	}
	log.Printf("📊 Daily Report for sprint: %s (ID: %d)", sprint.Name, sprint.ID)

	// Get stories using JQL (AMFDEVFT label) - works regardless of board
	stories, err := s.jiraService.GetSprintIssuesByJQL(sprint.ID)
	if err != nil {
		log.Printf("⚠️ Error fetching stories: %v", err)
		stories = []models.Story{}
	}
	log.Printf("📊 Found %d stories for sprint %s", len(stories), sprint.Name)

	defects, _ := s.jiraService.GetSprintDefects(s.config.BoardID, sprint.ID)

	// 1. Get state transitions
	stateTracker := GetStateTracker()
	transitions := stateTracker.GetTodaysTransitions()

	// 2. Get risks
	risks := s.riskService.AnalyzeSprintRisks(sprint, stories, defects)

	// 3. Get reported blockers
	blockerStore := GetBlockerStore()
	reportedBlockers := blockerStore.GetActiveBlockers()

	log.Printf("📊 Daily Report: %d transitions, %d risks, %d blockers", len(transitions), len(risks), len(reportedBlockers))

	// Calculate sprint stats
	dashboardService := NewDashboardService(s.jiraService)
	stats := dashboardService.CalculateStats(stories)

	// Calculate defect stats
	openDefects := 0
	resolvedDefects := 0
	for _, d := range defects {
		if d.Status == "Closed" || d.Status == "Resolved" || d.Status == "Accepted" || d.Status == "Done" {
			resolvedDefects++
		} else {
			openDefects++
		}
	}

	// Send consolidated report with full sprint details
	s.sendConsolidatedReport(sprint, stats, transitions, risks, reportedBlockers, len(stories), len(defects), openDefects, resolvedDefects)
}

func (s *Scheduler) sendConsolidatedReport(sprint models.Sprint, stats models.SprintStats, transitions []StateTransition, risks []AtRiskItem, blockers []ReportedBlocker, totalStories, totalDefects, openDefects, resolvedDefects int) {
	if !s.webexService.IsConfigured() {
		log.Println("❌ Webex not configured, skipping notification")
		return
	}

	// Calculate completion percentage
	completionPct := 0
	if stats.TotalPoints > 0 {
		completionPct = (stats.CompletedPoints * 100) / stats.TotalPoints
	}

	// Calculate days remaining
	daysRemaining := 0
	if sprint.EndDate != "" {
		endTime, err := time.Parse("2006-01-02", sprint.EndDate[:10])
		if err == nil {
			daysRemaining = int(time.Until(endTime).Hours() / 24)
			if daysRemaining < 0 {
				daysRemaining = 0
			}
		}
	}

	// Build sprint summary section
	sprintSummary := fmt.Sprintf(`📊 **Story Points Summary**
   ✅ Completed: **%d pts** (%d stories)
   🚧 In Progress: **%d pts** (%d stories)
   📋 To Do: **%d pts** (%d stories)
   📈 Total: **%d pts** (%d stories)
   🎯 Completion: **%d%%**

🐛 **Defect Summary**
   🔴 Open: **%d defects**
   ✅ Resolved: **%d defects**
   📈 Total: **%d defects**`,
		stats.CompletedPoints, stats.CompletedStories,
		stats.InProgressPoints, stats.InProgressStories,
		stats.RemainingPoints, stats.TodoStories,
		stats.TotalPoints, stats.TotalStories,
		completionPct,
		openDefects, resolvedDefects, totalDefects)

	// Build transitions section
	transitionSection := ""
	if len(transitions) == 0 {
		transitionSection = "   _No state changes today_\n"
	} else {
		doneCount := 0
		inProgressCount := 0
		for _, t := range transitions {
			emoji := "🔄"
			if t.ToState == "Done" || t.ToState == "Closed" || t.ToState == "Accepted" {
				emoji = "✅"
				doneCount++
			} else if t.ToState == "In Progress" || t.ToState == "In Development" {
				emoji = "🚧"
				inProgressCount++
			}
			typeEmoji := "📖"
			if t.ItemType == "defect" {
				typeEmoji = "🐛"
			}
			transitionSection += fmt.Sprintf("   %s %s **%s**: %s → %s (👤 %s)\n",
				emoji, typeEmoji, t.ItemKey, t.FromState, t.ToState, t.Assignee)
		}
		transitionSection = fmt.Sprintf("   ✅ %d completed | 🚧 %d in progress | 🔄 %d total\n\n%s",
			doneCount, inProgressCount, len(transitions), transitionSection)
	}

	// Build risks section
	riskSection := ""
	if len(risks) == 0 {
		riskSection = "   ✅ _No at-risk items_\n"
	} else {
		for _, risk := range risks {
			emoji := "🟡"
			if risk.RiskLevel == RiskHigh {
				emoji = "🔴"
			}
			typeEmoji := "📖"
			if risk.Type == "defect" {
				typeEmoji = "🐛"
			}
			riskSection += fmt.Sprintf("   %s %s **%s** - %s\n      ⚠️ %s | 👤 %s\n",
				emoji, typeEmoji, risk.Key, truncate(risk.Title, 40), risk.Reason, risk.Assignee)
		}
	}

	// Build blockers section
	blockerSection := ""
	if len(blockers) == 0 {
		blockerSection = "   ✅ _No blockers reported_\n"
	} else {
		for _, b := range blockers {
			blockerSection += fmt.Sprintf("   🚫 **%s** reported by %s\n      📝 %s\n",
				b.ItemKey, b.ReporterName, truncate(b.Description, 50))
		}
	}

	markdown := fmt.Sprintf(`📊 **Daily Sprint Summary**

🏃 **Sprint:** %s
📅 **Duration:** %s → %s
⏰ **Days Remaining:** %d days
📅 **Report Date:** %s

---

%s

---

**🔄 State Transitions Today**
%s
---

**⚠️ At-Risk Items** (%d)
%s
---

**🚫 Active Blockers** (%d)
%s
---
_Evening report from Scrum Insights_
`, sprint.Name,
		formatDate(sprint.StartDate), formatDate(sprint.EndDate),
		daysRemaining,
		time.Now().Format("Mon, 02 Jan 2006"),
		sprintSummary,
		transitionSection,
		len(risks), riskSection,
		len(blockers), blockerSection)

	err := s.webexService.SendCustomMessage(s.config.NotifyEmail, markdown)
	if err != nil {
		log.Printf("❌ Failed to send consolidated report: %v", err)
	} else {
		log.Printf("✅ Daily consolidated report sent to %s", s.config.NotifyEmail)
	}
}

// formatDate formats ISO date to readable format
func formatDate(isoDate string) string {
	if len(isoDate) < 10 {
		return isoDate
	}
	t, err := time.Parse("2006-01-02", isoDate[:10])
	if err != nil {
		return isoDate
	}
	return t.Format("02 Jan 2006")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// GetSchedulerConfig returns config from environment or defaults
func GetSchedulerConfig() SchedulerConfig {
	email := os.Getenv("SCHEDULER_NOTIFY_EMAIL")
	if email == "" {
		email = "prdewang@cisco.com"
	}

	reportTime := os.Getenv("SCHEDULER_REPORT_TIME")
	if reportTime == "" {
		reportTime = "19:00" // 7 PM default
	}

	return SchedulerConfig{
		NotifyEmail:  email,
		ReportTime:   reportTime,
		ActiveSprint: 0,
		BoardID:      "6991",
	}
}

// getCurrentSprintByDate finds the sprint that covers today's date
// Sprint naming convention: NM-XXYY where XX=year (26=2026) and YY=week number
func (s *Scheduler) getCurrentSprintByDate() (models.Sprint, error) {
	// Get all sprints
	sprints, err := s.jiraService.GetAllSprints(s.config.BoardID)
	if err != nil {
		return models.Sprint{}, err
	}

	today := time.Now()
	log.Printf("🔍 Looking for sprint covering date: %s", today.Format("2006-01-02"))

	// Find sprint where today falls between start and end date
	for _, sprint := range sprints {
		if sprint.StartDate == "" || sprint.EndDate == "" {
			continue
		}

		// Parse dates (format: 2026-01-13T...)
		startDate, err := time.Parse("2006-01-02", sprint.StartDate[:10])
		if err != nil {
			continue
		}
		endDate, err := time.Parse("2006-01-02", sprint.EndDate[:10])
		if err != nil {
			continue
		}

		// Check if today is within sprint dates (inclusive)
		if (today.After(startDate) || today.Equal(startDate)) && (today.Before(endDate) || today.Equal(endDate)) {
			log.Printf("✅ Found current sprint: %s (ID: %d) - %s to %s",
				sprint.Name, sprint.ID, sprint.StartDate[:10], sprint.EndDate[:10])
			return sprint, nil
		}
	}

	// Fallback: find sprint by name pattern NM-26XX (current year 2026)
	currentYear := today.Year() % 100 // 2026 -> 26
	_, currentWeek := today.ISOWeek()
	expectedName := fmt.Sprintf("NM-%d%02d", currentYear, currentWeek)
	log.Printf("🔍 Fallback: looking for sprint %s", expectedName)

	for _, sprint := range sprints {
		if sprint.Name == expectedName {
			log.Printf("✅ Found sprint by name: %s (ID: %d)", sprint.Name, sprint.ID)
			return sprint, nil
		}
	}

	// Last fallback: return most recent sprint
	if len(sprints) > 0 {
		// Sort by ID descending to get most recent
		mostRecent := sprints[0]
		for _, sprint := range sprints {
			if sprint.ID > mostRecent.ID {
				mostRecent = sprint
			}
		}
		log.Printf("⚠️ Using most recent sprint as fallback: %s (ID: %d)", mostRecent.Name, mostRecent.ID)
		return mostRecent, nil
	}

	return models.Sprint{}, fmt.Errorf("no sprint found for current date")
}
