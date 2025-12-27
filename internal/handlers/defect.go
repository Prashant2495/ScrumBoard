package handlers

import (
	"ScrumBoard/internal/services"
	"ScrumBoard/templates"
	"log"

	"github.com/gofiber/fiber/v2"
)

// DefectHandler handles defect dashboard requests
type DefectHandler struct {
	service *services.DefectDashboardService
}

// NewDefectHandler creates a new defect handler
func NewDefectHandler(service *services.DefectDashboardService) *DefectHandler {
	return &DefectHandler{service: service}
}

// Index renders the defect dashboard page
func (h *DefectHandler) Index(c *fiber.Ctx) error {
	boardID := c.Query("board", "")
	if boardID == "" {
		boardID = h.service.GetBoardID()
	}

	// Get sprint ID from query parameter (0 = active sprint)
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.service.GetDefectDashboardDataForSprint(boardID, sprintID)
	if err != nil {
		log.Printf("Error fetching defect data: %v", err)
		return c.Status(500).SendString("Error loading defect dashboard")
	}

	// Set content type and render
	c.Set("Content-Type", "text/html; charset=utf-8")
	return templates.DefectDashboard(*data).Render(c.Context(), c.Response().BodyWriter())
}

// Refresh handles HTMX refresh requests
func (h *DefectHandler) Refresh(c *fiber.Ctx) error {
	boardID := c.Query("board", "")
	if boardID == "" {
		boardID = h.service.GetBoardID()
	}

	// Get sprint ID from query parameter (0 = active sprint)
	sprintID := c.QueryInt("sprint", 0)

	data, err := h.service.GetDefectDashboardDataForSprint(boardID, sprintID)
	if err != nil {
		log.Printf("Error refreshing defect data: %v", err)
		return c.Status(500).SendString("Error refreshing data")
	}

	// Set content type and render only the content part for HTMX swap
	c.Set("Content-Type", "text/html; charset=utf-8")
	return templates.DefectDashboardContent(*data).Render(c.Context(), c.Response().BodyWriter())
}

// GetSprints returns all sprints for the board as JSON
func (h *DefectHandler) GetSprints(c *fiber.Ctx) error {
	boardID := c.Query("board", "")
	if boardID == "" {
		boardID = h.service.GetBoardID()
	}

	sprints, err := h.service.GetAllSprints(boardID)
	if err != nil {
		log.Printf("Error fetching sprints: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sprints"})
	}

	return c.JSON(sprints)
}
