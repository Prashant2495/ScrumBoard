package handlers

import (
	"ScrumBoard/internal/services"
	"ScrumBoard/templates"
	"strings"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
)

// DashboardHandler handles dashboard routes
type DashboardHandler struct {
	dashboardService *services.DashboardService
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(ds *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: ds}
}

// render is a helper to render templ components
func render(c *fiber.Ctx, component templ.Component) error {
	c.Set("Content-Type", "text/html")
	handler := adaptor.HTTPHandler(templ.Handler(component))
	return handler(c)
}

// Index renders the main dashboard page
func (h *DashboardHandler) Index(c *fiber.Ctx) error {
	boardID := h.dashboardService.GetBoardID() // Always use default board
	sprintID := c.QueryInt("sprint", 0)        // 0 = active sprint

	// If sprintName is provided but sprint ID is 0, search for sprint by name
	sprintName := c.Query("sprintName")
	if sprintID == 0 && sprintName != "" {
		sprints, err := h.dashboardService.GetAllSprints(boardID)
		if err == nil {
			for _, sprint := range sprints {
				if strings.EqualFold(sprint.Name, sprintName) {
					sprintID = sprint.ID
					break
				}
			}
		}
	}

	data, err := h.dashboardService.GetDashboardDataForSprint(boardID, sprintID)
	if err != nil {
		return c.Status(500).SendString("Error fetching dashboard data: " + err.Error())
	}

	return render(c, templates.Dashboard(*data))
}

// Refresh returns updated dashboard content (for HTMX)
func (h *DashboardHandler) Refresh(c *fiber.Ctx) error {
	boardID := h.dashboardService.GetBoardID() // Always use default board
	sprintID := c.QueryInt("sprint", 0)        // 0 = active sprint

	data, err := h.dashboardService.GetDashboardDataForSprint(boardID, sprintID)
	if err != nil {
		return c.Status(500).SendString("Error refreshing data: " + err.Error())
	}

	return render(c, templates.DashboardContent(*data))
}

// GetBoards returns all boards as JSON
func (h *DashboardHandler) GetBoards(c *fiber.Ctx) error {
	boards, err := h.dashboardService.GetBoards()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch boards"})
	}

	return c.JSON(boards)
}

// GetSprints returns all sprints for the board as JSON
func (h *DashboardHandler) GetSprints(c *fiber.Ctx) error {
	boardID := c.Query("board", h.dashboardService.GetBoardID())

	sprints, err := h.dashboardService.GetAllSprints(boardID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sprints"})
	}

	return c.JSON(sprints)
}

// SearchSprint searches for a sprint by name
func (h *DashboardHandler) SearchSprint(c *fiber.Ctx) error {
	name := c.Query("name")
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Sprint name is required"})
	}

	boardID := c.Query("board", h.dashboardService.GetBoardID())
	sprints, err := h.dashboardService.GetAllSprints(boardID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch sprints"})
	}

	// Search for matching sprint (case-insensitive)
	nameUpper := strings.ToUpper(name)
	for _, sprint := range sprints {
		if strings.ToUpper(sprint.Name) == nameUpper {
			return c.JSON(fiber.Map{
				"id":    sprint.ID,
				"name":  sprint.Name,
				"state": sprint.State,
			})
		}
	}

	return c.JSON(fiber.Map{"error": "Sprint not found", "name": name})
}
