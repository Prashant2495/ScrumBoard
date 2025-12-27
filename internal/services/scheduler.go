package services

import (
	"fmt"
	"log"
	"os"
	"time"
)

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	NotifyEmail  string // Email to send daily risk report to
	CheckTime    string // Time to run daily check (e.g., "09:00")
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

	go s.runScheduler()
	log.Printf("📅 Scheduler started - Daily risk check at %s, notifying %s", s.config.CheckTime, s.config.NotifyEmail)
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
	// Check every minute if it's time to run
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

			// Run if it's the configured time and we haven't run today
			if currentTime == s.config.CheckTime && currentDate != lastRunDate {
				log.Printf("⏰ Running scheduled risk check at %s", currentTime)
				s.runDailyRiskCheck()
				lastRunDate = currentDate
			}
		}
	}
}

// RunNow runs the risk check immediately (for testing)
func (s *Scheduler) RunNow() {
	log.Println("🔄 Running immediate risk check...")
	s.runDailyRiskCheck()
}

func (s *Scheduler) runDailyRiskCheck() {
	// Get active sprint if not specified
	sprintID := s.config.ActiveSprint
	if sprintID == 0 {
		sprint, err := s.jiraService.GetActiveSprint(s.config.BoardID)
		if err != nil {
			log.Printf("❌ Failed to get active sprint: %v", err)
			return
		}
		sprintID = sprint.ID
	}

	// Get sprint info
	sprint, err := s.jiraService.GetSprintByID(sprintID)
	if err != nil {
		log.Printf("❌ Failed to get sprint: %v", err)
		return
	}

	// Get stories and defects
	stories, _ := s.jiraService.GetSprintIssuesByJQL(sprintID)
	defects, _ := s.jiraService.GetSprintDefects(s.config.BoardID, sprintID)

	// Analyze risks
	risks := s.riskService.AnalyzeSprintRisks(sprint, stories, defects)

	log.Printf("📊 Found %d at-risk items in sprint %s", len(risks), sprint.Name)

	// Send consolidated report
	s.sendDailyReport(sprint.Name, risks)
}

func (s *Scheduler) sendDailyReport(sprintName string, risks []AtRiskItem) {
	if !s.webexService.IsConfigured() {
		log.Println("❌ Webex not configured, skipping notification")
		return
	}

	var markdown string
	if len(risks) == 0 {
		markdown = fmt.Sprintf(`✅ **Daily Sprint Risk Report**

🏃 **Sprint:** %s
📅 **Date:** %s

🎉 **All items on track!** No at-risk items found.

---
_Automated report from Scrum Insights_
`, sprintName, time.Now().Format("Mon, 02 Jan 2006"))
	} else {
		// Build risk list
		riskList := ""
		highCount := 0
		mediumCount := 0

		for _, risk := range risks {
			emoji := "🟡"
			if risk.RiskLevel == RiskHigh {
				emoji = "🔴"
				highCount++
			} else {
				mediumCount++
			}
			typeEmoji := "📖"
			if risk.Type == "defect" {
				typeEmoji = "🐛"
			}
			riskList += fmt.Sprintf("\n%s %s **%s** - %s\n   ⚠️ %s | 👤 %s | ⏰ %d days left\n",
				emoji, typeEmoji, risk.Key, truncate(risk.Title, 50), risk.Reason, risk.Assignee, risk.DaysLeft)
		}

		markdown = fmt.Sprintf(`🚨 **Daily Sprint Risk Report**

🏃 **Sprint:** %s
📅 **Date:** %s

⚠️ **%d items at risk** (🔴 %d High, 🟡 %d Medium)
%s
---
_Automated report from Scrum Insights_
`, sprintName, time.Now().Format("Mon, 02 Jan 2006"), len(risks), highCount, mediumCount, riskList)
	}

	err := s.webexService.SendCustomMessage(s.config.NotifyEmail, markdown)
	if err != nil {
		log.Printf("❌ Failed to send daily report: %v", err)
	} else {
		log.Printf("✅ Daily risk report sent to %s", s.config.NotifyEmail)
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
		email = "prdewang@cisco.com" // Default to you
	}

	checkTime := os.Getenv("SCHEDULER_CHECK_TIME")
	if checkTime == "" {
		checkTime = "09:00" // 9 AM
	}

	return SchedulerConfig{
		NotifyEmail:  email,
		CheckTime:    checkTime,
		ActiveSprint: 0, // Auto-detect active sprint
		BoardID:      "6991",
	}
}
