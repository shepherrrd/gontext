package models

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ModelSnapshot struct {
	Version   string                     `json:"version"`
	Timestamp time.Time                  `json:"timestamp"`
	Entities  map[string]EntitySnapshot  `json:"entities"`
	Checksum  string                     `json:"checksum"`
}

type EntitySnapshot struct {
	Name      string                    `json:"name"`
	TableName string                    `json:"table_name"`
	Fields    map[string]FieldSnapshot  `json:"fields"`
	Indexes   []IndexSnapshot           `json:"indexes"`
}

type FieldSnapshot struct {
	Name         string                 `json:"name"`
	ColumnName   string                 `json:"column_name"`
	Type         string                 `json:"type"`
	IsPrimary    bool                   `json:"is_primary"`
	IsNullable   bool                   `json:"is_nullable"`
	IsUnique     bool                   `json:"is_unique"`
	DefaultValue *string                `json:"default_value"`
	Tags         map[string]string      `json:"tags"`
}

type IndexSnapshot struct {
	Name     string   `json:"name"`
	Columns  []string `json:"columns"`
	IsUnique bool     `json:"is_unique"`
}

func NewModelSnapshot(entities map[string]*EntityModel) *ModelSnapshot {
	snapshot := &ModelSnapshot{
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Entities:  make(map[string]EntitySnapshot),
	}

	for _, entity := range entities {
		entitySnapshot := EntitySnapshot{
			Name:      entity.Name,
			TableName: entity.TableName,
			Fields:    make(map[string]FieldSnapshot),
			Indexes:   []IndexSnapshot{},
		}

		for fieldName, field := range entity.Fields {
			fieldSnapshot := FieldSnapshot{
				Name:         field.Name,
				ColumnName:   field.ColumnName,
				Type:         field.Type,
				IsPrimary:    field.IsPrimary,
				IsNullable:   field.IsNullable,
				IsUnique:     field.IsUnique,
				DefaultValue: field.DefaultValue,
				Tags:         field.Tags,
			}
			entitySnapshot.Fields[fieldName] = fieldSnapshot
		}

		snapshot.Entities[entity.Name] = entitySnapshot
	}

	// Calculate checksum
	snapshot.Checksum = snapshot.calculateChecksum()
	return snapshot
}

func (s *ModelSnapshot) calculateChecksum() string {
	// Create a stable representation for checksum
	data := make(map[string]interface{})
	data["version"] = s.Version
	data["entities"] = s.Entities

	jsonData, _ := json.Marshal(data)
	return fmt.Sprintf("%x", md5.Sum(jsonData))
}

func (s *ModelSnapshot) Compare(other *ModelSnapshot) *SnapshotComparison {
	comparison := &SnapshotComparison{
		HasChanges: false,
		Changes:    []SnapshotChange{},
	}

	// Compare entities
	for entityName, currentEntity := range s.Entities {
		if otherEntity, exists := other.Entities[entityName]; exists {
			entityChanges := s.compareEntities(currentEntity, otherEntity)
			comparison.Changes = append(comparison.Changes, entityChanges...)
		} else {
			// New entity
			comparison.Changes = append(comparison.Changes, SnapshotChange{
				Type:       EntityAdded,
				EntityName: entityName,
				Details:    currentEntity,
			})
		}
	}

	// Check for removed entities
	for entityName, otherEntity := range other.Entities {
		if _, exists := s.Entities[entityName]; !exists {
			comparison.Changes = append(comparison.Changes, SnapshotChange{
				Type:       EntityRemoved,
				EntityName: entityName,
				Details:    otherEntity,
			})
		}
	}

	comparison.HasChanges = len(comparison.Changes) > 0
	return comparison
}

func (s *ModelSnapshot) compareEntities(current, other EntitySnapshot) []SnapshotChange {
	var changes []SnapshotChange
	
	// First pass: identify all renames to avoid double-processing
	renamedFields := make(map[string]string) // oldName -> newName
	fieldsInvolved := make(map[string]bool)  // track fields involved in renames
	
	// Find all rename operations first
	for oldFieldName, otherField := range other.Fields {
		if _, exists := current.Fields[oldFieldName]; !exists {
			// Field doesn't exist with same name, check if it's renamed
			if newFieldName := s.findRenamedField(otherField, current.Fields); newFieldName != nil {
				renamedFields[oldFieldName] = *newFieldName
				fieldsInvolved[oldFieldName] = true
				fieldsInvolved[*newFieldName] = true
				
				// Add rename operation
				changes = append(changes, SnapshotChange{
					Type:       FieldRenamed,
					EntityName: current.Name,
					FieldName:  &oldFieldName,
					Details: FieldRename{
						OldName: oldFieldName,
						NewName: *newFieldName,
						Field:   current.Fields[*newFieldName],
					},
				})
			}
		}
	}

	// Second pass: handle field modifications and additions (excluding renamed fields)
	for fieldName, currentField := range current.Fields {
		if fieldsInvolved[fieldName] {
			// Skip fields involved in renames
			continue
		}
		
		if otherField, exists := other.Fields[fieldName]; exists {
			// Check for field modifications
			if !s.fieldsEqual(currentField, otherField) {
				changes = append(changes, SnapshotChange{
					Type:       FieldModified,
					EntityName: current.Name,
					FieldName:  &fieldName,
					Details: FieldComparison{
						Old: otherField,
						New: currentField,
					},
				})
			}
		} else {
			// New field (not involved in rename)
			changes = append(changes, SnapshotChange{
				Type:       FieldAdded,
				EntityName: current.Name,
				FieldName:  &fieldName,
				Details:    currentField,
			})
		}
	}

	// Third pass: handle field removals (excluding renamed fields)
	for fieldName, otherField := range other.Fields {
		if fieldsInvolved[fieldName] {
			// Skip fields involved in renames
			continue
		}
		
		if _, exists := current.Fields[fieldName]; !exists {
			// Field was removed (not renamed)
			changes = append(changes, SnapshotChange{
				Type:       FieldRemoved,
				EntityName: current.Name,
				FieldName:  &fieldName,
				Details:    otherField,
			})
		}
	}

	return changes
}

func (s *ModelSnapshot) findRenamedField(oldField FieldSnapshot, currentFields map[string]FieldSnapshot) *string {
	// First check for explicit old_name tag
	for fieldName, currentField := range currentFields {
		if oldName, exists := currentField.Tags["old_name"]; exists {
			if oldName == oldField.ColumnName || oldName == oldField.Name {
				return &fieldName
			}
		}
	}
	
	// Enhanced heuristic-based rename detection
	// Look for fields that match the old field's characteristics but have different names
	var candidates []string
	
	for fieldName, currentField := range currentFields {
		// Skip if field name is the same (not a rename)
		if fieldName == oldField.Name {
			continue
		}
		
		// Check if the field types and characteristics match exactly
		if s.fieldsMatch(oldField, currentField) {
			candidates = append(candidates, fieldName)
		}
	}
	
	// If we found exactly one candidate, it's likely a rename
	if len(candidates) == 1 {
		return &candidates[0]
	}
	
	// If multiple candidates, try to find the best match using name similarity
	if len(candidates) > 1 {
		bestMatch := s.findBestNameMatch(oldField.Name, candidates)
		if bestMatch != nil {
			return bestMatch
		}
	}
	
	return nil
}

// fieldsMatch checks if two fields have identical characteristics (type, constraints, etc.)
func (s *ModelSnapshot) fieldsMatch(field1, field2 FieldSnapshot) bool {
	return field1.Type == field2.Type &&
		field1.IsPrimary == field2.IsPrimary &&
		field1.IsNullable == field2.IsNullable &&
		field1.IsUnique == field2.IsUnique &&
		s.defaultValuesMatch(field1.DefaultValue, field2.DefaultValue)
}

// defaultValuesMatch compares default values properly handling nil cases
func (s *ModelSnapshot) defaultValuesMatch(val1, val2 *string) bool {
	if val1 == nil && val2 == nil {
		return true
	}
	if val1 != nil && val2 != nil {
		return *val1 == *val2
	}
	return false
}

// findBestNameMatch finds the most similar field name using simple string similarity
func (s *ModelSnapshot) findBestNameMatch(oldName string, candidates []string) *string {
	if len(candidates) == 0 {
		return nil
	}
	
	// Simple heuristics for common rename patterns
	oldLower := strings.ToLower(oldName)
	
	for _, candidate := range candidates {
		candidateLower := strings.ToLower(candidate)
		
		// Check for common patterns like UpdatedAt -> UpdatedAtTime
		if strings.HasPrefix(candidateLower, oldLower) || strings.HasPrefix(oldLower, candidateLower) {
			return &candidate
		}
		
		// Check if one contains the other (e.g., "UpdatedAt" and "UpdatedAtTime")
		if strings.Contains(candidateLower, oldLower) || strings.Contains(oldLower, candidateLower) {
			return &candidate
		}
	}
	
	// If no clear pattern match, return the first candidate
	return &candidates[0]
}

func (s *ModelSnapshot) fieldsEqual(field1, field2 FieldSnapshot) bool {
	return field1.Type == field2.Type &&
		field1.IsPrimary == field2.IsPrimary &&
		field1.IsNullable == field2.IsNullable &&
		field1.IsUnique == field2.IsUnique &&
		((field1.DefaultValue == nil && field2.DefaultValue == nil) ||
			(field1.DefaultValue != nil && field2.DefaultValue != nil && *field1.DefaultValue == *field2.DefaultValue))
}

type SnapshotComparison struct {
	HasChanges bool            `json:"has_changes"`
	Changes    []SnapshotChange `json:"changes"`
}

type SnapshotChange struct {
	Type       SnapshotChangeType `json:"type"`
	EntityName string             `json:"entity_name"`
	FieldName  *string            `json:"field_name,omitempty"`
	Details    interface{}        `json:"details"`
}

type SnapshotChangeType int

const (
	EntityAdded SnapshotChangeType = iota
	EntityRemoved
	EntityModified
	FieldAdded
	FieldRemoved
	FieldModified
	FieldRenamed
)

type FieldComparison struct {
	Old FieldSnapshot `json:"old"`
	New FieldSnapshot `json:"new"`
}

type FieldRename struct {
	OldName string        `json:"old_name"`
	NewName string        `json:"new_name"`
	Field   FieldSnapshot `json:"field"`
}