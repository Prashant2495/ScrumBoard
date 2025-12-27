package handlers

import (
	"ScrumBoard/internal/models"
	"ScrumBoard/internal/services"
	"ScrumBoard/templates"
	"log"
	"os"
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

// PingEngineer sends a ping to an engineer via Webex
func (h *EngineerHandler) PingEngineer(c *fiber.Ctx) error {
	email := c.FormValue("email")
	name := c.FormValue("name")
	sprintName := c.FormValue("sprint_name")

	if email == "" || name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Email and name required",
		})
	}

	log.Printf("📤 Pinging engineer %s (%s) for status update", name, email)

	// Check if Webex token is configured
	webexToken := os.Getenv("WEBEX_BOT_TOKEN")
	if webexToken == "" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Webex bot not configured",
		})
	}

	// Get stories for this engineer
	var assignedItems []string
	if h.engineerService != nil {
		data, err := h.engineerService.GetEngineerDashboard(email, 0)
		if err == nil && data != nil {
			for _, story := range data.Stories {
				assignedItems = append(assignedItems, story.Key+" - "+story.Title)
			}
		}
	}

	// Send Webex message
	webexService := services.NewWebexService()
	err := webexService.SendStatusPing(email, name, sprintName, assignedItems)
	if err != nil {
		log.Printf("❌ Failed to send Webex ping: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to send message: " + err.Error(),
		})
	}

	// Store the ping
	pingStore := services.GetPingStore()
	ping := pingStore.SavePing(email, name, sprintName, "Status update requested")

	log.Printf("✅ Ping sent and stored with ID: %s", ping.ID)
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Ping sent successfully",
		"ping_id": ping.ID,
	})
}

// RespondToPing saves a response to a ping
func (h *EngineerHandler) RespondToPing(c *fiber.Ctx) error {
	pingID := c.FormValue("ping_id")
	response := c.FormValue("response")

	if pingID == "" || response == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Ping ID and response required",
		})
	}

	pingStore := services.GetPingStore()
	if pingStore.SaveResponse(pingID, response) {
		log.Printf("✅ Response saved for ping %s", pingID)
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Response saved",
		})
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"success": false,
		"message": "Ping not found",
	})
}

// GetPings returns all pings for an engineer
func (h *EngineerHandler) GetPings(c *fiber.Ctx) error {
	email := c.Query("email")

	pingStore := services.GetPingStore()

	var pings []models.PingMessage
	if email != "" {
		pings = pingStore.GetPingsByEngineer(email)
	} else {
		pings = pingStore.GetAllPings()
	}

	return c.JSON(fiber.Map{
		"success": true,
		"pings":   pings,
	})
}
