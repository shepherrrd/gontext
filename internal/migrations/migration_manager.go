package migrations

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"github.com/shepherrrd/gontext/internal/context"
	"github.com/shepherrrd/gontext/internal/drivers"
	"github.com/shepherrrd/gontext/internal/models"
)

// migrationFields provides statically typed field name access for Migration struct
type migrationFields struct {
	Id        string
	Name      string  
	AppliedAt string
	Version   string
	Checksum  string
	DependsOn string
}

// getMigrationFields returns the actual field names from the Migration struct using reflection
// This ensures type safety - if field names change in the struct, this will update automatically
func getMigrationFields() migrationFields {
	var m models.Migration
	t := reflect.TypeOf(m)
	
	fields := migrationFields{}
	fieldValue := reflect.ValueOf(&fields).Elem()
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name
		
		// Set the corresponding field in migrationFields to the actual struct field name
		switch fieldName {
		case "Id":
			fieldValue.FieldByName("Id").SetString(fieldName)
		case "Name":
			fieldValue.FieldByName("Name").SetString(fieldName)
		case "AppliedAt":
			fieldValue.FieldByName("AppliedAt").SetString(fieldName)
		case "Version":
			fieldValue.FieldByName("Version").SetString(fieldName)
		case "Checksum":
			fieldValue.FieldByName("Checksum").SetString(fieldName)
		case "DependsOn":
			fieldValue.FieldByName("DependsOn").SetString(fieldName)
		}
	}
	
	return fields
}

type MigrationManager struct {
	context       *context.DbContext
	migrationsDir string
	packageName   string
}

type MigrationFile struct {
	Id          string
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
		Id:         migrationID,
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
	fields := getMigrationFields()
	err = mm.context.GetDB().Where(`"`+fields.Id+`" = ?`, lastMigration).Delete(&models.Migration{}).Error
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
	fields := getMigrationFields()
	err := mm.context.GetDB().Model(&models.Migration{}).Order(`"` + fields.AppliedAt + `"`).Pluck(`"` + fields.Id + `"`, &appliedMigrations).Error
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
	
	// Drop all tables in reverse order using double quotes for PostgreSQL case-sensitive names
	for _, entity := range entityModels {
		err := mm.context.GetDB().Exec(fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", entity.TableName)).Error
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
	fields := getMigrationFields()
	// Get most recent migrations first (reverse chronological order)
	err := mm.context.GetDB().Order(`"`+fields.AppliedAt+`" DESC`).Limit(steps).Find(&appliedMigrations).Error
	if err != nil {
		return err
	}

	if len(appliedMigrations) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	for _, migration := range appliedMigrations {
		fmt.Printf("Rolling back migration: %s\n", migration.Id)
		
		// Execute rollback in transaction
		err := mm.context.GetDB().Transaction(func(tx *gorm.DB) error {
			// Execute the rollback operations
			if err := mm.executeRollbackOperations(migration.Id, tx); err != nil {
				return fmt.Errorf("failed to execute rollback operations: %w", err)
			}

			// Remove migration record from database using Where clause
			fields := getMigrationFields()
			err := tx.Where(`"`+fields.Id+`" = ?`, migration.Id).Delete(&models.Migration{}).Error
			if err != nil {
				return fmt.Errorf("failed to remove migration record: %w", err)
			}

			return nil
		})
		
		if err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", migration.Id, err)
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
	var indexes []models.IndexDefinition
	entityModels := mm.context.GetEntityModels() // Get entity models for foreign key resolution

	for _, field := range entity.Fields {
		column := models.ColumnDefinition{
			Name:         field.ColumnName,
			Type:         driver.MapGoTypeToSQL(field.Type),
			IsNullable:   field.IsNullable,
			IsPrimary:    field.IsPrimary,
			IsUnique:     field.IsUnique,
			DefaultValue: field.DefaultValue,
		}

		// Parse GORM tags for additional constraints
		if len(field.Tags) > 0 {
			// Parse foreign key relationships from tags
			if foreignKey := mm.parseForeignKeyFromTags(field.Tags, entity.Name); foreignKey != nil {
				column.References = foreignKey
			}

			// Parse unique indexes
			if _, hasUniqueIndex := field.Tags["uniqueIndex"]; hasUniqueIndex {
				column.IsUnique = true
				indexes = append(indexes, models.IndexDefinition{
					Name:     fmt.Sprintf("idx_%s_%s", entity.TableName, field.ColumnName),
					Columns:  []string{field.ColumnName},
					IsUnique: true,
				})
			}

			// Parse regular indexes  
			if _, hasIndex := field.Tags["index"]; hasIndex {
				indexes = append(indexes, models.IndexDefinition{
					Name:     fmt.Sprintf("idx_%s_%s", entity.TableName, field.ColumnName),
					Columns:  []string{field.ColumnName},
					IsUnique: false,
				})
			}
		}

		// Also check field names for common foreign key patterns (only for UUID fields)
		if column.References == nil && strings.Contains(field.Type, "uuid.UUID") {
			if foreignKey := mm.parseForeignKeyFromFieldName(field.ColumnName, entityModels); foreignKey != nil {
				column.References = foreignKey
			}
		}

		columns = append(columns, column)
	}

	return models.MigrationOperation{
		Type:       models.CreateTable,
		EntityName: entity.Name,
		Details: models.CreateTableOperation{
			TableName: entity.TableName,
			Columns:   columns,
			Indexes:   indexes,
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

	filePath := filepath.Join(mm.migrationsDir, migration.Id+".go")
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
`, mm.packageName, migration.Timestamp, migration.Timestamp, migration.Id, migration.Timestamp))

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
	if err := db.Exec("DROP TABLE IF EXISTS \"%s\"").Error; err != nil {
		return err
	}
`, op.EntityName, createOp.TableName)
			}
		} else {
			if createOp, ok := op.Details.(models.CreateTableOperation); ok {
				sql := mm.generateCreateTableSQL(createOp)
				// Escape quotes in the SQL for Go string literal
				escapedSQL := strings.ReplaceAll(sql, `"`, `\"`)
				return fmt.Sprintf(`	// Create table %s
	if err := db.Exec("%s").Error; err != nil {
		return err
	}
`, op.EntityName, escapedSQL)
			}
		}
	case models.AddColumn:
		if isRollback {
			if addOp, ok := op.Details.(models.AddColumnOperation); ok {
				return fmt.Sprintf(`	// Remove column %s from %s
	if err := db.Exec("ALTER TABLE \\\"%s\\\" DROP COLUMN \\\"%s\\\"").Error; err != nil {
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
	if err := db.Exec("ALTER TABLE \\\"%s\\\" ADD COLUMN \\\"%s\\\" %s%s").Error; err != nil {
		return err
	}
`, addOp.Column.Name, addOp.TableName, addOp.TableName, addOp.Column.Name, addOp.Column.Type, nullable)
			}
		}
	case models.RenameColumn:
		if renameOp, ok := op.Details.(models.RenameColumnOperation); ok {
			if isRollback {
				return fmt.Sprintf(`	// Rename column %s back to %s in %s
	if err := db.Exec("ALTER TABLE \\\"%s\\\" RENAME COLUMN \\\"%s\\\" TO \\\"%s\\\"").Error; err != nil {
		return err
	}
`, renameOp.NewName, renameOp.OldName, renameOp.TableName, renameOp.TableName, renameOp.NewName, renameOp.OldName)
			} else {
				return fmt.Sprintf(`	// Rename column %s to %s in %s
	if err := db.Exec("ALTER TABLE \\\"%s\\\" RENAME COLUMN \\\"%s\\\" TO \\\"%s\\\"").Error; err != nil {
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
	sql.WriteString(fmt.Sprintf("CREATE TABLE \"%s\" (", createOp.TableName))
	
	var columns []string
	var primaryKeys []string
	var foreignKeys []string
	var uniqueConstraints []string
	
	for _, col := range createOp.Columns {
		columnDef := fmt.Sprintf("\"%s\" %s", col.Name, col.Type)
		if !col.IsNullable {
			columnDef += " NOT NULL"
		}
		if col.IsUnique && !col.IsPrimary {
			// Use named unique constraints for better error messages
			uniqueConstraintName := fmt.Sprintf("uni_%s_%s", createOp.TableName, col.Name)
			uniqueConstraints = append(uniqueConstraints, 
				fmt.Sprintf("CONSTRAINT \"%s\" UNIQUE (\"%s\")", uniqueConstraintName, col.Name))
		}
		if col.DefaultValue != nil {
			columnDef += fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
		}
		columns = append(columns, columnDef)
		
		if col.IsPrimary {
			primaryKeys = append(primaryKeys, fmt.Sprintf("\"%s\"", col.Name))
		}
		
		// Add foreign key constraints
		if col.References != nil {
			fkConstraintName := fmt.Sprintf("fk_%s_%s", createOp.TableName, col.Name)
			foreignKeys = append(foreignKeys, 
				fmt.Sprintf("CONSTRAINT \"%s\" FOREIGN KEY (\"%s\") REFERENCES \"%s\" (\"%s\")", 
					fkConstraintName, col.Name, col.References.ReferencedTable, col.References.ReferencedColumn))
		}
	}
	
	sql.WriteString(strings.Join(columns, ", "))
	
	if len(primaryKeys) > 0 {
		sql.WriteString(fmt.Sprintf(", PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}
	
	// Add unique constraints
	for _, uniqueConstraint := range uniqueConstraints {
		sql.WriteString(", ")
		sql.WriteString(uniqueConstraint)
	}
	
	// Add foreign key constraints
	for _, foreignKey := range foreignKeys {
		sql.WriteString(", ")
		sql.WriteString(foreignKey)
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
	fields := getMigrationFields()
	err = mm.context.GetDB().Model(&models.Migration{}).Pluck(`"`+fields.Id+`"`, &appliedMigrations).Error
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

	// Sort pending migrations by timestamp (chronological order)
	// Migration IDs format: 20250810160535_migrationname
	sort.Slice(pending, func(i, j int) bool {
		timestampI := extractTimestamp(pending[i])
		timestampJ := extractTimestamp(pending[j])
		return timestampI < timestampJ
	})

	// Validate dependency order - ensure no migration depends on a later migration
	if err := mm.validateMigrationDependencies(pending, appliedMigrations); err != nil {
		return nil, fmt.Errorf("migration dependency validation failed: %w", err)
	}

	return pending, nil
}

func (mm *MigrationManager) runMigrationFile(migrationID string) error {
	return mm.context.GetDB().Transaction(func(tx *gorm.DB) error {
		// Execute the migration operations directly from the current state
		// This is a simplified approach - in a full implementation, we would parse and execute the Go migration file
		if err := mm.executeMigrationOperations(tx); err != nil {
			return fmt.Errorf("failed to execute migration operations: %w", err)
		}

		// Find the most recent migration to set dependency
		var dependsOn *string
		if lastMigration, err := mm.getLastAppliedMigration(tx); err == nil && lastMigration != nil {
			dependsOn = &lastMigration.Id
		}

		// Record the migration as applied
		migration := &models.Migration{
			Id:        migrationID,
			Name:      extractMigrationName(migrationID),
			AppliedAt: time.Now(),
			Version:   1,
			Checksum:  "",
			DependsOn: dependsOn,
		}

		return tx.Create(migration).Error
	})
}

func (mm *MigrationManager) executeMigrationSQL(migrationID string, tx *gorm.DB) error {
	// For now, let's use a simpler approach - execute the operations from the current migration
	// In the future, this could be enhanced to parse and execute the actual migration file
	
	// Load the migration file operations that were already generated
	previousSnapshot, err := mm.loadLastSnapshot()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load previous snapshot: %w", err)
	}

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
		if comparison.HasChanges {
			operations, err = mm.generateOperationsFromComparison(comparison)
			if err != nil {
				return fmt.Errorf("failed to generate operations from comparison: %w", err)
			}
		}
	}

	// Execute the operations
	for _, op := range operations {
		sql := mm.generateOperationExecutionSQL(op)
		if sql != "" {
			fmt.Printf("Executing SQL: %s\n", sql)
			if err := tx.Exec(sql).Error; err != nil {
				return fmt.Errorf("failed to execute SQL: %s, error: %w", sql, err)
			}
		}
	}
	
	return nil
}

func (mm *MigrationManager) executeMigrationOperations(tx *gorm.DB) error {
	// For initial migrations, use GORM's AutoMigrate to create tables
	entityModelsMap := mm.context.GetEntityModels()
	
	for _, entityModel := range entityModelsMap {
		// Get a pointer to a new instance of the entity type
		entityPtr := reflect.New(entityModel.Type).Interface()
		
		fmt.Printf("Creating table for entity: %s (table: %s)\n", entityModel.Name, entityModel.TableName)
		if err := tx.AutoMigrate(entityPtr); err != nil {
			return fmt.Errorf("failed to auto-migrate entity %s: %w", entityModel.Name, err)
		}
	}
	
	return nil
}

func (mm *MigrationManager) executeRollbackOperations(migrationId string, tx *gorm.DB) error {
	// For initial migrations, rollback means dropping all entity tables
	// This is a simplified approach - in a full implementation, we would parse the Down() method from the migration file
	
	entityModels := mm.context.GetEntityModels()
	
	// Convert map to slice for ordered dropping
	var entityList []*models.EntityModel
	for _, entityModel := range entityModels {
		entityList = append(entityList, entityModel)
	}
	
	// Drop tables in reverse order to handle foreign key dependencies
	for i := len(entityList) - 1; i >= 0; i-- {
		entityModel := entityList[i]
		tableName := entityModel.TableName
		
		fmt.Printf("Dropping table: %s\n", tableName)
		
		// Use quoted table name for PostgreSQL case sensitivity
		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", tableName)
		if err := tx.Exec(dropSQL).Error; err != nil {
			return fmt.Errorf("failed to drop table %s: %w", tableName, err)
		}
	}
	
	return nil
}

func (mm *MigrationManager) getEntityInstance(entityModel *models.EntityModel) interface{} {
	// Create a new instance of the entity type
	return reflect.New(entityModel.Type).Interface()
}

func (mm *MigrationManager) generateOperationExecutionSQL(op models.MigrationOperation) string {
	switch op.Type {
	case models.CreateTable:
		if createOp, ok := op.Details.(models.CreateTableOperation); ok {
			return mm.generateCreateTableSQL(createOp)
		}
	case models.AddColumn:
		if addOp, ok := op.Details.(models.AddColumnOperation); ok {
			nullable := ""
			if !addOp.Column.IsNullable {
				nullable = " NOT NULL"
			}
			defaultVal := ""
			if addOp.Column.DefaultValue != nil {
				defaultVal = fmt.Sprintf(" DEFAULT %s", *addOp.Column.DefaultValue)
			}
			return fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN \"%s\" %s%s%s", 
				addOp.TableName, addOp.Column.Name, addOp.Column.Type, nullable, defaultVal)
		}
	case models.RenameColumn:
		if renameOp, ok := op.Details.(models.RenameColumnOperation); ok {
			return fmt.Sprintf("ALTER TABLE \"%s\" RENAME COLUMN \"%s\" TO \"%s\"", 
				renameOp.TableName, renameOp.OldName, renameOp.NewName)
		}
	case models.DropColumn:
		if dropOp, ok := op.Details.(models.DropColumnOperation); ok {
			return fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN \"%s\"", 
				dropOp.TableName, dropOp.ColumnName)
		}
	}
	return ""
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

// extractTimestamp extracts the timestamp from a migration ID
// Migration ID format: 20250810160535_migrationname
func extractTimestamp(migrationID string) string {
	parts := strings.SplitN(migrationID, "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return migrationID
}

// getLastAppliedMigration gets the most recently applied migration for dependency tracking
func (mm *MigrationManager) getLastAppliedMigration(tx *gorm.DB) (*models.Migration, error) {
	var lastMigration models.Migration
	fields := getMigrationFields()
	
	err := tx.Model(&models.Migration{}).
		Order(`"`+fields.AppliedAt+`" DESC`).
		First(&lastMigration).Error
		
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No migrations applied yet
		}
		return nil, err
	}
	
	return &lastMigration, nil
}

// validateMigrationDependencies ensures that migration dependencies are satisfied
func (mm *MigrationManager) validateMigrationDependencies(pendingMigrations, appliedMigrations []string) error {
	// Create a set of all available migrations (pending + applied)
	availableMigrations := make(map[string]bool)
	for _, migration := range appliedMigrations {
		availableMigrations[migration] = true
	}
	for _, migration := range pendingMigrations {
		availableMigrations[migration] = true
	}
	
	// For timestamp-based dependencies, ensure chronological order
	for i := 1; i < len(pendingMigrations); i++ {
		currentTimestamp := extractTimestamp(pendingMigrations[i])
		previousTimestamp := extractTimestamp(pendingMigrations[i-1])
		
		if currentTimestamp < previousTimestamp {
			return fmt.Errorf("migration %s has timestamp %s which is earlier than previous migration %s with timestamp %s", 
				pendingMigrations[i], currentTimestamp, pendingMigrations[i-1], previousTimestamp)
		}
	}
	
	// Check for chronological conflicts with applied migrations
	if err := mm.detectChronologicalConflicts(pendingMigrations, appliedMigrations); err != nil {
		return fmt.Errorf("chronological conflict detected: %w", err)
	}
	
	fmt.Printf("✅ Migration dependency validation passed for %d pending migrations\n", len(pendingMigrations))
	return nil
}

// detectChronologicalConflicts detects when pending migrations have timestamps that conflict with applied migrations
func (mm *MigrationManager) detectChronologicalConflicts(pendingMigrations, appliedMigrations []string) error {
	if len(appliedMigrations) == 0 {
		return nil // No conflicts possible
	}
	
	// Find the latest applied migration timestamp
	var latestAppliedTimestamp string
	for _, applied := range appliedMigrations {
		timestamp := extractTimestamp(applied)
		if timestamp > latestAppliedTimestamp {
			latestAppliedTimestamp = timestamp
		}
	}
	
	// Check if any pending migration has an older timestamp than the latest applied
	var conflicts []string
	for _, pending := range pendingMigrations {
		pendingTimestamp := extractTimestamp(pending)
		if pendingTimestamp < latestAppliedTimestamp {
			conflicts = append(conflicts, fmt.Sprintf("Migration %s (timestamp: %s) is older than latest applied migration (timestamp: %s)", 
				pending, pendingTimestamp, latestAppliedTimestamp))
		}
	}
	
	if len(conflicts) > 0 {
		fmt.Printf("⚠️  WARNING: Found %d chronological conflicts:\n", len(conflicts))
		for _, conflict := range conflicts {
			fmt.Printf("  - %s\n", conflict)
		}
		fmt.Println("💡 These migrations will be applied in timestamp order, which may cause issues if there are schema dependencies.")
		fmt.Println("💡 Consider recreating these migrations with newer timestamps if they depend on recent schema changes.")
		return nil // Return nil to continue with warning, not error
	}
	
	return nil
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

	// Sort entities by dependencies (parent tables first)
	sortedEntities := mm.sortEntitiesByDependencies(entityModels)

	for _, entityModel := range sortedEntities {
		operation := mm.createTableOperation(entityModel, driver)
		operations = append(operations, operation)
	}

	return operations, nil
}

// sortEntitiesByDependencies sorts entities so parent tables are created before child tables
// Uses dynamic topological sorting based on foreign key relationships detected from GORM tags
func (mm *MigrationManager) sortEntitiesByDependencies(entityModels map[string]*models.EntityModel) []*models.EntityModel {
	// Build dependency graph from foreign key relationships
	dependencies := make(map[string][]string) // entity -> list of entities it depends on
	allEntities := make(map[string]*models.EntityModel)
	
	// Initialize maps
	for _, entity := range entityModels {
		allEntities[entity.Name] = entity
		dependencies[entity.Name] = []string{}
	}
	
	// Analyze each entity for foreign key dependencies
	for _, entity := range entityModels {
		for _, field := range entity.Fields {
			// Check if field has foreign key relationship via GORM tags
			if gormTag, exists := field.Tags["gorm"]; exists {
				if strings.Contains(gormTag, "foreignKey:") {
					// Parse foreignKey tag to find referenced entity
					parts := strings.Split(gormTag, ";")
					for _, part := range parts {
						part = strings.TrimSpace(part)
						if strings.HasPrefix(part, "references:") {
							// This indicates a relationship - find the referenced entity
							// The field type should indicate the referenced entity
							fieldType := strings.TrimPrefix(field.Type, "[]") // Handle slices
							fieldType = strings.TrimPrefix(fieldType, "*")    // Handle pointers
							
							// Check if this type corresponds to another entity
							for _, otherEntity := range entityModels {
								if otherEntity.Name == fieldType {
									dependencies[entity.Name] = append(dependencies[entity.Name], fieldType)
								}
							}
						}
					}
				}
			}
			
			// Also check for UUID fields that follow naming conventions (e.g., UserId, BucketId)
			if strings.Contains(field.Type, "uuid.UUID") && strings.HasSuffix(field.Name, "Id") {
				// Extract potential entity name (e.g., UserId -> User, BucketId -> Bucket)
				potentialEntityName := strings.TrimSuffix(field.Name, "Id")
				if referencedEntity, exists := allEntities[potentialEntityName]; exists && referencedEntity.Name != entity.Name {
					// Avoid duplicates
					found := false
					for _, dep := range dependencies[entity.Name] {
						if dep == potentialEntityName {
							found = true
							break
						}
					}
					if !found {
						dependencies[entity.Name] = append(dependencies[entity.Name], potentialEntityName)
					}
				}
			}
		}
	}
	
	// Perform topological sort
	result := []*models.EntityModel{}
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	
	var visit func(string) error
	visit = func(entityName string) error {
		if visiting[entityName] {
			return fmt.Errorf("circular dependency detected involving entity: %s", entityName)
		}
		if visited[entityName] {
			return nil
		}
		
		visiting[entityName] = true
		
		// Visit all dependencies first
		for _, dep := range dependencies[entityName] {
			if _, exists := allEntities[dep]; exists {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		
		visiting[entityName] = false
		visited[entityName] = true
		result = append(result, allEntities[entityName])
		
		return nil
	}
	
	// Visit all entities
	for entityName := range allEntities {
		if !visited[entityName] {
			if err := visit(entityName); err != nil {
				// If topological sort fails due to cycles, fall back to simple ordering
				fmt.Printf("Warning: %v. Using simple entity ordering.\n", err)
				result = []*models.EntityModel{}
				for _, entity := range entityModels {
					result = append(result, entity)
				}
				break
			}
		}
	}
	
	return result
}

func (mm *MigrationManager) generateOperationsFromComparison(comparison *models.SnapshotComparison) ([]models.MigrationOperation, error) {
	var operations []models.MigrationOperation
	driver := mm.context.GetDriver()
	entityModels := mm.context.GetEntityModels()

	for _, change := range comparison.Changes {
		switch change.Type {
		case models.EntityAdded:
			entitySnapshot := change.Details.(models.EntitySnapshot)
			operation := mm.createTableOperationFromSnapshot(entitySnapshot, driver, entityModels)
			operations = append(operations, operation)

		case models.FieldAdded:
			fieldSnapshot := change.Details.(models.FieldSnapshot)
			operation := models.MigrationOperation{
				Type:       models.AddColumn,
				EntityName: change.EntityName,
				Details: models.AddColumnOperation{
					TableName: change.EntityName, 
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
					TableName: change.EntityName, 
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
					TableName:  change.EntityName, // Use Pascal case
					ColumnName: fieldSnapshot.ColumnName,
				},
			}
			operations = append(operations, operation)
		}
	}

	return operations, nil
}

func (mm *MigrationManager) createTableOperationFromSnapshot(entitySnapshot models.EntitySnapshot, driver drivers.DatabaseDriver, entityModels map[string]*models.EntityModel) models.MigrationOperation {
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

// parseForeignKeyFromTags extracts foreign key information from GORM tags
func (mm *MigrationManager) parseForeignKeyFromTags(tags map[string]string, entityName string) *models.ForeignKeyReference {
	// Look for navigation properties in related entities that reference this field
	// This is a simplified approach - in practice we'd need to analyze all entities to find relationships
	
	entityModels := mm.context.GetEntityModels()
	for _, relatedEntity := range entityModels {
		for _, field := range relatedEntity.Fields {
			if len(field.Tags) > 0 {
				// Check if this field has a foreignKey tag pointing to our entity
				if foreignKeyValue, hasForeignKey := field.Tags["foreignKey"]; hasForeignKey {
					if foreignKeyValue != "" {
						// Check if this foreign key refers to a field in our current entity
						for _, ourEntity := range entityModels {
							if ourEntity.Name == entityName {
								for _, ourField := range ourEntity.Fields {
									if ourField.ColumnName == foreignKeyValue {
										return &models.ForeignKeyReference{
											ReferencedTable:  relatedEntity.TableName,
											ReferencedColumn: "Id", // Most foreign keys reference the Id field
											OnDelete:         "CASCADE",
											OnUpdate:         "CASCADE",
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	
	return nil
}

// parseForeignKeyFromFieldName checks field names for common foreign key patterns dynamically
func (mm *MigrationManager) parseForeignKeyFromFieldName(fieldName string, entityModels map[string]*models.EntityModel) *models.ForeignKeyReference {
	fieldNameLower := strings.ToLower(fieldName)
	
	// Only create foreign keys for UUID fields that match specific patterns
	// Skip primary key field and non-ID fields
	if fieldNameLower == "id" || !strings.Contains(fieldNameLower, "id") {
		return nil
	}
	
	// Build map of available entities for reference lookup
	allEntities := make(map[string]*models.EntityModel)
	for _, entity := range entityModels {
		allEntities[strings.ToLower(entity.Name)] = entity
	}
	
	// Dynamic pattern matching: <EntityName>Id -> <EntityName>.Id
	// Be more specific about what constitutes a valid foreign key field
	if strings.HasSuffix(fieldNameLower, "id") && len(fieldNameLower) > 2 {
		// Extract potential entity name (e.g., "userid" -> "user", "bucketid" -> "bucket")
		potentialEntityName := fieldNameLower[:len(fieldNameLower)-2] // Remove "id" suffix
		
		// Only create foreign key if:
		// 1. The potential entity name matches an existing entity
		// 2. The field name follows proper naming convention (entity name + Id)
		if referencedEntity, exists := allEntities[potentialEntityName]; exists {
			// Additional validation: the field name should be a reasonable match
			expectedFieldName := referencedEntity.Name + "Id"
			if strings.EqualFold(fieldName, expectedFieldName) {
				return &models.ForeignKeyReference{
					ReferencedTable:  referencedEntity.Name, 
					ReferencedColumn: "Id",
					OnDelete:         "CASCADE",
					OnUpdate:         "CASCADE",
				}
			}
		}
	}
	
	// Handle special cases for common field patterns that typically reference user-like entities
	// Try to find the most likely entity that represents users/accounts
	var userLikeEntity *models.EntityModel
	possibleUserNames := []string{"user", "account", "person", "member", "customer", "client"}
	
	for _, possibleName := range possibleUserNames {
		if entity, exists := allEntities[possibleName]; exists {
			userLikeEntity = entity
			break
		}
	}
	
	// Only apply special cases if we found a user-like entity
	if userLikeEntity != nil {
		specialCases := []string{"uploadedby", "createdby", "modifiedby", "ownerid", "assignedto"}
		
		for _, specialCase := range specialCases {
			if fieldNameLower == specialCase {
				return &models.ForeignKeyReference{
					ReferencedTable:  userLikeEntity.Name,
					ReferencedColumn: "Id",
					OnDelete:         "CASCADE", 
					OnUpdate:         "CASCADE",
				}
			}
		}
	}
	
	return nil
}