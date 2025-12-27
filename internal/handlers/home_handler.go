package handlers

import (
	"ScrumBoard/templates"

	"github.com/gofiber/fiber/v2"
)

type HomeHandler struct {
}

func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

func (h *HomeHandler) Index(c *fiber.Ctx) error {
	// For now, create empty sprint list
	// In future, can fetch from Jira if needed
	modelSprints := []templates.SprintInfo{}

	// Set content type to HTML
	c.Set("Content-Type", "text/html; charset=utf-8")

	// Render home page
	return templates.Home(modelSprints).Render(c.Context(), c.Response().BodyWriter())
}
