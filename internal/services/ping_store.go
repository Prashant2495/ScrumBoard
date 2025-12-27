package services

import (
	"ScrumBoard/internal/models"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PingStore stores ping messages in memory
type PingStore struct {
	mu       sync.RWMutex
	messages map[string]*models.PingMessage // ID -> PingMessage
}

// Global ping store instance
var pingStore = &PingStore{
	messages: make(map[string]*models.PingMessage),
}

// GetPingStore returns the global ping store
func GetPingStore() *PingStore {
	return pingStore
}

// SavePing stores a new ping message (legacy - for general pings)
func (ps *PingStore) SavePing(email, name, sprintName, message string) *models.PingMessage {
	return ps.SaveInfoRequest(email, name, sprintName, "", "", "", message)
}

// SaveInfoRequest stores a new info request for a specific item
func (ps *PingStore) SaveInfoRequest(email, name, sprintName, itemKey, itemTitle, itemType, message string) *models.PingMessage {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ping := &models.PingMessage{
		ID:            uuid.New().String(),
		EngineerEmail: email,
		EngineerName:  name,
		SprintName:    sprintName,
		ItemKey:       itemKey,
		ItemTitle:     itemTitle,
		ItemType:      itemType,
		Message:       message,
		SentAt:        time.Now().Format("2006-01-02 15:04:05"),
		Status:        "pending",
	}

	ps.messages[ping.ID] = ping
	return ping
}

// SaveResponse saves a response to a ping
func (ps *PingStore) SaveResponse(pingID, response string) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ping, exists := ps.messages[pingID]; exists {
		ping.Response = response
		ping.RespondedAt = time.Now().Format("2006-01-02 15:04:05")
		ping.Status = "responded"
		return true
	}
	return false
}

// GetPingsByEngineer returns all pings for an engineer
func (ps *PingStore) GetPingsByEngineer(email string) []models.PingMessage {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var result []models.PingMessage
	for _, ping := range ps.messages {
		if ping.EngineerEmail == email {
			result = append(result, *ping)
		}
	}
	return result
}

// GetAllPings returns all pings
func (ps *PingStore) GetAllPings() []models.PingMessage {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var result []models.PingMessage
	for _, ping := range ps.messages {
		result = append(result, *ping)
	}
	return result
}

// GetPingByID returns a specific ping
func (ps *PingStore) GetPingByID(id string) *models.PingMessage {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ping, exists := ps.messages[id]; exists {
		return ping
	}
	return nil
}
