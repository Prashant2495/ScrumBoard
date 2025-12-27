package main

import (
	"log"
	"os"

	"ScrumBoard/internal/handlers"
	"ScrumBoard/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if exists
	godotenv.Load()

	// Initialize services
	jiraService := services.NewJiraService()
	dashboardService := services.NewDashboardService(jiraService)

	// Initialize handlers
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Scrum Dashboard",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Static files
	app.Static("/static", "./static")

	// Routes
	app.Get("/", dashboardHandler.Index)
	app.Get("/api/refresh", dashboardHandler.Refresh)

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Scrum Dashboard starting on http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}

