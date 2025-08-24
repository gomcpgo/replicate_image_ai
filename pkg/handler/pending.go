package handler

import (
	"sync"
	"time"
)

// PendingOperation represents an in-progress operation
type PendingOperation struct {
	PredictionID string
	StorageID    string
	Operation    string
	StartTime    time.Time
	Model        string
	Params       map[string]interface{}
}

// PendingOperationsManager manages in-progress operations
type PendingOperationsManager struct {
	operations map[string]*PendingOperation
	mu         sync.RWMutex
}

// NewPendingOperationsManager creates a new pending operations manager
func NewPendingOperationsManager() *PendingOperationsManager {
	pom := &PendingOperationsManager{
		operations: make(map[string]*PendingOperation),
	}
	// Start cleanup goroutine to remove expired operations
	go pom.cleanupExpired()
	return pom
}

// Add stores a new pending operation
func (pom *PendingOperationsManager) Add(predictionID string, op *PendingOperation) {
	pom.mu.Lock()
	defer pom.mu.Unlock()
	pom.operations[predictionID] = op
}

// Get retrieves a pending operation by prediction ID
func (pom *PendingOperationsManager) Get(predictionID string) (*PendingOperation, bool) {
	pom.mu.RLock()
	defer pom.mu.RUnlock()
	op, exists := pom.operations[predictionID]
	return op, exists
}

// Remove deletes a pending operation
func (pom *PendingOperationsManager) Remove(predictionID string) {
	pom.mu.Lock()
	defer pom.mu.Unlock()
	delete(pom.operations, predictionID)
}

// cleanupExpired removes operations older than 10 minutes
func (pom *PendingOperationsManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		pom.mu.Lock()
		now := time.Now()
		for id, op := range pom.operations {
			if now.Sub(op.StartTime) > 10*time.Minute {
				delete(pom.operations, id)
			}
		}
		pom.mu.Unlock()
	}
}

// EstimateRemainingTime estimates remaining time based on operation type
func EstimateRemainingTime(operation string, elapsed time.Duration) int {
	// Typical operation times (in seconds)
	typicalTimes := map[string]int{
		"generate_image":              30,
		"generate_with_visual_context": 45,
		"edit_image":                  35,
		"remove_background":           20,
		"upscale_image":               25,
		"enhance_face":                20,
		"restore_photo":               30,
	}
	
	typical, ok := typicalTimes[operation]
	if !ok {
		typical = 30 // default
	}
	
	remaining := typical - int(elapsed.Seconds())
	if remaining < 5 {
		remaining = 5 // minimum estimate
	}
	if remaining > 60 {
		remaining = 60 // maximum estimate
	}
	
	return remaining
}