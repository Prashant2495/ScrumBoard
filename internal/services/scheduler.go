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
	// Get active sprint
	sprintID := s.config.ActiveSprint
	if sprintID == 0 {
		sprint, err := s.jiraService.GetActiveSprint(s.config.BoardID)
		if err != nil {
			log.Printf("❌ Failed to get active sprint: %v", err)
			return
		}
		sprintID = sprint.ID
	}

	sprint, err := s.jiraService.GetSprintByID(sprintID)
	if err != nil {
		log.Printf("❌ Failed to get sprint: %v", err)
		return
	}

	// Get data from both AMF boards (6991 and 6992)
	stories6991, _ := s.jiraService.GetSprintIssues("6991", sprintID)
	stories6992, _ := s.jiraService.GetSprintIssues("6992", sprintID)

	// Combine stories from both boards (dedupe by key)
	storyMap := make(map[string]models.Story)
	for _, s := range stories6991 {
		storyMap[s.Key] = s
	}
	for _, s := range stories6992 {
		storyMap[s.Key] = s
	}
	stories := make([]models.Story, 0, len(storyMap))
	for _, s := range storyMap {
		stories = append(stories, s)
	}
	log.Printf("📊 Combined stories: %d from board 6991, %d from board 6992, %d total unique", len(stories6991), len(stories6992), len(stories))

	defects, _ := s.jiraService.GetSprintDefects(s.config.BoardID, sprintID)

	// 1. Get state transitions
	stateTracker := GetStateTracker()
	transitions := stateTracker.GetTodaysTransitions()

	// 2. Get risks
	risks := s.riskService.AnalyzeSprintRisks(sprint, stories, defects)

	// 3. Get reported blockers
	blockerStore := GetBlockerStore()
	reportedBlockers := blockerStore.GetActiveBlockers()

	log.Printf("📊 Daily Report: %d transitions, %d risks, %d blockers", len(transitions), len(risks), len(reportedBlockers))

	// Send consolidated report
	s.sendConsolidatedReport(sprint.Name, transitions, risks, reportedBlockers, len(stories), len(defects))
}

func (s *Scheduler) sendConsolidatedReport(sprintName string, transitions []StateTransition, risks []AtRiskItem, blockers []ReportedBlocker, totalStories, totalDefects int) {
	if !s.webexService.IsConfigured() {
		log.Println("❌ Webex not configured, skipping notification")
		return
	}

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
📅 **Date:** %s
📈 **Items:** %d stories, %d defects

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
`, sprintName, time.Now().Format("Mon, 02 Jan 2006"),
		totalStories, totalDefects,
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
