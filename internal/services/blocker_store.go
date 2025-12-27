package services

import (
	"ScrumBoard/internal/models"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ReportedBlocker represents a blocker reported via Webex
type ReportedBlocker struct {
	ID            string    `json:"id"`
	ReporterEmail string    `json:"reporterEmail"`
	ReporterName  string    `json:"reporterName"`
	ItemKey       string    `json:"itemKey"`   // Related story/defect key if any
	ItemTitle     string    `json:"itemTitle"` // Related item title if any
	Description   string    `json:"description"`
	ReportedAt    time.Time `json:"reportedAt"`
	ResolvedAt    *time.Time `json:"resolvedAt,omitempty"`
	Status        string    `json:"status"` // "active", "resolved"
	SprintName    string    `json:"sprintName"`
}

// BlockerStore manages reported blockers
type BlockerStore struct {
	blockers map[string]*ReportedBlocker
	mu       sync.RWMutex
}

var blockerStoreInstance *BlockerStore
var blockerStoreOnce sync.Once

// GetBlockerStore returns singleton blocker store
func GetBlockerStore() *BlockerStore {
	blockerStoreOnce.Do(func() {
		blockerStoreInstance = &BlockerStore{
			blockers: make(map[string]*ReportedBlocker),
		}
	})
	return blockerStoreInstance
}

// SaveBlocker saves a new blocker
func (s *BlockerStore) SaveBlocker(reporterEmail, reporterName, description, itemKey, itemTitle, sprintName string) *ReportedBlocker {
	s.mu.Lock()
	defer s.mu.Unlock()

	blocker := &ReportedBlocker{
		ID:            uuid.New().String(),
		ReporterEmail: reporterEmail,
		ReporterName:  reporterName,
		ItemKey:       itemKey,
		ItemTitle:     itemTitle,
		Description:   description,
		ReportedAt:    time.Now(),
		Status:        "active",
		SprintName:    sprintName,
	}

	s.blockers[blocker.ID] = blocker
	return blocker
}

// GetActiveBlockers returns all active blockers
func (s *BlockerStore) GetActiveBlockers() []ReportedBlocker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []ReportedBlocker
	for _, b := range s.blockers {
		if b.Status == "active" {
			result = append(result, *b)
		}
	}
	return result
}

// GetAllBlockers returns all blockers
func (s *BlockerStore) GetAllBlockers() []ReportedBlocker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []ReportedBlocker
	for _, b := range s.blockers {
		result = append(result, *b)
	}
	return result
}

// ResolveBlocker marks a blocker as resolved
func (s *BlockerStore) ResolveBlocker(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if blocker, ok := s.blockers[id]; ok {
		now := time.Now()
		blocker.ResolvedAt = &now
		blocker.Status = "resolved"
		return true
	}
	return false
}

// ToModelBlocker converts ReportedBlocker to models.Blocker for dashboard
func (b *ReportedBlocker) ToModelBlocker() models.Blocker {
	ageInDays := int(time.Since(b.ReportedAt).Hours() / 24)
	return models.Blocker{
		ID:          b.ID,
		Key:         b.ItemKey,
		Title:       b.Description,
		Description: b.Description,
		Priority:    "High",
		Status:      b.Status,
		AgeInDays:   ageInDays,
		AssignedTo:  b.ReporterName,
		CreatedAt:   b.ReportedAt.Format("2006-01-02"),
	}
}

// DetectBlocker checks if message contains blocker keywords
func DetectBlocker(message string) bool {
	blockerKeywords := []string{
		"blocked", "blocker", "blocking", "impediment",
		"stuck", "waiting for", "dependency", "can't proceed",
		"cannot proceed", "need help", "issue with",
	}

	lowerMsg := strings.ToLower(message)
	for _, keyword := range blockerKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}
	return false
}

