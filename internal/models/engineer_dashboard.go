package models

// EngineerDashboardData holds all data for engineer dashboard
type EngineerDashboardData struct {
	Engineer    Engineer
	Sprint      Sprint
	Stories     []Story
	Defects     []Defect
	Stats       EngineerStats
	StoryStats  StoryStatusStats
	DefectStats DefectStatusStats
}

// Engineer represents an engineer
type Engineer struct {
	UserID      string
	Name        string
	Email       string
	AvatarColor string
}

// EngineerStats holds statistics for an engineer
type EngineerStats struct {
	TotalStories      int
	CompletedStories  int
	InProgressStories int
	TodoStories       int
	TotalDefects      int
	ResolvedDefects   int
	OpenDefects       int
	InProgressDefects int
	StoryPoints       int
	CompletedPoints   int
}

// StoryStatusStats holds story breakdown by status
type StoryStatusStats struct {
	Todo       int
	InProgress int
	Done       int
}

// DefectStatusStats holds defect breakdown by status
type DefectStatusStats struct {
	Open       int
	InProgress int
	Resolved   int
	Closed     int
}

// PingMessage represents an info request sent to an engineer
type PingMessage struct {
	ID            string `json:"id"`
	EngineerEmail string `json:"engineer_email"`
	EngineerName  string `json:"engineer_name"`
	SprintName    string `json:"sprint_name"`
	ItemKey       string `json:"item_key"`   // Story/Defect key e.g. "PMOB-1234"
	ItemTitle     string `json:"item_title"` // Story/Defect title
	ItemType      string `json:"item_type"`  // "story" or "defect"
	Message       string `json:"message"`
	SentAt        string `json:"sent_at"`
	Response      string `json:"response"`
	RespondedAt   string `json:"responded_at"`
	Status        string `json:"status"` // "pending", "responded"
}
