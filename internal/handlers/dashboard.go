package handlers

import (
	"ScrumBoard/internal/services"
	"ScrumBoard/templates"

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
	boardID := c.Query("board", h.dashboardService.GetBoardID())

	data, err := h.dashboardService.GetDashboardData(boardID)
	if err != nil {
		return c.Status(500).SendString("Error fetching dashboard data: " + err.Error())
	}

	return render(c, templates.Dashboard(*data))
}

// Refresh returns updated dashboard content (for HTMX)
func (h *DashboardHandler) Refresh(c *fiber.Ctx) error {
	boardID := c.Query("board", h.dashboardService.GetBoardID())

	data, err := h.dashboardService.GetDashboardData(boardID)
	if err != nil {
		return c.Status(500).SendString("Error refreshing data: " + err.Error())
	}

	return render(c, templates.DashboardContent(*data))
}
