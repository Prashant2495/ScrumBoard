package handlers

import (
	"ScrumBoard/internal/services"
	"ScrumBoard/templates"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
)

// ScrumMasterHandler handles scrum master dashboard requests
type ScrumMasterHandler struct {
	scrumMasterService *services.ScrumMasterService
	jiraService        *services.JiraService
	webexService       *services.WebexService
}

// NewScrumMasterHandler creates a new scrum master handler
func NewScrumMasterHandler(sm *services.ScrumMasterService, jira *services.JiraService, webex *services.WebexService) *ScrumMasterHandler {
	return &ScrumMasterHandler{
		scrumMasterService: sm,
		jiraService:        jira,
		webexService:       webex,
	}
}

// Dashboard renders the scrum master dashboard
func (h *ScrumMasterHandler) Dashboard(c *fiber.Ctx) error {
	log.Printf("📊 Scrum Master Dashboard Request")

	sprintID := c.QueryInt("sprint", 0)

	data, err := h.scrumMasterService.GetScrumMasterDashboard(sprintID)
	if err != nil {
		log.Printf("❌ Error fetching scrum master dashboard: %v", err)
		return c.Status(500).SendString("Error: " + err.Error())
	}

	log.Printf("✅ Scrum Master dashboard rendered")
	return render(c, templates.ScrumMasterDashboard(*data))
}

// Refresh returns updated dashboard content (for HTMX)
func (h *ScrumMasterHandler) Refresh(c *fiber.Ctx) error {
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.scrumMasterService.GetScrumMasterDashboard(sprintID)
	if err != nil {
		return c.Status(500).SendString("Error refreshing data: " + err.Error())
	}

	return render(c, templates.ScrumMasterDashboardContent(*data))
}

// GetSprints returns sprints as JSON for dropdown
func (h *ScrumMasterHandler) GetSprints(c *fiber.Ctx) error {
	boardID := "6537" // Default board
	sprints, err := h.jiraService.GetAllSprints(boardID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sprints"})
	}
	return c.JSON(sprints)
}

// GetVelocityData returns velocity data as JSON
func (h *ScrumMasterHandler) GetVelocityData(c *fiber.Ctx) error {
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.scrumMasterService.GetScrumMasterDashboard(sprintID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(data.VelocityData)
}

// GetTeamHealth returns team health data as JSON
func (h *ScrumMasterHandler) GetTeamHealth(c *fiber.Ctx) error {
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.scrumMasterService.GetScrumMasterDashboard(sprintID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(data.TeamHealth)
}

// GetBlockers returns blockers as JSON (combines Jira blockers + reported blockers)
func (h *ScrumMasterHandler) GetBlockers(c *fiber.Ctx) error {
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.scrumMasterService.GetScrumMasterDashboard(sprintID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Also include reported blockers from Webex
	blockerStore := services.GetBlockerStore()
	reportedBlockers := blockerStore.GetActiveBlockers()

	// Convert reported blockers to model blockers
	allBlockers := data.Blockers
	for _, rb := range reportedBlockers {
		allBlockers = append(allBlockers, rb.ToModelBlocker())
	}

	return c.JSON(allBlockers)
}

// GetReportedBlockers returns only Webex-reported blockers
func (h *ScrumMasterHandler) GetReportedBlockers(c *fiber.Ctx) error {
	blockerStore := services.GetBlockerStore()
	blockers := blockerStore.GetActiveBlockers()
	return c.JSON(fiber.Map{
		"success":  true,
		"blockers": blockers,
		"count":    len(blockers),
	})
}

// ResolveBlocker marks a reported blocker as resolved
func (h *ScrumMasterHandler) ResolveBlocker(c *fiber.Ctx) error {
	blockerID := c.Params("id")
	if blockerID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Blocker ID required"})
	}

	blockerStore := services.GetBlockerStore()
	if blockerStore.ResolveBlocker(blockerID) {
		log.Printf("✅ Blocker %s resolved", blockerID)
		return c.JSON(fiber.Map{"success": true, "message": "Blocker resolved"})
	}

	return c.Status(404).JSON(fiber.Map{"error": "Blocker not found"})
}

// GetRisks returns risk indicators as JSON
func (h *ScrumMasterHandler) GetRisks(c *fiber.Ctx) error {
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.scrumMasterService.GetScrumMasterDashboard(sprintID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(data.RiskIndicators)
}

// PingRequest represents a request to ping a user
type PingRequest struct {
	Email      string   `json:"email"`
	Name       string   `json:"name"`
	SprintName string   `json:"sprintName"`
	Items      []string `json:"items"`
}

// PingUser sends a Webex message to request status update
func (h *ScrumMasterHandler) PingUser(c *fiber.Ctx) error {
	var req PingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Email == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Email is required"})
	}

	log.Printf("📤 Pinging user %s (%s) for status update", req.Name, req.Email)

	// Check if Webex is configured
	if !h.webexService.IsConfigured() {
		log.Printf("⚠️ Webex not configured, simulating ping")
		return c.JSON(fiber.Map{
			"success": true,
			"message": fmt.Sprintf("Ping simulated for %s (Webex not configured)", req.Name),
			"demo":    true,
		})
	}

	// Send actual Webex message
	err := h.webexService.SendStatusPing(req.Email, req.Name, req.SprintName, req.Items)
	if err != nil {
		log.Printf("❌ Failed to ping user: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Store the ping in ping store
	pingStore := services.GetPingStore()
	message := "Status update requested"
	if len(req.Items) > 0 {
		message = req.Items[0]
	}
	ping := pingStore.SavePing(req.Email, req.Name, req.SprintName, message)
	log.Printf("✅ Ping sent and stored with ID: %s", ping.ID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Status request sent to %s via Webex", req.Name),
	})
}

// CheckWebexStatus returns whether Webex bot is configured
func (h *ScrumMasterHandler) CheckWebexStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"configured": h.webexService.IsConfigured(),
	})
}

// InfoRequest represents a request for info on a specific item
type InfoRequest struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	SprintName string `json:"sprintName"`
	ItemKey    string `json:"itemKey"`
	ItemTitle  string `json:"itemTitle"`
	ItemType   string `json:"itemType"` // "story" or "defect"
}

// RequestInfo sends a Webex message requesting info on a specific story/defect
func (h *ScrumMasterHandler) RequestInfo(c *fiber.Ctx) error {
	var req InfoRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.Email == "" || req.ItemKey == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Email and item key are required"})
	}

	// For testing: redirect all requests to Prashant Dewangan
	testEmail := "prdewang@cisco.com"
	log.Printf("📋 Requesting info from %s (%s) for %s: %s [redirected to %s for testing]", req.Name, req.Email, req.ItemKey, req.ItemTitle, testEmail)
	req.Email = testEmail

	// Check if Webex is configured
	if !h.webexService.IsConfigured() {
		log.Printf("⚠️ Webex not configured, simulating info request")
		// Still store the request
		pingStore := services.GetPingStore()
		ping := pingStore.SaveInfoRequest(req.Email, req.Name, req.SprintName, req.ItemKey, req.ItemTitle, req.ItemType, "Info requested")
		log.Printf("✅ Info request stored with ID: %s (Webex not configured)", ping.ID)
		return c.JSON(fiber.Map{
			"success": true,
			"message": fmt.Sprintf("Info request stored for %s (Webex not configured)", req.Name),
			"demo":    true,
			"ping_id": ping.ID,
		})
	}

	// Send Webex message for specific item
	err := h.webexService.SendInfoRequest(req.Email, req.Name, req.SprintName, req.ItemKey, req.ItemTitle, req.ItemType)
	if err != nil {
		log.Printf("❌ Failed to send info request: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Store the info request
	pingStore := services.GetPingStore()
	ping := pingStore.SaveInfoRequest(req.Email, req.Name, req.SprintName, req.ItemKey, req.ItemTitle, req.ItemType, "Info requested")
	log.Printf("✅ Info request sent and stored with ID: %s", ping.ID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Info request sent to %s via Webex for %s", req.Name, req.ItemKey),
		"ping_id": ping.ID,
	})
}

// WebexWebhookPayload represents incoming Webex webhook data
type WebexWebhookPayload struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Resource string `json:"resource"`
	Event    string `json:"event"`
	Data     struct {
		ID          string `json:"id"`
		RoomId      string `json:"roomId"`
		RoomType    string `json:"roomType"`
		PersonId    string `json:"personId"`
		PersonEmail string `json:"personEmail"`
		Created     string `json:"created"`
	} `json:"data"`
}

// WebexWebhook handles incoming Webex webhook for message replies
func (h *ScrumMasterHandler) WebexWebhook(c *fiber.Ctx) error {
	var payload WebexWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		log.Printf("❌ Webhook parse error: %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid webhook payload"})
	}

	log.Printf("📩 Webex webhook received: %s from %s", payload.Event, payload.Data.PersonEmail)

	// Only process message created events
	if payload.Resource != "messages" || payload.Event != "created" {
		return c.JSON(fiber.Map{"status": "ignored", "reason": "not a message event"})
	}

	// Get the actual message content from Webex API
	messageText, err := h.webexService.GetMessage(payload.Data.ID)
	if err != nil {
		log.Printf("❌ Failed to get message: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get message"})
	}

	log.Printf("📨 Message from %s: %s", payload.Data.PersonEmail, messageText)

	// Check if message contains blocker keywords
	if services.DetectBlocker(messageText) {
		log.Printf("🚫 Blocker detected from %s: %s", payload.Data.PersonEmail, messageText)

		// Get reporter name from pending ping if exists
		pingStore := services.GetPingStore()
		reporterName := payload.Data.PersonEmail
		itemKey := ""
		itemTitle := ""
		sprintName := ""

		// Try to get context from pending request
		pendingPings := pingStore.GetPendingPings()
		for _, ping := range pendingPings {
			if ping.EngineerEmail == payload.Data.PersonEmail {
				reporterName = ping.EngineerName
				itemKey = ping.ItemKey
				itemTitle = ping.ItemTitle
				sprintName = ping.SprintName
				break
			}
		}

		// Save the blocker
		blockerStore := services.GetBlockerStore()
		blocker := blockerStore.SaveBlocker(
			payload.Data.PersonEmail,
			reporterName,
			messageText,
			itemKey,
			itemTitle,
			sprintName,
		)
		log.Printf("✅ Blocker saved with ID: %s", blocker.ID)

		// Notify scrum master about the blocker
		go h.notifyBlocker(blocker)

		return c.JSON(fiber.Map{"status": "blocker_detected", "blocker_id": blocker.ID})
	}

	// Find pending ping for this user and update with response
	pingStore := services.GetPingStore()
	updated := pingStore.UpdateResponseByEmail(payload.Data.PersonEmail, messageText)

	if updated {
		log.Printf("✅ Response saved for %s", payload.Data.PersonEmail)
		return c.JSON(fiber.Map{"status": "success", "message": "Response recorded"})
	}

	log.Printf("⚠️ No pending request found for %s", payload.Data.PersonEmail)
	return c.JSON(fiber.Map{"status": "no_pending", "message": "No pending request found"})
}

// notifyBlocker sends notification to scrum master about new blocker
func (h *ScrumMasterHandler) notifyBlocker(blocker *services.ReportedBlocker) {
	scrumMasterEmail := "prdewang@cisco.com" // Your email

	markdown := fmt.Sprintf(`🚫 **New Blocker Reported!**

👤 **Reported by:** %s
📧 **Email:** %s
📋 **Related Item:** %s
📝 **Description:** %s
⏰ **Time:** %s

---
_Please check the Scrum Master Dashboard for details_`,
		blocker.ReporterName,
		blocker.ReporterEmail,
		blocker.ItemKey,
		blocker.Description,
		blocker.ReportedAt.Format("Mon, 02 Jan 2006 15:04"),
	)

	err := h.webexService.SendCustomMessage(scrumMasterEmail, markdown)
	if err != nil {
		log.Printf("❌ Failed to notify scrum master about blocker: %v", err)
	} else {
		log.Printf("✅ Scrum master notified about blocker from %s", blocker.ReporterEmail)
	}
}
