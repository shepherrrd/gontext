package query

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
)

// PostgreSQLPlugin is a GORM plugin that automatically translates queries for PostgreSQL Pascal case
type PostgreSQLPlugin struct {
	translator *PostgreSQLQueryTranslator
	entityMap  map[string]reflect.Type // table name -> entity type
}

// NewPostgreSQLPlugin creates a new PostgreSQL plugin
func NewPostgreSQLPlugin() *PostgreSQLPlugin {
	return &PostgreSQLPlugin{
		translator: NewPostgreSQLQueryTranslator(),
		entityMap:  make(map[string]reflect.Type),
	}
}

// Name returns the plugin name
func (p *PostgreSQLPlugin) Name() string {
	return "postgres-pascal-case"
}

// Initialize initializes the plugin
func (p *PostgreSQLPlugin) Initialize(db *gorm.DB) error {
	// Register callbacks for query translation
	db.Callback().Query().Before("gorm:query").Register("postgres:translate_where", p.translateWhereCallback)
	db.Callback().Update().Before("gorm:update").Register("postgres:translate_update_where", p.translateWhereCallback)
	db.Callback().Delete().Before("gorm:delete").Register("postgres:translate_delete_where", p.translateWhereCallback)
	return nil
}

// RegisterEntity registers an entity for query translation
func (p *PostgreSQLPlugin) RegisterEntity(entityType reflect.Type, tableName string) {
	p.entityMap[tableName] = entityType
	
	// Extract field names
	var fieldNames []string
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if field.PkgPath == "" { // exported field
			fieldNames = append(fieldNames, field.Name)
		}
	}
	
	p.translator.RegisterEntityFields(tableName, fieldNames)
}

// translateWhereCallback translates WHERE conditions in GORM callbacks
func (p *PostgreSQLPlugin) translateWhereCallback(db *gorm.DB) {
	// Get the current statement
	stmt := db.Statement
	if stmt == nil {
		return
	}
	
	// Get the table name
	tableName := stmt.Table
	if tableName == "" && stmt.Model != nil {
		// Try to get table name from model
		modelType := reflect.TypeOf(stmt.Model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		
		// Check if model implements Tabler interface
		if tabler, ok := stmt.Model.(interface{ TableName() string }); ok {
			tableName = tabler.TableName()
		} else {
			tableName = modelType.Name()
		}
	}
	
	if tableName == "" {
		return
	}
	
	// Translate WHERE conditions
	if len(stmt.Clauses) > 0 {
		for name, clause := range stmt.Clauses {
			if strings.Contains(strings.ToLower(name), "where") {
				p.translateWhereClause(tableName, clause)
			}
		}
	}
}

// translateWhereClause translates a specific WHERE clause
func (p *PostgreSQLPlugin) translateWhereClause(tableName string, clause interface{}) {
	// This is a simplified implementation
	// In a full implementation, we would need to handle GORM's internal clause structure
	// For now, we'll handle the common cases through the public API
}

// TranslateCondition is a public method to translate WHERE conditions
func (p *PostgreSQLPlugin) TranslateCondition(tableName, condition string) string {
	return p.translator.TranslateQuery(tableName, condition)
}

// GetTranslator returns the query translator
func (p *PostgreSQLPlugin) GetTranslator() *PostgreSQLQueryTranslator {
	return p.translator
}