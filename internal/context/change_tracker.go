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
	Entity         interface{}
	State          EntityState
	OriginalEntity interface{} // Store original state for change detection
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
		Entity:         entity,
		State:          state,
		OriginalEntity: ct.deepCopy(entity), // Store original state
	}
}

// TrackLoaded tracks an entity that was loaded from the database
// This stores the original state for later change detection
func (ct *ChangeTracker) TrackLoaded(entity interface{}) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	key := ct.entityKey(entity)
	// Only track if not already tracked
	if _, exists := ct.entries[key]; !exists {
		fmt.Printf("[GONTEXT DEBUG] Tracking loaded entity: %s\n", key)
		ct.entries[key] = &EntityEntry{
			Entity:         entity,
			State:          EntityUnchanged,
			OriginalEntity: ct.deepCopy(entity),
		}
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

// DetectChanges automatically detects changes by comparing current state with original
func (ct *ChangeTracker) DetectChanges() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	changeCount := 0
	for key, entry := range ct.entries {
		// Skip if already marked as modified, added, or deleted
		if entry.State != EntityUnchanged {
			continue
		}

		// Compare current entity with original
		if !ct.entitiesEqual(entry.Entity, entry.OriginalEntity) {
			fmt.Printf("[GONTEXT DEBUG] Change detected for entity %s\n", key)
			entry.State = EntityModified
			ct.entries[key] = entry
			changeCount++
		}
	}
	
	if changeCount > 0 {
		fmt.Printf("[GONTEXT DEBUG] DetectChanges found %d modified entities\n", changeCount)
	}
}

// deepCopy creates a deep copy of an entity
func (ct *ChangeTracker) deepCopy(entity interface{}) interface{} {
	if entity == nil {
		return nil
	}

	original := reflect.ValueOf(entity)
	if original.Kind() == reflect.Ptr {
		// Handle pointer
		if original.IsNil() {
			return nil
		}
		originalElem := original.Elem()
		copyPtr := reflect.New(originalElem.Type())
		ct.copyRecursive(originalElem, copyPtr.Elem())
		return copyPtr.Interface()
	}

	// Handle non-pointer
	copy := reflect.New(original.Type()).Elem()
	ct.copyRecursive(original, copy)
	return copy.Interface()
}

// copyRecursive recursively copies values
func (ct *ChangeTracker) copyRecursive(original, copy reflect.Value) {
	switch original.Kind() {
	case reflect.Struct:
		originalType := original.Type()
		for i := 0; i < original.NumField(); i++ {
			field := originalType.Field(i)
			originalField := original.Field(i)
			copyField := copy.Field(i)
			
			// Skip unexported fields - we can't access them safely
			if field.PkgPath != "" {
				continue
			}
			
			// Only copy if both are accessible and the copy field can be set
			if originalField.CanInterface() && copyField.CanSet() {
				ct.copyRecursive(originalField, copyField)
			}
		}
	case reflect.Slice:
		if !original.IsNil() {
			copy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
			for i := 0; i < original.Len(); i++ {
				ct.copyRecursive(original.Index(i), copy.Index(i))
			}
		}
	case reflect.Map:
		if !original.IsNil() {
			copy.Set(reflect.MakeMap(original.Type()))
			for _, key := range original.MapKeys() {
				copyKey := reflect.New(key.Type()).Elem()
				ct.copyRecursive(key, copyKey)
				copyValue := reflect.New(original.MapIndex(key).Type()).Elem()
				ct.copyRecursive(original.MapIndex(key), copyValue)
				copy.SetMapIndex(copyKey, copyValue)
			}
		}
	case reflect.Ptr:
		if !original.IsNil() {
			copy.Set(reflect.New(original.Elem().Type()))
			ct.copyRecursive(original.Elem(), copy.Elem())
		}
	default:
		if copy.CanSet() && original.CanInterface() {
			copy.Set(original)
		}
	}
}

// entitiesEqual compares two entities for equality
func (ct *ChangeTracker) entitiesEqual(entity1, entity2 interface{}) bool {
	if entity1 == nil && entity2 == nil {
		return true
	}
	if entity1 == nil || entity2 == nil {
		return false
	}

	value1 := reflect.ValueOf(entity1)
	value2 := reflect.ValueOf(entity2)

	// Dereference pointers
	if value1.Kind() == reflect.Ptr {
		if value1.IsNil() && value2.IsNil() {
			return true
		}
		if value1.IsNil() || value2.IsNil() {
			return false
		}
		value1 = value1.Elem()
	}
	if value2.Kind() == reflect.Ptr {
		if value2.IsNil() {
			return false
		}
		value2 = value2.Elem()
	}

	// Types must match
	if value1.Type() != value2.Type() {
		return false
	}

	return ct.valuesEqual(value1, value2)
}

// valuesEqual recursively compares reflect.Values
func (ct *ChangeTracker) valuesEqual(value1, value2 reflect.Value) bool {
	if value1.Type() != value2.Type() {
		return false
	}

	switch value1.Kind() {
	case reflect.Struct:
		structType := value1.Type()
		for i := 0; i < value1.NumField(); i++ {
			field := structType.Field(i)
			field1 := value1.Field(i)
			field2 := value2.Field(i)
			
			// Skip unexported fields - we can't access them safely
			if field.PkgPath != "" {
				continue
			}
			
			// Only compare if both fields can be accessed
			if field1.CanInterface() && field2.CanInterface() {
				if !ct.valuesEqual(field1, field2) {
					return false
				}
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if value1.Len() != value2.Len() {
			return false
		}
		for i := 0; i < value1.Len(); i++ {
			if !ct.valuesEqual(value1.Index(i), value2.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Map:
		if value1.Len() != value2.Len() {
			return false
		}
		for _, key := range value1.MapKeys() {
			val1 := value1.MapIndex(key)
			val2 := value2.MapIndex(key)
			if !val2.IsValid() || !ct.valuesEqual(val1, val2) {
				return false
			}
		}
		return true
	case reflect.Ptr:
		if value1.IsNil() && value2.IsNil() {
			return true
		}
		if value1.IsNil() || value2.IsNil() {
			return false
		}
		return ct.valuesEqual(value1.Elem(), value2.Elem())
	default:
		// Use safe comparison for basic types
		if value1.CanInterface() && value2.CanInterface() {
			return reflect.DeepEqual(value1.Interface(), value2.Interface())
		}
		// If we can't get interfaces, try to compare using other methods
		// For basic types, we can compare values directly
		switch value1.Kind() {
		case reflect.Bool:
			return value1.Bool() == value2.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return value1.Int() == value2.Int()
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return value1.Uint() == value2.Uint()
		case reflect.Float32, reflect.Float64:
			return value1.Float() == value2.Float()
		case reflect.Complex64, reflect.Complex128:
			return value1.Complex() == value2.Complex()
		case reflect.String:
			return value1.String() == value2.String()
		default:
			// For other types we can't safely compare, assume they're equal
			return true
		}
	}
}