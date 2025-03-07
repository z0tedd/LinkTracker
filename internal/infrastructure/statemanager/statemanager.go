package statemanager

import "sync"

type InMemoryStateManager struct {
	mu     sync.Mutex
	states map[int64]string
	data   map[int64]map[string]any
}

func NewInMemoryStateManager() *InMemoryStateManager {
	return &InMemoryStateManager{
		states: make(map[int64]string),
		data:   make(map[int64]map[string]any),
	}
}

func (f *InMemoryStateManager) SetState(chatID int64, state string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[chatID] = state
}

func (f *InMemoryStateManager) GetState(chatID int64) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.states[chatID]
}

func (f *InMemoryStateManager) SetData(chatID int64, key string, value any) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.data[chatID]; !exists {
		f.data[chatID] = make(map[string]any)
	}

	f.data[chatID][key] = value
}

func (f *InMemoryStateManager) GetData(chatID int64, key string) any {
	f.mu.Lock()
	defer f.mu.Unlock()

	if data, exists := f.data[chatID]; exists {
		return data[key]
	}

	return nil
}
