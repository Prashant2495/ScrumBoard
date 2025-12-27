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
	defectDashboardService := services.NewDefectDashboardService(jiraService)
	engineerDashboardService := services.NewEngineerDashboardService(jiraService)

	// Initialize handlers
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	defectHandler := handlers.NewDefectHandler(defectDashboardService)
	engineerHandler := handlers.NewEngineerHandler(engineerDashboardService, jiraService)

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
	app.Get("/api/boards", dashboardHandler.GetBoards)
	app.Get("/api/sprints", dashboardHandler.GetSprints)

	// Defect Dashboard Routes
	app.Get("/defects", defectHandler.Index)
	app.Get("/defects/api/refresh", defectHandler.Refresh)
	app.Get("/defects/api/sprints", defectHandler.GetSprints)

	// Engineer Dashboard Routes
	app.Get("/engineer", engineerHandler.HandleEngineerDashboard)
	app.Get("/engineer/api/refresh", engineerHandler.HandleEngineerRefresh)

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Scrum Dashboard starting on http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}
