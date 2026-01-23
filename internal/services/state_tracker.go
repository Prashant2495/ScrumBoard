package services

import (
	"sync"
	"time"
)

// StateTransition represents a state change for an item
type StateTransition struct {
	ItemKey         string    `json:"itemKey"`
	ItemTitle       string    `json:"itemTitle"`
	ItemType        string    `json:"itemType"` // "story", "subtask", or "defect"
	FromState       string    `json:"fromState"`
	ToState         string    `json:"toState"`
	Assignee        string    `json:"assignee"`
	ChangedAt       time.Time `json:"changedAt"`
	SprintName      string    `json:"sprintName"`
	ParentStoryKey  string    `json:"parentStoryKey"`  // For subtasks: parent story key
	ParentStoryName string    `json:"parentStoryName"` // For subtasks: parent story title
}

// StateTracker tracks state changes for items
type StateTracker struct {
	mu          sync.RWMutex
	transitions []StateTransition
	lastStates  map[string]string // itemKey -> last known state
	lastCleared time.Time
}

var stateTrackerInstance *StateTracker
var stateTrackerOnce sync.Once

// GetStateTracker returns singleton state tracker
func GetStateTracker() *StateTracker {
	stateTrackerOnce.Do(func() {
		stateTrackerInstance = &StateTracker{
			transitions: make([]StateTransition, 0),
			lastStates:  make(map[string]string),
			lastCleared: time.Now(),
		}
	})
	return stateTrackerInstance
}

// CheckAndRecordTransition checks if state changed and records it
func (s *StateTracker) CheckAndRecordTransition(itemKey, itemTitle, itemType, currentState, assignee, sprintName string) {
	s.CheckAndRecordTransitionWithParent(itemKey, itemTitle, itemType, currentState, assignee, sprintName, "", "")
}

// CheckAndRecordTransitionWithParent checks if state changed and records it with parent story info
func (s *StateTracker) CheckAndRecordTransitionWithParent(itemKey, itemTitle, itemType, currentState, assignee, sprintName, parentKey, parentName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastState, exists := s.lastStates[itemKey]

	// If state changed, record transition
	if exists && lastState != currentState {
		transition := StateTransition{
			ItemKey:         itemKey,
			ItemTitle:       itemTitle,
			ItemType:        itemType,
			FromState:       lastState,
			ToState:         currentState,
			Assignee:        assignee,
			ChangedAt:       time.Now(),
			SprintName:      sprintName,
			ParentStoryKey:  parentKey,
			ParentStoryName: parentName,
		}
		s.transitions = append(s.transitions, transition)
	}

	// Update last known state
	s.lastStates[itemKey] = currentState
}

// GetTodaysTransitions returns all transitions from today
func (s *StateTracker) GetTodaysTransitions() []StateTransition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := time.Now().Truncate(24 * time.Hour)
	var todaysTransitions []StateTransition

	for _, t := range s.transitions {
		if t.ChangedAt.After(today) || t.ChangedAt.Equal(today) {
			todaysTransitions = append(todaysTransitions, t)
		}
	}

	return todaysTransitions
}

// GetTransitionsSince returns transitions since a specific time
func (s *StateTracker) GetTransitionsSince(since time.Time) []StateTransition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []StateTransition
	for _, t := range s.transitions {
		if t.ChangedAt.After(since) {
			result = append(result, t)
		}
	}
	return result
}

// ClearOldTransitions removes transitions older than 7 days
func (s *StateTracker) ClearOldTransitions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -7)
	var kept []StateTransition

	for _, t := range s.transitions {
		if t.ChangedAt.After(cutoff) {
			kept = append(kept, t)
		}
	}

	s.transitions = kept
	s.lastCleared = time.Now()
}

// GetTransitionStats returns stats about transitions
func (s *StateTracker) GetTransitionStats() map[string]int {
	transitions := s.GetTodaysTransitions()

	stats := map[string]int{
		"total":        len(transitions),
		"toDone":       0,
		"toInProgress": 0,
		"toBlocked":    0,
	}

	for _, t := range transitions {
		switch t.ToState {
		case "Done", "Closed", "Resolved", "Accepted":
			stats["toDone"]++
		case "In Progress", "In Development", "In Review":
			stats["toInProgress"]++
		case "Blocked", "On Hold":
			stats["toBlocked"]++
		}
	}

	return stats
}
