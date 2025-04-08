package statemanager_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/central-university-dev/go-z0tedd/internal/infrastructure/statemanager"
)

const (
	key   = "test_key"
	value = "test_value"
)

func TestSetAndGetState(t *testing.T) {
	manager := statemanager.NewInMemoryStateManager()
	chatID := int64(123)
	state := "test_state"

	manager.SetState(chatID, state)
	got := manager.GetState(chatID)
	assert.Equal(t, state, got, "Expected state to match")

	// Test state update
	newState := "updated_state"
	manager.SetState(chatID, newState)
	got = manager.GetState(chatID)
	assert.Equal(t, newState, got, "Expected updated state to match")

	// Test non-existent chatID
	assert.Empty(t, manager.GetState(999), "Expected empty string for non-existent chatID")
}

func TestSetAndGetData(t *testing.T) {
	manager := statemanager.NewInMemoryStateManager()
	chatID := int64(123)

	manager.SetData(chatID, key, value)
	got := manager.GetData(chatID, key)
	assert.Equal(t, value, got, "Expected data to match")

	// Test different data types
	intKey := "int_key"
	intValue := 42
	manager.SetData(chatID, intKey, intValue)
	assert.Equal(t, intValue, manager.GetData(chatID, intKey), "Integer value mismatch")

	// Test non-existent key
	assert.Nil(t, manager.GetData(chatID, "invalid_key"), "Expected nil for non-existent key")

	// Test non-existent chatID
	assert.Nil(t, manager.GetData(999, key), "Expected nil for non-existent chatID")
}

func TestConcurrentSetState(t *testing.T) {
	manager := statemanager.NewInMemoryStateManager()

	var wg sync.WaitGroup

	numRoutines := 100

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			chatID := int64(i)
			state := fmt.Sprintf("state_%d", i)
			manager.SetState(chatID, state)
		}(i)
	}

	wg.Wait()

	for i := 0; i < numRoutines; i++ {
		chatID := int64(i)
		expected := fmt.Sprintf("state_%d", i)
		got := manager.GetState(chatID)
		assert.Equal(t, expected, got, "Expected state to match for chatID %d", chatID)
	}
}

func TestConcurrentSetData(t *testing.T) {
	manager := statemanager.NewInMemoryStateManager()
	chatID := int64(123)
	numRoutines := 100

	var wg sync.WaitGroup

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			manager.SetData(chatID, key, value)
		}(i)
	}

	wg.Wait()

	for i := 0; i < numRoutines; i++ {
		key := fmt.Sprintf("key_%d", i)
		expected := fmt.Sprintf("value_%d", i)
		got := manager.GetData(chatID, key)
		assert.Equal(t, expected, got, "Expected data to match for key %s", key)
	}
}

func TestDataInitialization(t *testing.T) {
	manager := statemanager.NewInMemoryStateManager()
	chatID := int64(123)

	manager.SetData(chatID, key, value)
	dataMap := manager.GetData(chatID, key)
	assert.Equal(t, value, dataMap, "Value mismatch in data map")
}

func TestStateAndDataSeparation(t *testing.T) {
	manager := statemanager.NewInMemoryStateManager()
	chatID := int64(123)
	state := "test_state"

	manager.SetState(chatID, state)
	manager.SetData(chatID, key, value)

	// Check state remains unaffected by data operations
	assert.Equal(t, state, manager.GetState(chatID), "State was modified by data operations")

	// Check data remains unaffected by state operations
	assert.Equal(t, value, manager.GetData(chatID, key), "Data was modified by state operations")
}
