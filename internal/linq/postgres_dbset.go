package linq

import (
	"reflect"
	"strings"

	"gorm.io/gorm"
	"github.com/shepherrrd/gontext/internal/query"
)

// PostgreSQLLinqDbSet extends LinqDbSet with PostgreSQL-specific query translation
type PostgreSQLLinqDbSet[T any] struct {
	*LinqDbSet[T]
	translator *query.PostgreSQLQueryTranslator
	tableName  string
}

// NewPostgreSQLLinqDbSet creates a new PostgreSQL-aware LINQ DbSet
func NewPostgreSQLLinqDbSet[T any](db *gorm.DB, ctx interface{}) *PostgreSQLLinqDbSet[T] {
	baseDbSet := NewLinqDbSetWithContext[T](db, ctx)
	
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	
	// Get table name
	tableName := entityType.Name()
	if tabler, ok := interface{}(zero).(interface{ TableName() string }); ok {
		tableName = tabler.TableName()
	}
	
	// Create translator
	translator := query.NewPostgreSQLQueryTranslator()
	
	// Register field names
	var fieldNames []string
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if field.PkgPath == "" { // exported field
			fieldNames = append(fieldNames, field.Name)
		}
	}
	translator.RegisterEntityFields(tableName, fieldNames)
	
	return &PostgreSQLLinqDbSet[T]{
		LinqDbSet:  baseDbSet,
		translator: translator,
		tableName:  tableName,
	}
}

// Where provides a raw SQL WHERE clause with automatic field name translation
func (ds *PostgreSQLLinqDbSet[T]) Where(condition string, args ...interface{}) *PostgreSQLLinqDbSet[T] {
	// Translate the condition
	translatedCondition := ds.translator.TranslateQuery(ds.tableName, condition)
	
	// Use the underlying GORM DB directly
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(translatedCondition, args...)
	
	return ds
}

// WhereComplex handles complex WHERE queries with AND, OR, parentheses
func (ds *PostgreSQLLinqDbSet[T]) WhereComplex(condition string, args ...interface{}) *PostgreSQLLinqDbSet[T] {
	// Translate the complex condition
	translatedCondition := ds.translator.TranslateComplexQuery(ds.tableName, condition)
	
	// Use the underlying GORM DB directly
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(translatedCondition, args...)
	
	return ds
}

// OrderBy overrides to translate field names
func (ds *PostgreSQLLinqDbSet[T]) OrderBy(field string) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(field)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Order(quotedField + " ASC")
	return ds
}

// OrderByDescending overrides to translate field names
func (ds *PostgreSQLLinqDbSet[T]) OrderByDescending(field string) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(field)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Order(quotedField + " DESC")
	return ds
}

// Select overrides to translate field names
func (ds *PostgreSQLLinqDbSet[T]) Select(fields ...string) *PostgreSQLLinqDbSet[T] {
	quotedFields := make([]string, len(fields))
	for i, field := range fields {
		quotedFields[i] = ds.translator.GetQuotedFieldName(field)
	}
	ds.LinqDbSet.db = ds.LinqDbSet.db.Select(quotedFields)
	return ds
}

// GroupBy translates field names for GROUP BY
func (ds *PostgreSQLLinqDbSet[T]) GroupBy(fields ...string) *PostgreSQLLinqDbSet[T] {
	quotedFields := make([]string, len(fields))
	for i, field := range fields {
		quotedFields[i] = ds.translator.GetQuotedFieldName(field)
	}
	
	// GORM doesn't have a direct GroupBy method on LinqDbSet, so we'll use Group
	groupClause := strings.Join(quotedFields, ", ")
	ds.LinqDbSet.db = ds.LinqDbSet.db.Group(groupClause)
	
	return ds
}

// Having translates field names for HAVING clause
func (ds *PostgreSQLLinqDbSet[T]) Having(condition string, args ...interface{}) *PostgreSQLLinqDbSet[T] {
	translatedCondition := ds.translator.TranslateQuery(ds.tableName, condition)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Having(translatedCondition, args...)
	return ds
}

// WhereField provides a convenient method for simple field comparisons
func (ds *PostgreSQLLinqDbSet[T]) WhereField(fieldName string, value interface{}) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField+" = ?", value)
	return ds
}

// WhereIn provides a convenient method for IN clauses
func (ds *PostgreSQLLinqDbSet[T]) WhereIn(fieldName string, values interface{}) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField+" IN (?)", values)
	return ds
}

// WhereNotIn provides a convenient method for NOT IN clauses
func (ds *PostgreSQLLinqDbSet[T]) WhereNotIn(fieldName string, values interface{}) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField+" NOT IN (?)", values)
	return ds
}

// WhereLike provides a convenient method for LIKE queries
func (ds *PostgreSQLLinqDbSet[T]) WhereLike(fieldName, pattern string) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField+" LIKE ?", pattern)
	return ds
}

// WhereILike provides a convenient method for case-insensitive LIKE queries (PostgreSQL specific)
func (ds *PostgreSQLLinqDbSet[T]) WhereILike(fieldName, pattern string) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField+" ILIKE ?", pattern)
	return ds
}

// WhereBetween provides a convenient method for BETWEEN queries
func (ds *PostgreSQLLinqDbSet[T]) WhereBetween(fieldName string, start, end interface{}) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField+" BETWEEN ? AND ?", start, end)
	return ds
}

// WhereNull provides a convenient method for IS NULL queries
func (ds *PostgreSQLLinqDbSet[T]) WhereNull(fieldName string) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField + " IS NULL")
	return ds
}

// WhereNotNull provides a convenient method for IS NOT NULL queries
func (ds *PostgreSQLLinqDbSet[T]) WhereNotNull(fieldName string) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedField + " IS NOT NULL")
	return ds
}

// Scan allows querying into custom structs
func (ds *PostgreSQLLinqDbSet[T]) Scan(dest interface{}) error {
	return ds.LinqDbSet.db.Scan(dest).Error
}

// Delete deletes records matching the current query
func (ds *PostgreSQLLinqDbSet[T]) Delete() error {
	return ds.LinqDbSet.db.Delete(new(T)).Error
}

// Add adds an entity (EF Core style)
func (ds *PostgreSQLLinqDbSet[T]) Add(entity T) (*T, error) {
	err := ds.LinqDbSet.db.Create(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}