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
	scrumMasterService := services.NewScrumMasterService(jiraService, dashboardService)

	// Initialize handlers
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	defectHandler := handlers.NewDefectHandler(defectDashboardService)
	engineerHandler := handlers.NewEngineerHandler(engineerDashboardService, jiraService)
	scrumMasterHandler := handlers.NewScrumMasterHandler(scrumMasterService, jiraService)
	homeHandler := handlers.NewHomeHandler()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Scrum Insights",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Static files
	app.Static("/static", "./static")

	// Routes
	app.Get("/", homeHandler.Index)
	app.Get("/sprint", dashboardHandler.Index)
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

	// Scrum Master Dashboard Routes
	app.Get("/scrum-master", scrumMasterHandler.Dashboard)
	app.Get("/scrum-master/api/refresh", scrumMasterHandler.Refresh)
	app.Get("/scrum-master/api/velocity", scrumMasterHandler.GetVelocityData)
	app.Get("/scrum-master/api/health", scrumMasterHandler.GetTeamHealth)
	app.Get("/scrum-master/api/blockers", scrumMasterHandler.GetBlockers)
	app.Get("/scrum-master/api/risks", scrumMasterHandler.GetRisks)

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Scrum Dashboard starting on http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}
