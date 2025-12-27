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
	webexService := services.NewWebexService()

	// Initialize handlers
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	defectHandler := handlers.NewDefectHandler(defectDashboardService)
	engineerHandler := handlers.NewEngineerHandler(engineerDashboardService, jiraService)
	scrumMasterHandler := handlers.NewScrumMasterHandler(scrumMasterService, jiraService, webexService)
	homeHandler := handlers.NewHomeHandler()

	// Start daily risk scheduler (9 AM daily, notifies you)
	schedulerConfig := services.GetSchedulerConfig()
	scheduler := services.NewScheduler(jiraService, schedulerConfig)
	scheduler.Start()

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
	app.Post("/engineer/api/ping", engineerHandler.PingEngineer)
	app.Post("/engineer/api/ping/respond", engineerHandler.RespondToPing)
	app.Get("/engineer/api/pings", engineerHandler.GetPings)
	app.Get("/api/risk/items", engineerHandler.GetAtRiskItems)
	app.Post("/api/risk/alert", engineerHandler.AlertAtRiskItems)

	// Scrum Master Dashboard Routes
	app.Get("/scrum-master", scrumMasterHandler.Dashboard)
	app.Get("/scrum-master/api/refresh", scrumMasterHandler.Refresh)
	app.Get("/scrum-master/api/velocity", scrumMasterHandler.GetVelocityData)
	app.Get("/scrum-master/api/health", scrumMasterHandler.GetTeamHealth)
	app.Get("/scrum-master/api/blockers", scrumMasterHandler.GetBlockers)
	app.Get("/scrum-master/api/risks", scrumMasterHandler.GetRisks)
	app.Post("/scrum-master/api/ping", scrumMasterHandler.PingUser)
	app.Get("/scrum-master/api/webex-status", scrumMasterHandler.CheckWebexStatus)

	// Request Info API (per story/defect)
	app.Post("/api/request-info", scrumMasterHandler.RequestInfo)

	// Webex Webhook for receiving replies
	app.Post("/api/webex/webhook", scrumMasterHandler.WebexWebhook)

	// Manual trigger for daily risk report (for testing)
	app.Post("/api/risk/daily-report", func(c *fiber.Ctx) error {
		scheduler.RunNow()
		return c.JSON(fiber.Map{"success": true, "message": "Daily risk report triggered"})
	})

	// Get port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Scrum Dashboard starting on http://localhost:%s", port)
	log.Fatal(app.Listen(":" + port))
}
