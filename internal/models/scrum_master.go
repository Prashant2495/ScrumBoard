package models

// ScrumMasterDashboardData holds all data for scrum master dashboard
type ScrumMasterDashboardData struct {
	Sprint          Sprint
	Stats           SprintStats
	UserStats       []UserStats
	VelocityData    VelocityData
	TeamHealth      TeamHealth
	Blockers        []Blocker
	SprintHistory   []SprintSummary
	RiskIndicators  []RiskIndicator
}

// VelocityData holds sprint velocity metrics
type VelocityData struct {
	CurrentVelocity  int            `json:"currentVelocity"`
	AverageVelocity  int            `json:"averageVelocity"`
	VelocityTrend    string         `json:"velocityTrend"` // up, down, stable
	TrendPercentage  int            `json:"trendPercentage"`
	SprintVelocities []SprintVelocity `json:"sprintVelocities"`
	PredictedNext    int            `json:"predictedNext"`
}

// SprintVelocity holds velocity for a single sprint
type SprintVelocity struct {
	SprintName string `json:"sprintName"`
	SprintID   int    `json:"sprintId"`
	Committed  int    `json:"committed"`
	Completed  int    `json:"completed"`
	Carryover  int    `json:"carryover"`
}

// TeamHealth holds team health metrics
type TeamHealth struct {
	OverallScore     int              `json:"overallScore"`     // 0-100
	HealthStatus     string           `json:"healthStatus"`     // healthy, at-risk, critical
	CapacityBalance  CapacityBalance  `json:"capacityBalance"`
	WorkDistribution WorkDistribution `json:"workDistribution"`
	CommitmentRatio  float64          `json:"commitmentRatio"`  // committed vs completed
	SprintBurndown   []BurndownPoint  `json:"sprintBurndown"`
}

// CapacityBalance shows team capacity utilization
type CapacityBalance struct {
	TotalCapacity     int             `json:"totalCapacity"`
	UsedCapacity      int             `json:"usedCapacity"`
	OverloadedMembers []TeamMember    `json:"overloadedMembers"`
	UnderutilizedMbrs []TeamMember    `json:"underutilizedMembers"`
}

// TeamMember represents a team member with load info
type TeamMember struct {
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	AssignedPoints int     `json:"assignedPoints"`
	Capacity       int     `json:"capacity"`
	LoadPercentage float64 `json:"loadPercentage"`
}

// WorkDistribution shows work type distribution
type WorkDistribution struct {
	Stories    int `json:"stories"`
	Defects    int `json:"defects"`
	Tasks      int `json:"tasks"`
	TechDebt   int `json:"techDebt"`
	Unplanned  int `json:"unplanned"`
}

// BurndownPoint represents a point in burndown chart
type BurndownPoint struct {
	Date           string `json:"date"`
	Day            int    `json:"day"`
	RemainingPoints int   `json:"remainingPoints"`
	IdealRemaining int    `json:"idealRemaining"`
}

// Blocker represents an impediment/blocker
type Blocker struct {
	ID           string `json:"id"`
	Key          string `json:"key"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	Priority     string `json:"priority"`
	Status       string `json:"status"`
	AgeInDays    int    `json:"ageInDays"`
	AssignedTo   string `json:"assignedTo"`
	BlockedItems []string `json:"blockedItems"`
	CreatedAt    string `json:"createdAt"`
}

// SprintSummary holds summary of a past sprint
type SprintSummary struct {
	SprintID      int     `json:"sprintId"`
	SprintName    string  `json:"sprintName"`
	StartDate     string  `json:"startDate"`
	EndDate       string  `json:"endDate"`
	Committed     int     `json:"committed"`
	Completed     int     `json:"completed"`
	Carryover     int     `json:"carryover"`
	DefectsFound  int     `json:"defectsFound"`
	DefectsFixed  int     `json:"defectsFixed"`
	HealthScore   int     `json:"healthScore"`
	CompletionPct float64 `json:"completionPct"`
}

// RiskIndicator represents a sprint risk
type RiskIndicator struct {
	Type        string `json:"type"`        // scope-creep, velocity-drop, blocker, capacity
	Severity    string `json:"severity"`    // high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Metric      string `json:"metric"`
	Threshold   string `json:"threshold"`
}

// SprintHealthScore calculates overall sprint health
type SprintHealthScore struct {
	Score           int    `json:"score"`
	Status          string `json:"status"`
	VelocityHealth  int    `json:"velocityHealth"`
	CapacityHealth  int    `json:"capacityHealth"`
	BlockerHealth   int    `json:"blockerHealth"`
	ProgressHealth  int    `json:"progressHealth"`
}

