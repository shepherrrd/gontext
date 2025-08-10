package query

import (
	"reflect"

	"gorm.io/gorm/schema"
)

// PostgreSQLNamingStrategy implements GORM's NamingStrategy interface for PostgreSQL Pascal case
type PostgreSQLNamingStrategy struct {
	schema.NamingStrategy
	translator *PostgreSQLQueryTranslator
}

// NewPostgreSQLNamingStrategy creates a new PostgreSQL naming strategy
func NewPostgreSQLNamingStrategy() *PostgreSQLNamingStrategy {
	return &PostgreSQLNamingStrategy{
		translator: NewPostgreSQLQueryTranslator(),
	}
}

// TableName returns the table name (Pascal case)
func (ns *PostgreSQLNamingStrategy) TableName(table string) string {
	return table // Keep Pascal case as-is
}

// ColumnName returns the column name (Pascal case)  
func (ns *PostgreSQLNamingStrategy) ColumnName(table, column string) string {
	return column // Keep Pascal case as-is
}

// JoinTableName returns the join table name
func (ns *PostgreSQLNamingStrategy) JoinTableName(joinTable string) string {
	return joinTable
}

// RelationshipFKName returns the foreign key name
func (ns *PostgreSQLNamingStrategy) RelationshipFKName(rel schema.Relationship) string {
	return rel.Name + "ID"
}

// CheckerName returns the checker name
func (ns *PostgreSQLNamingStrategy) CheckerName(table, column string) string {
	return "chk_" + table + "_" + column
}

// IndexName returns the index name
func (ns *PostgreSQLNamingStrategy) IndexName(table, column string) string {
	return "idx_" + table + "_" + column
}

// RegisterEntityFields registers field names for query translation
func (ns *PostgreSQLNamingStrategy) RegisterEntityFields(entityName string, entityType reflect.Type) {
	var fieldNames []string
	
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if field.PkgPath == "" { // exported field
			fieldNames = append(fieldNames, field.Name)
		}
	}
	
	ns.translator.RegisterEntityFields(entityName, fieldNames)
}

// TranslateQuery translates WHERE conditions for PostgreSQL
func (ns *PostgreSQLNamingStrategy) TranslateQuery(entityName, condition string) string {
	return ns.translator.TranslateQuery(entityName, condition)
}

// GetTranslator returns the query translator
func (ns *PostgreSQLNamingStrategy) GetTranslator() *PostgreSQLQueryTranslator {
	return ns.translator
}