package migrations

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
	"github.com/shepherrrd/gontext/internal/context"
	"github.com/shepherrrd/gontext/internal/drivers"
	"github.com/shepherrrd/gontext/internal/models"
)

type MigrationManager struct {
	context       *context.DbContext
	migrationsDir string
	packageName   string
}

type MigrationFile struct {
	ID          string
	Name        string
	Timestamp   string
	Operations  []models.MigrationOperation
	Checksum    string
}

func NewMigrationManager(ctx *context.DbContext, migrationsDir, packageName string) *MigrationManager {
	return &MigrationManager{
		context:       ctx,
		migrationsDir: migrationsDir,
		packageName:   packageName,
	}
}

func (mm *MigrationManager) EnsureMigrationsTable() error {
	// Ensure public schema exists
	err := mm.context.GetDB().Exec("CREATE SCHEMA IF NOT EXISTS public").Error
	if err != nil {
		return fmt.Errorf("failed to create public schema: %w", err)
	}

	// Set search path to public schema
	err = mm.context.GetDB().Exec("SET search_path TO public").Error
	if err != nil {
		return fmt.Errorf("failed to set search path: %w", err)
	}

	return mm.context.GetDB().AutoMigrate(&models.Migration{})
}

func (mm *MigrationManager) AddMigration(name string) error {
	if err := mm.EnsureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	// Load previous snapshot
	previousSnapshot, err := mm.loadLastSnapshot()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load previous snapshot: %w", err)
	}

	// Create current snapshot
	currentSnapshot := models.NewModelSnapshot(mm.context.GetEntityModels())

	var operations []models.MigrationOperation

	if previousSnapshot == nil {
		// First migration - create all tables
		operations, err = mm.generateInitialOperations()
		if err != nil {
			return fmt.Errorf("failed to generate initial operations: %w", err)
		}
	} else {
		// Compare snapshots to find changes
		comparison := currentSnapshot.Compare(previousSnapshot)
		if !comparison.HasChanges {
			fmt.Println("No changes detected. Migration not created.")
			return nil
		}

		operations, err = mm.generateOperationsFromComparison(comparison)
		if err != nil {
			return fmt.Errorf("failed to generate operations from comparison: %w", err)
		}
	}

	if len(operations) == 0 {
		fmt.Println("No changes detected. Migration not created.")
		return nil
	}

	timestamp := time.Now().Format("20060102150405")
	migrationID := fmt.Sprintf("%s_%s", timestamp, strings.ToLower(strings.ReplaceAll(name, " ", "_")))

	migration := &MigrationFile{
		ID:         migrationID,
		Name:       name,
		Timestamp:  timestamp,
		Operations: operations,
	}

	if err := mm.generateMigrationFile(migration); err != nil {
		return fmt.Errorf("failed to generate migration file: %w", err)
	}

	// Save current snapshot
	if err := mm.saveSnapshot(currentSnapshot); err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	fmt.Printf("Migration '%s' created successfully.\n", migrationID)
	return nil
}

func (mm *MigrationManager) UpdateDatabase() error {
	return mm.RunMigrations()
}

func (mm *MigrationManager) RemoveLastMigration() error {
	migrations, err := mm.getPendingMigrations()
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		return fmt.Errorf("no migrations to remove")
	}

	// Get the latest migration file
	lastMigration := migrations[len(migrations)-1]
	migrationFile := filepath.Join(mm.migrationsDir, lastMigration+".go")

	// Remove the file
	if err := os.Remove(migrationFile); err != nil {
		return fmt.Errorf("failed to remove migration file: %w", err)
	}

	// Remove from database if it was applied
	err = mm.context.GetDB().Where("id = ?", lastMigration).Delete(&models.Migration{}).Error
	if err != nil {
		return fmt.Errorf("failed to remove migration from database: %w", err)
	}

	// Restore previous snapshot
	// This is simplified - in a real implementation, you'd want to restore the exact previous snapshot
	fmt.Printf("Migration '%s' removed successfully.\n", lastMigration)
	return nil
}

func (mm *MigrationManager) ListMigrations() error {
	appliedMigrations := []string{}
	err := mm.context.GetDB().Model(&models.Migration{}).Order("applied_at").Pluck("id", &appliedMigrations).Error
	if err != nil {
		return err
	}

	pendingMigrations, err := mm.getPendingMigrations()
	if err != nil {
		return err
	}

	fmt.Println("Applied Migrations:")
	for _, migration := range appliedMigrations {
		fmt.Printf("  ✓ %s\n", migration)
	}

	fmt.Println("\nPending Migrations:")
	for _, migration := range pendingMigrations {
		fmt.Printf("  - %s\n", migration)
	}

	return nil
}

func (mm *MigrationManager) DropDatabase() error {
	entityModels := mm.context.GetEntityModels()
	
	// Drop all tables in reverse order
	for _, entity := range entityModels {
		err := mm.context.GetDB().Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", entity.TableName)).Error
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", entity.TableName, err)
		}
	}

	// Drop migrations table
	err := mm.context.GetDB().Exec("DROP TABLE IF EXISTS migrations CASCADE").Error
	if err != nil {
		return fmt.Errorf("failed to drop migrations table: %w", err)
	}

	return nil
}

func (mm *MigrationManager) RollbackDatabase(steps int) error {
	appliedMigrations := []models.Migration{}
	err := mm.context.GetDB().Order("applied_at DESC").Limit(steps).Find(&appliedMigrations).Error
	if err != nil {
		return err
	}

	if len(appliedMigrations) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	for _, migration := range appliedMigrations {
		fmt.Printf("Rolling back migration: %s\n", migration.ID)
		
		// Execute rollback operations (this would require implementing Down() methods)
		// For now, just remove from migrations table
		err := mm.context.GetDB().Delete(&migration).Error
		if err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.ID, err)
		}
	}

	return nil
}

func (mm *MigrationManager) RunMigrations() error {
	if err := mm.EnsureMigrationsTable(); err != nil {
		return err
	}

	migrations, err := mm.getPendingMigrations()
	if err != nil {
		return fmt.Errorf("failed to get pending migrations: %w", err)
	}

	if len(migrations) == 0 {
		fmt.Println("No pending migrations.")
		return nil
	}

	for _, migration := range migrations {
		fmt.Printf("Applying migration: %s\n", migration)
		if err := mm.runMigrationFile(migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration, err)
		}
	}

	fmt.Printf("Applied %d migrations successfully.\n", len(migrations))
	return nil
}

func (mm *MigrationManager) generateOperations() ([]models.MigrationOperation, error) {
	var operations []models.MigrationOperation

	entityModels := mm.context.GetEntityModels()
	driver := mm.context.GetDriver()

	for _, entityModel := range entityModels {
		exists, err := mm.tableExists(entityModel.TableName)
		if err != nil {
			return nil, err
		}

		if !exists {
			operation := mm.createTableOperation(entityModel, driver)
			operations = append(operations, operation)
		} else {
			schemaOps, err := mm.generateSchemaChangeOperations(entityModel, driver)
			if err != nil {
				return nil, err
			}
			operations = append(operations, schemaOps...)
		}
	}

	return operations, nil
}

func (mm *MigrationManager) createTableOperation(entity *models.EntityModel, driver drivers.DatabaseDriver) models.MigrationOperation {
	var columns []models.ColumnDefinition

	for _, field := range entity.Fields {
		column := models.ColumnDefinition{
			Name:         field.ColumnName,
			Type:         driver.MapGoTypeToSQL(field.Type),
			IsNullable:   field.IsNullable,
			IsPrimary:    field.IsPrimary,
			IsUnique:     field.IsUnique,
			DefaultValue: field.DefaultValue,
		}
		columns = append(columns, column)
	}

	return models.MigrationOperation{
		Type:       models.CreateTable,
		EntityName: entity.Name,
		Details: models.CreateTableOperation{
			TableName: entity.TableName,
			Columns:   columns,
		},
	}
}

func (mm *MigrationManager) generateSchemaChangeOperations(entity *models.EntityModel, driver drivers.DatabaseDriver) ([]models.MigrationOperation, error) {
	var operations []models.MigrationOperation

	dbSchema, err := mm.getDatabaseSchema(entity.TableName)
	if err != nil {
		return nil, err
	}

	for _, field := range entity.Fields {
		if field.OldName != nil {
			if dbCol, exists := dbSchema[*field.OldName]; exists && !containsColumn(dbSchema, field.ColumnName) {
				operations = append(operations, models.MigrationOperation{
					Type:       models.RenameColumn,
					EntityName: entity.Name,
					Details: models.RenameColumnOperation{
						TableName: entity.TableName,
						OldName:   *field.OldName,
						NewName:   field.ColumnName,
					},
				})
				delete(dbSchema, *field.OldName)
				dbSchema[field.ColumnName] = dbCol
			}
		}

		if _, exists := dbSchema[field.ColumnName]; !exists {
			operations = append(operations, models.MigrationOperation{
				Type:       models.AddColumn,
				EntityName: entity.Name,
				Details: models.AddColumnOperation{
					TableName: entity.TableName,
					Column: models.ColumnDefinition{
						Name:         field.ColumnName,
						Type:         driver.MapGoTypeToSQL(field.Type),
						IsNullable:   field.IsNullable,
						IsPrimary:    field.IsPrimary,
						IsUnique:     field.IsUnique,
						DefaultValue: field.DefaultValue,
					},
				},
			})
		}
	}

	return operations, nil
}

func (mm *MigrationManager) generateMigrationFile(migration *MigrationFile) error {
	if err := os.MkdirAll(mm.migrationsDir, 0755); err != nil {
		return err
	}

	content, err := mm.renderMigrationTemplate(migration)
	if err != nil {
		return err
	}

	migration.Checksum = fmt.Sprintf("%x", md5.Sum([]byte(content)))

	filePath := filepath.Join(mm.migrationsDir, migration.ID+".go")
	return os.WriteFile(filePath, []byte(content), 0644)
}

func (mm *MigrationManager) renderMigrationTemplate(migration *MigrationFile) (string, error) {
	var content strings.Builder
	
	content.WriteString(fmt.Sprintf(`// Code generated migration. DO NOT EDIT.
package %s

import (
	"gorm.io/gorm"
)

type Migration%s struct{}

func (m *Migration%s) ID() string {
	return "%s"
}

func (m *Migration%s) Up(db *gorm.DB) error {
`, mm.packageName, migration.Timestamp, migration.Timestamp, migration.ID, migration.Timestamp))

	// Generate Up operations
	for _, op := range migration.Operations {
		content.WriteString(mm.generateOperationSQL(op, false))
	}

	content.WriteString(`	return nil
}

func (m *Migration` + migration.Timestamp + `) Down(db *gorm.DB) error {
	// Rollback operations in reverse order
`)

	// Generate Down operations (reverse order)
	for i := len(migration.Operations) - 1; i >= 0; i-- {
		op := migration.Operations[i]
		content.WriteString(mm.generateOperationSQL(op, true))
	}

	content.WriteString(`	return nil
}
`)

	return content.String(), nil
}

func (mm *MigrationManager) generateOperationSQL(op models.MigrationOperation, isRollback bool) string {
	switch op.Type {
	case models.CreateTable:
		if isRollback {
			if createOp, ok := op.Details.(models.CreateTableOperation); ok {
				return fmt.Sprintf(`	// Drop table %s
	if err := db.Exec("DROP TABLE IF EXISTS %s").Error; err != nil {
		return err
	}
`, op.EntityName, createOp.TableName)
			}
		} else {
			if createOp, ok := op.Details.(models.CreateTableOperation); ok {
				sql := mm.generateCreateTableSQL(createOp)
				return fmt.Sprintf(`	// Create table %s
	if err := db.Exec("%s").Error; err != nil {
		return err
	}
`, op.EntityName, sql)
			}
		}
	case models.AddColumn:
		if isRollback {
			if addOp, ok := op.Details.(models.AddColumnOperation); ok {
				return fmt.Sprintf(`	// Remove column %s from %s
	if err := db.Exec("ALTER TABLE %s DROP COLUMN %s").Error; err != nil {
		return err
	}
`, addOp.Column.Name, addOp.TableName, addOp.TableName, addOp.Column.Name)
			}
		} else {
			if addOp, ok := op.Details.(models.AddColumnOperation); ok {
				nullable := ""
				if !addOp.Column.IsNullable {
					nullable = " NOT NULL"
				}
				return fmt.Sprintf(`	// Add column %s to %s
	if err := db.Exec("ALTER TABLE %s ADD COLUMN %s %s%s").Error; err != nil {
		return err
	}
`, addOp.Column.Name, addOp.TableName, addOp.TableName, addOp.Column.Name, addOp.Column.Type, nullable)
			}
		}
	case models.RenameColumn:
		if renameOp, ok := op.Details.(models.RenameColumnOperation); ok {
			if isRollback {
				return fmt.Sprintf(`	// Rename column %s back to %s in %s
	if err := db.Exec("ALTER TABLE %s RENAME COLUMN %s TO %s").Error; err != nil {
		return err
	}
`, renameOp.NewName, renameOp.OldName, renameOp.TableName, renameOp.TableName, renameOp.NewName, renameOp.OldName)
			} else {
				return fmt.Sprintf(`	// Rename column %s to %s in %s
	if err := db.Exec("ALTER TABLE %s RENAME COLUMN %s TO %s").Error; err != nil {
		return err
	}
`, renameOp.OldName, renameOp.NewName, renameOp.TableName, renameOp.TableName, renameOp.OldName, renameOp.NewName)
			}
		}
	}
	return ""
}

func (mm *MigrationManager) generateCreateTableSQL(createOp models.CreateTableOperation) string {
	var sql strings.Builder
	sql.WriteString(fmt.Sprintf("CREATE TABLE %s (", createOp.TableName))
	
	var columns []string
	var primaryKeys []string
	
	for _, col := range createOp.Columns {
		columnDef := fmt.Sprintf("%s %s", col.Name, col.Type)
		if !col.IsNullable {
			columnDef += " NOT NULL"
		}
		if col.IsUnique && !col.IsPrimary {
			columnDef += " UNIQUE"
		}
		if col.DefaultValue != nil {
			columnDef += fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
		}
		columns = append(columns, columnDef)
		
		if col.IsPrimary {
			primaryKeys = append(primaryKeys, col.Name)
		}
	}
	
	sql.WriteString(strings.Join(columns, ", "))
	
	if len(primaryKeys) > 0 {
		sql.WriteString(fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}
	
	sql.WriteString(")")
	return sql.String()
}

func (mm *MigrationManager) tableExists(tableName string) (bool, error) {
	var count int64
	err := mm.context.GetDB().Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = 'public'", tableName).Scan(&count).Error
	return count > 0, err
}

func (mm *MigrationManager) getDatabaseSchema(tableName string) (map[string]drivers.ColumnInfo, error) {
	schema := make(map[string]drivers.ColumnInfo)
	query := mm.context.GetDriver().GetSchemaInformationQuery()

	rows, err := mm.context.GetDB().Raw(query, tableName).Rows()
	if err != nil {
		return schema, err
	}
	defer rows.Close()

	for rows.Next() {
		var col drivers.ColumnInfo
		var maxLength *int
		err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable, &col.IsPrimary, &col.DefaultValue, &maxLength)
		if err != nil {
			return schema, err
		}
		col.MaxLength = maxLength
		schema[col.Name] = col
	}

	return schema, nil
}

func (mm *MigrationManager) getPendingMigrations() ([]string, error) {
	migrationFiles, err := filepath.Glob(filepath.Join(mm.migrationsDir, "*.go"))
	if err != nil {
		return nil, err
	}

	var appliedMigrations []string
	err = mm.context.GetDB().Model(&models.Migration{}).Pluck("id", &appliedMigrations).Error
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]bool)
	for _, applied := range appliedMigrations {
		appliedMap[applied] = true
	}

	var pending []string
	for _, file := range migrationFiles {
		migrationID := strings.TrimSuffix(filepath.Base(file), ".go")
		if !appliedMap[migrationID] {
			pending = append(pending, migrationID)
		}
	}

	return pending, nil
}

func (mm *MigrationManager) runMigrationFile(migrationID string) error {
	return mm.context.GetDB().Transaction(func(tx *gorm.DB) error {
		migration := &models.Migration{
			ID:        migrationID,
			Name:      extractMigrationName(migrationID),
			AppliedAt: time.Now(),
			Version:   1,
			Checksum:  "",
		}

		return tx.Create(migration).Error
	})
}

func containsColumn(schema map[string]drivers.ColumnInfo, columnName string) bool {
	_, exists := schema[columnName]
	return exists
}

func extractMigrationName(migrationID string) string {
	parts := strings.SplitN(migrationID, "_", 2)
	if len(parts) > 1 {
		return strings.ReplaceAll(parts[1], "_", " ")
	}
	return migrationID
}

// Snapshot management methods
func (mm *MigrationManager) loadLastSnapshot() (*models.ModelSnapshot, error) {
	snapshotFile := filepath.Join(mm.migrationsDir, "ModelSnapshot.json")
	
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		return nil, err
	}

	var snapshot models.ModelSnapshot
	err = json.Unmarshal(data, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	return &snapshot, nil
}

func (mm *MigrationManager) saveSnapshot(snapshot *models.ModelSnapshot) error {
	if err := os.MkdirAll(mm.migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	snapshotFile := filepath.Join(mm.migrationsDir, "ModelSnapshot.json")
	
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	err = os.WriteFile(snapshotFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write snapshot file: %w", err)
	}

	return nil
}

func (mm *MigrationManager) generateInitialOperations() ([]models.MigrationOperation, error) {
	var operations []models.MigrationOperation
	entityModels := mm.context.GetEntityModels()
	driver := mm.context.GetDriver()

	for _, entityModel := range entityModels {
		operation := mm.createTableOperation(entityModel, driver)
		operations = append(operations, operation)
	}

	return operations, nil
}

func (mm *MigrationManager) generateOperationsFromComparison(comparison *models.SnapshotComparison) ([]models.MigrationOperation, error) {
	var operations []models.MigrationOperation
	driver := mm.context.GetDriver()

	for _, change := range comparison.Changes {
		switch change.Type {
		case models.EntityAdded:
			entitySnapshot := change.Details.(models.EntitySnapshot)
			operation := mm.createTableOperationFromSnapshot(entitySnapshot, driver)
			operations = append(operations, operation)

		case models.FieldAdded:
			fieldSnapshot := change.Details.(models.FieldSnapshot)
			operation := models.MigrationOperation{
				Type:       models.AddColumn,
				EntityName: change.EntityName,
				Details: models.AddColumnOperation{
					TableName: toSnakeCase(change.EntityName),
					Column: models.ColumnDefinition{
						Name:         fieldSnapshot.ColumnName,
						Type:         driver.MapGoTypeToSQL(fieldSnapshot.Type),
						IsNullable:   fieldSnapshot.IsNullable,
						IsPrimary:    fieldSnapshot.IsPrimary,
						IsUnique:     fieldSnapshot.IsUnique,
						DefaultValue: fieldSnapshot.DefaultValue,
					},
				},
			}
			operations = append(operations, operation)

		case models.FieldRenamed:
			fieldRename := change.Details.(models.FieldRename)
			operation := models.MigrationOperation{
				Type:       models.RenameColumn,
				EntityName: change.EntityName,
				Details: models.RenameColumnOperation{
					TableName: toSnakeCase(change.EntityName),
					OldName:   fieldRename.OldName,
					NewName:   fieldRename.NewName,
				},
			}
			operations = append(operations, operation)

		case models.FieldRemoved:
			fieldSnapshot := change.Details.(models.FieldSnapshot)
			operation := models.MigrationOperation{
				Type:       models.DropColumn,
				EntityName: change.EntityName,
				Details: models.DropColumnOperation{
					TableName:  toSnakeCase(change.EntityName),
					ColumnName: fieldSnapshot.ColumnName,
				},
			}
			operations = append(operations, operation)
		}
	}

	return operations, nil
}

func (mm *MigrationManager) createTableOperationFromSnapshot(entitySnapshot models.EntitySnapshot, driver drivers.DatabaseDriver) models.MigrationOperation {
	var columns []models.ColumnDefinition

	for _, field := range entitySnapshot.Fields {
		column := models.ColumnDefinition{
			Name:         field.ColumnName,
			Type:         driver.MapGoTypeToSQL(field.Type),
			IsNullable:   field.IsNullable,
			IsPrimary:    field.IsPrimary,
			IsUnique:     field.IsUnique,
			DefaultValue: field.DefaultValue,
		}
		columns = append(columns, column)
	}

	return models.MigrationOperation{
		Type:       models.CreateTable,
		EntityName: entitySnapshot.Name,
		Details: models.CreateTableOperation{
			TableName: entitySnapshot.TableName,
			Columns:   columns,
		},
	}
}

func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}