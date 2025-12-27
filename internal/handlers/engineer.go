package handlers

import (
	"ScrumBoard/internal/models"
	"ScrumBoard/internal/services"
	"ScrumBoard/templates"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// EngineerHandler handles engineer dashboard requests
type EngineerHandler struct {
	engineerService *services.EngineerDashboardService
	jiraService     *services.JiraService
}

// NewEngineerHandler creates a new engineer handler
func NewEngineerHandler(engineerService *services.EngineerDashboardService, jiraService *services.JiraService) *EngineerHandler {
	return &EngineerHandler{
		engineerService: engineerService,
		jiraService:     jiraService,
	}
}

// HandleEngineerDashboard handles the main engineer dashboard page
func (h *EngineerHandler) HandleEngineerDashboard(c *fiber.Ctx) error {
	log.Println("📊 Engineer Dashboard Request")

	// Get query parameters
	engineerEmail := c.Query("email")
	sprintIDStr := c.Query("sprint")

	// Default to active sprint if not specified
	sprintID := 0
	if sprintIDStr != "" {
		var err error
		sprintID, err = strconv.Atoi(sprintIDStr)
		if err != nil {
			log.Printf("❌ Invalid sprint ID: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid sprint ID")
		}
	}

	// If no engineer selected, show empty dashboard
	if engineerEmail == "" {
		emptyData := models.EngineerDashboardData{
			Engineer: models.Engineer{
				Name:  "Select an Engineer",
				Email: "",
			},
			Sprint:  models.Sprint{},
			Stories: []models.Story{},
			Defects: []models.Defect{},
		}

		c.Set("Content-Type", "text/html")
		component := templates.EngineerDashboard(emptyData, []models.Engineer{}, []models.Sprint{})
		return component.Render(c.Context(), c.Response().BodyWriter())
	}

	// Fetch engineer dashboard data
	data, err := h.engineerService.GetEngineerDashboard(engineerEmail, sprintID)
	if err != nil {
		log.Printf("❌ Error fetching engineer dashboard: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching dashboard data")
	}

	// Render template
	c.Set("Content-Type", "text/html")
	component := templates.EngineerDashboard(*data, []models.Engineer{}, []models.Sprint{})
	if err := component.Render(c.Context(), c.Response().BodyWriter()); err != nil {
		log.Printf("❌ Error rendering template: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error rendering template")
	}

	log.Printf("✅ Engineer dashboard rendered for %s", engineerEmail)
	return nil
}

// HandleEngineerRefresh handles HTMX refresh requests
func (h *EngineerHandler) HandleEngineerRefresh(c *fiber.Ctx) error {
	log.Println("🔄 Engineer Dashboard Refresh")

	engineerEmail := c.Query("email")
	sprintIDStr := c.Query("sprint")

	sprintID := 0
	if sprintIDStr != "" {
		var err error
		sprintID, err = strconv.Atoi(sprintIDStr)
		if err != nil {
			log.Printf("❌ Invalid sprint ID: %v", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid sprint ID")
		}
	}

	if engineerEmail == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Engineer email required")
	}

	data, err := h.engineerService.GetEngineerDashboard(engineerEmail, sprintID)
	if err != nil {
		log.Printf("❌ Error fetching engineer dashboard: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching dashboard data")
	}

	c.Set("Content-Type", "text/html")
	component := templates.EngineerDashboardContent(*data)
	if err := component.Render(c.Context(), c.Response().BodyWriter()); err != nil {
		log.Printf("❌ Error rendering template: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error rendering template")
	}

	log.Printf("✅ Engineer dashboard refreshed for %s", engineerEmail)
	return nil
}
