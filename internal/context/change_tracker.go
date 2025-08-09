package context

import (
	"sync"
)

type EntityState int

const (
	EntityUnchanged EntityState = iota
	EntityAdded
	EntityModified
	EntityDeleted
)

type EntityEntry struct {
	Entity interface{}
	State  EntityState
}

type ChangeTracker struct {
	entries map[interface{}]*EntityEntry
	mu      sync.RWMutex
}

func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		entries: make(map[interface{}]*EntityEntry),
	}
}

func (ct *ChangeTracker) Add(entity interface{}, state EntityState) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.entries[entity] = &EntityEntry{
		Entity: entity,
		State:  state,
	}
}

func (ct *ChangeTracker) GetState(entity interface{}) EntityState {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if entry, exists := ct.entries[entity]; exists {
		return entry.State
	}
	return EntityUnchanged
}

func (ct *ChangeTracker) GetChanges() map[interface{}]*EntityEntry {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make(map[interface{}]*EntityEntry)
	for k, v := range ct.entries {
		if v.State != EntityUnchanged {
			result[k] = v
		}
	}
	return result
}

func (ct *ChangeTracker) Clear() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.entries = make(map[interface{}]*EntityEntry)
}

func (ct *ChangeTracker) HasChanges() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	for _, entry := range ct.entries {
		if entry.State != EntityUnchanged {
			return true
		}
	}
	return false
}