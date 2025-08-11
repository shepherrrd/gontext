package context

import (
	"crypto/sha256"
	"fmt"
	"reflect"
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
	entries map[string]*EntityEntry  // Use string keys instead of interface{} keys
	mu      sync.RWMutex
}

func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		entries: make(map[string]*EntityEntry),
	}
}

// entityKey generates a unique string key for an entity based on its type and primary key
func (ct *ChangeTracker) entityKey(entity interface{}) string {
	value := reflect.ValueOf(entity)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	
	entityType := value.Type()
	
	// Try to find the primary key field (typically "Id" or "ID")
	var pkValue interface{} = ""
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := entityType.Field(i)
		
		if fieldType.Name == "Id" || fieldType.Name == "ID" {
			if field.IsValid() && field.CanInterface() {
				pkValue = field.Interface()
			}
			break
		}
	}
	
	// If no primary key value found or it's a zero value, create a unique hash based on field values
	if pkValue == "" || pkValue == nil || isZeroValue(reflect.ValueOf(pkValue)) {
		if value.Kind() == reflect.Struct {
			// Create a hash based on hashable field values only
			hash := ct.hashStructFields(value, entityType)
			return fmt.Sprintf("%s:%s", entityType.Name(), hash)
		}
	}
	
	return fmt.Sprintf("%s:%v", entityType.Name(), pkValue)
}

// hashStructFields creates a hash based on hashable field values
func (ct *ChangeTracker) hashStructFields(value reflect.Value, entityType reflect.Type) string {
	hasher := sha256.New()
	
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := entityType.Field(i)
		
		// Skip unexported fields and unhashable field types
		if fieldType.PkgPath != "" || isUnhashableType(field.Type()) {
			continue
		}
		
		if field.IsValid() && field.CanInterface() {
			// Include field name and value in hash
			hasher.Write([]byte(fieldType.Name + ":"))
			hasher.Write([]byte(fmt.Sprintf("%v", field.Interface())))
			hasher.Write([]byte(";"))
		}
	}
	
	return fmt.Sprintf("%x", hasher.Sum(nil))[:16] // Use first 16 chars of hash
}

// isZeroValue checks if a value is a zero value
func isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0.0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Chan:
		return v.Len() == 0
	default:
		// Use reflect.DeepEqual with zero value for other types
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(v.Interface(), zero.Interface())
	}
}

// isUnhashableType checks if a type contains unhashable elements (slices, maps, channels)
func isUnhashableType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return true
	case reflect.Ptr:
		return isUnhashableType(t.Elem())
	case reflect.Array:
		return isUnhashableType(t.Elem())
	case reflect.Struct:
		// Check if any field is unhashable
		for i := 0; i < t.NumField(); i++ {
			if isUnhashableType(t.Field(i).Type) {
				return true
			}
		}
	}
	return false
}

func (ct *ChangeTracker) Add(entity interface{}, state EntityState) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	key := ct.entityKey(entity)
	ct.entries[key] = &EntityEntry{
		Entity: entity,
		State:  state,
	}
}

func (ct *ChangeTracker) GetState(entity interface{}) EntityState {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	key := ct.entityKey(entity)
	if entry, exists := ct.entries[key]; exists {
		return entry.State
	}
	return EntityUnchanged
}

func (ct *ChangeTracker) GetChanges() []*EntityEntry {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var result []*EntityEntry
	for _, v := range ct.entries {
		if v.State != EntityUnchanged {
			result = append(result, v)
		}
	}
	return result
}

func (ct *ChangeTracker) Clear() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.entries = make(map[string]*EntityEntry)
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