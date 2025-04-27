package statemanager

import (
	"encoding/json"
	"fmt"
	"sync"
)

type InMemoryStateManager struct {
	mu     sync.Mutex
	states map[int64]string
	data   map[int64]map[string][]byte // Store data as JSON-encoded bytes
}

func NewInMemoryStateManager() *InMemoryStateManager {
	return &InMemoryStateManager{
		states: make(map[int64]string),
		data:   make(map[int64]map[string][]byte),
	}
}

// SetState sets the state for a given chatID.
func (f *InMemoryStateManager) SetState(chatID int64, state string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[chatID] = state
}

// GetState retrieves the state for a given chatID.
func (f *InMemoryStateManager) GetState(chatID int64) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.states[chatID]
}

// SetData sets arbitrary data for a given chatID and key.
func (f *InMemoryStateManager) SetData(chatID int64, key string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Ensure the chatID map exists
	if _, exists := f.data[chatID]; !exists {
		f.data[chatID] = make(map[string][]byte)
	}

	// Serialize the value to JSON
	valueJSON, err := json.Marshal(value)
	if err != nil {
		fmt.Printf("Error marshalling data: %v\n", err)
		return
	}

	// Store the serialized data
	f.data[chatID][key] = valueJSON
}

// GetData retrieves arbitrary data for a given chatID and key.
func (f *InMemoryStateManager) GetData(chatID int64, key string) any {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Retrieve the serialized data
	if dataMap, exists := f.data[chatID]; exists {
		if jsonData, ok := dataMap[key]; ok {
			// If it's not a string slice, try unmarshalling into a generic interface{}
			var value any

			err := json.Unmarshal(jsonData, &value)
			if err != nil {
				fmt.Printf("Error unmarshalling data: %v\n", err)
				return nil
			}

			return value
		}
	}

	return nil // Key does not exist
}
