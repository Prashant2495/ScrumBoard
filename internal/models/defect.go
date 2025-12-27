package models

import "time"

// Defect represents a bug/defect from Jira
type Defect struct {
	ID             string    `json:"id"`
	Key            string    `json:"key"`
	Summary        string    `json:"summary"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`         // Open, In Progress, Resolved, Closed, etc.
	StatusCategory string    `json:"statusCategory"` // done, indeterminate, new
	Priority       string    `json:"priority"`       // Blocker, Critical, Major, Minor, Trivial
	Severity       string    `json:"severity"`       // Critical, High, Medium, Low
	Assignee       User      `json:"assignee"`
	Reporter       User      `json:"reporter"`
	Labels         []string  `json:"labels"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	AgeInDays      int       `json:"ageInDays"`
	ResolutionTime int       `json:"resolutionTime"` // in hours, 0 if not resolved
	Environment    string    `json:"environment"`    // Production, Staging, Dev
	AffectedVersion string   `json:"affectedVersion"`
}

// DefectStats holds calculated statistics for defects
type DefectStats struct {
	TotalDefects     int `json:"totalDefects"`
	OpenDefects      int `json:"openDefects"`
	InProgressDefects int `json:"inProgressDefects"`
	ResolvedDefects  int `json:"resolvedDefects"`
	ClosedDefects    int `json:"closedDefects"`
	
	// By Severity
	CriticalDefects int `json:"criticalDefects"`
	HighDefects     int `json:"highDefects"`
	MediumDefects   int `json:"mediumDefects"`
	LowDefects      int `json:"lowDefects"`
	
	// By Priority
	BlockerDefects  int `json:"blockerDefects"`
	MajorDefects    int `json:"majorDefects"`
	MinorDefects    int `json:"minorDefects"`
	TrivialDefects  int `json:"trivialDefects"`
	
	// Metrics
	AvgResolutionTimeHours float64 `json:"avgResolutionTimeHours"`
	OldestDefectDays       int     `json:"oldestDefectDays"`
	DefectsAgedOver7Days   int     `json:"defectsAgedOver7Days"`
	DefectsAgedOver30Days  int     `json:"defectsAgedOver30Days"`
}

// DefectByAssignee holds defect statistics per assignee
type DefectByAssignee struct {
	User            User `json:"user"`
	TotalDefects    int  `json:"totalDefects"`
	OpenDefects     int  `json:"openDefects"`
	ResolvedDefects int  `json:"resolvedDefects"`
	CriticalDefects int  `json:"criticalDefects"`
	HighDefects     int  `json:"highDefects"`
}

// DefectBySeverity holds defect count by severity level
type DefectBySeverity struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
	Open     int    `json:"open"`
	Resolved int    `json:"resolved"`
}

// DefectByPriority holds defect count by priority level
type DefectByPriority struct {
	Priority string `json:"priority"`
	Count    int    `json:"count"`
	Open     int    `json:"open"`
	Resolved int    `json:"resolved"`
}

// DefectDashboardData holds all data needed for the defect dashboard
type DefectDashboardData struct {
	Sprint            Sprint             `json:"sprint"`
	Stats             DefectStats        `json:"stats"`
	Defects           []Defect           `json:"defects"`
	AssigneeStats     []DefectByAssignee `json:"assigneeStats"`
	SeverityBreakdown []DefectBySeverity `json:"severityBreakdown"`
	PriorityBreakdown []DefectByPriority `json:"priorityBreakdown"`
	OpenDefects       []Defect           `json:"openDefects"`
	InProgressDefects []Defect           `json:"inProgressDefects"`
	ResolvedDefects   []Defect           `json:"resolvedDefects"`
}

