package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shepherrrd/gontext"
)

// ModelSnapshot represents the current state of all entities
type ModelSnapshot struct {
	Version   string                 `json:"version"`
	Timestamp string                 `json:"timestamp"`
	Entities  map[string]EntityInfo  `json:"entities"`
}

// EntityInfo holds entity schema information
type EntityInfo struct {
	Name      string            `json:"name"`
	TableName string            `json:"tableName"`
	Fields    map[string]FieldInfo `json:"fields"`
}

// FieldInfo holds field schema information
type FieldInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Tags     string `json:"tags"`
	Nullable bool   `json:"nullable"`
}

// AddMigration creates a new migration with snapshot comparison
func AddMigration(name string) error {
	fmt.Printf("🔄 Adding migration: %s\n", name)
	
	// Create design-time context
	ctx, err := CreateDesignTimeContext()
	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}
	defer ctx.Close()

	// Create migrations directory
	if err := os.MkdirAll("./migrations", 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	// Generate current snapshot
	currentSnapshot, err := generateModelSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate model snapshot: %w", err)
	}

	// Load previous snapshot
	previousSnapshot, err := loadModelSnapshot()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load previous snapshot: %w", err)
	}

	// Compare snapshots and generate migration
	changes := compareSnapshots(previousSnapshot, currentSnapshot)
	if len(changes) == 0 {
		fmt.Println("⚠️  No changes detected in entity models")
		return nil
	}

	fmt.Printf("📊 Detected %d schema changes\n", len(changes))
	for _, change := range changes {
		fmt.Printf("  • %s\n", change)
	}

	// Generate migration file
	if err := generateMigrationFile(name, changes, currentSnapshot); err != nil {
		return fmt.Errorf("failed to generate migration file: %w", err)
	}

	// Save current snapshot
	if err := saveModelSnapshot(currentSnapshot); err != nil {
		return fmt.Errorf("failed to save model snapshot: %w", err)
	}

	fmt.Printf("✅ Migration '%s' created successfully!\n", name)
	return nil
}

// UpdateDatabase applies pending migrations
func UpdateDatabase() error {
	fmt.Println("🔄 Updating database...")
	
	ctx, err := CreateDesignTimeContext()
	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}
	defer ctx.Close()

	// For this example, we'll use EnsureCreated
	// In a real implementation, you'd execute migration files
	if err := ctx.EnsureCreated(); err != nil {
		return fmt.Errorf("failed to update database: %w", err)
	}
	
	fmt.Println("✅ Database updated successfully!")
	return nil
}

// ListMigrations shows all migration files
func ListMigrations() error {
	fmt.Println("📋 Migration files:")
	
	files, err := filepath.Glob("./migrations/*.go")
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}
	
	if len(files) == 0 {
		fmt.Println("   No migration files found")
	} else {
		for i, file := range files {
			fmt.Printf("   %d. %s\n", i+1, filepath.Base(file))
		}
	}
	
	return nil
}

// ShowMigrationStatus shows current migration status
func ShowMigrationStatus() error {
	fmt.Println("📊 Migration Status")
	fmt.Println("==================")
	
	// Check if snapshot exists
	snapshot, err := loadModelSnapshot()
	if err != nil {
		fmt.Println("❌ No model snapshot found")
		fmt.Println("   Run: go run . migrate:add InitialCreate")
		return nil
	}

	fmt.Printf("✅ Model snapshot: %s\n", snapshot.Timestamp)
	fmt.Printf("📊 Tracked entities: %d\n", len(snapshot.Entities))
	
	for name, entity := range snapshot.Entities {
		fmt.Printf("   • %s (%s) - %d fields\n", name, entity.TableName, len(entity.Fields))
	}

	// List migration files
	files, _ := filepath.Glob("./migrations/*.go")
	fmt.Printf("📁 Migration files: %d\n", len(files))
	
	return nil
}

// generateModelSnapshot creates a snapshot of current entity models
func generateModelSnapshot(ctx *gontext.DbContext) (*ModelSnapshot, error) {
	entityModels := ctx.GetEntityModels()
	if len(entityModels) == 0 {
		return nil, fmt.Errorf("no entities registered in context")
	}

	entities := make(map[string]EntityInfo)
	
	for _, model := range entityModels {
		fields := make(map[string]FieldInfo)
		
		for _, field := range model.Fields {
			fields[field.Name] = FieldInfo{
				Name:     field.ColumnName,
				Type:     field.Type,
				Tags:     "", // Could extract from struct tags
				Nullable: field.IsNullable,
			}
		}
		
		entities[model.Name] = EntityInfo{
			Name:      model.Name,
			TableName: model.TableName,
			Fields:    fields,
		}
	}

	return &ModelSnapshot{
		Version:   "1.0",
		Timestamp: time.Now().Format(time.RFC3339),
		Entities:  entities,
	}, nil
}

// loadModelSnapshot loads the existing model snapshot
func loadModelSnapshot() (*ModelSnapshot, error) {
	data, err := os.ReadFile("./migrations/ModelSnapshot.json")
	if err != nil {
		return nil, err
	}

	var snapshot ModelSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse model snapshot: %w", err)
	}

	return &snapshot, nil
}

// saveModelSnapshot saves the model snapshot
func saveModelSnapshot(snapshot *ModelSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return os.WriteFile("./migrations/ModelSnapshot.json", data, 0644)
}

// compareSnapshots compares two snapshots and returns changes
func compareSnapshots(previous, current *ModelSnapshot) []string {
	var changes []string

	if previous == nil {
		// First migration - all entities are new
		for name := range current.Entities {
			changes = append(changes, fmt.Sprintf("Create table: %s", name))
		}
		return changes
	}

	// Check for new entities
	for name := range current.Entities {
		if _, exists := previous.Entities[name]; !exists {
			changes = append(changes, fmt.Sprintf("Create table: %s", name))
		}
	}

	// Check for removed entities
	for name := range previous.Entities {
		if _, exists := current.Entities[name]; !exists {
			changes = append(changes, fmt.Sprintf("Drop table: %s", name))
		}
	}

	// Check for field changes
	for name, currentEntity := range current.Entities {
		if previousEntity, exists := previous.Entities[name]; exists {
			// Check for new fields
			for fieldName := range currentEntity.Fields {
				if _, exists := previousEntity.Fields[fieldName]; !exists {
					changes = append(changes, fmt.Sprintf("Add field: %s.%s", name, fieldName))
				}
			}
			
			// Check for removed fields
			for fieldName := range previousEntity.Fields {
				if _, exists := currentEntity.Fields[fieldName]; !exists {
					changes = append(changes, fmt.Sprintf("Remove field: %s.%s", name, fieldName))
				}
			}
		}
	}

	return changes
}

// generateMigrationFile creates the migration file
func generateMigrationFile(name string, changes []string, snapshot *ModelSnapshot) error {
	timestamp := time.Now().Format("20060102150405")
	migrationID := fmt.Sprintf("%s_%s", timestamp, strings.ReplaceAll(name, " ", ""))
	
	var upStatements, downStatements string
	
	// Generate statements based on changes
	for _, change := range changes {
		if strings.HasPrefix(change, "Create table:") {
			tableName := strings.TrimPrefix(change, "Create table: ")
			if entity, exists := snapshot.Entities[tableName]; exists {
				upStatements += fmt.Sprintf("\t// Create table %s\n", entity.TableName)
				upStatements += fmt.Sprintf("\tif err := db.AutoMigrate(&%s{}); err != nil {\n", tableName)
				upStatements += "\t\treturn err\n\t}\n\n"
				
				downStatements += fmt.Sprintf("\t// Drop table %s\n", entity.TableName)
				downStatements += fmt.Sprintf("\tif err := db.Exec(\"DROP TABLE IF EXISTS %s CASCADE\").Error; err != nil {\n", entity.TableName)
				downStatements += "\t\treturn err\n\t}\n\n"
			}
		}
	}
	
	migrationContent := fmt.Sprintf(`// Code generated migration. DO NOT EDIT.
// Migration: %s
// Created: %s

package migrations

import (
	"gorm.io/gorm"
)

type Migration%s struct{}

func (m *Migration%s) ID() string {
	return "%s"
}

func (m *Migration%s) Up(db *gorm.DB) error {
%s	return nil
}

func (m *Migration%s) Down(db *gorm.DB) error {
%s	return nil
}
`, name, time.Now().Format(time.RFC3339), timestamp, timestamp, migrationID, timestamp, upStatements, timestamp, downStatements)

	filename := filepath.Join("./migrations", migrationID+".go")
	if err := os.WriteFile(filename, []byte(migrationContent), 0644); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}
	
	fmt.Printf("📁 Generated: %s\n", filename)
	return nil
}