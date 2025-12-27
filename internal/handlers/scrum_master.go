package handlers

import (
	"ScrumBoard/internal/services"
	"ScrumBoard/templates"
	"log"

	"github.com/gofiber/fiber/v2"
)

// ScrumMasterHandler handles scrum master dashboard requests
type ScrumMasterHandler struct {
	scrumMasterService *services.ScrumMasterService
	jiraService        *services.JiraService
}

// NewScrumMasterHandler creates a new scrum master handler
func NewScrumMasterHandler(sm *services.ScrumMasterService, jira *services.JiraService) *ScrumMasterHandler {
	return &ScrumMasterHandler{
		scrumMasterService: sm,
		jiraService:        jira,
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

// GetBlockers returns blockers as JSON
func (h *ScrumMasterHandler) GetBlockers(c *fiber.Ctx) error {
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.scrumMasterService.GetScrumMasterDashboard(sprintID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(data.Blockers)
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

