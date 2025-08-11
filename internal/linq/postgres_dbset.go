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

// Where - overloaded method that supports multiple patterns:
// 1. Where("Id = ?", value) - SQL with parameters
// 2. Where("Id", value) - field name with value  
// 3. Where(&User{Id: 1}) - struct pointer like GORM
func (ds *PostgreSQLLinqDbSet[T]) Where(args ...interface{}) *PostgreSQLLinqDbSet[T] {
	if len(args) == 0 {
		return ds
	}
	
	// Pattern 1: Struct pointer like GORM Where(&User{Id: 1})
	if len(args) == 1 {
		arg := args[0]
		// Check if it's a pointer to our entity type
		if entityPtr, ok := arg.(*T); ok {
			return ds.WhereEntity(*entityPtr)
		}
		// Check if it's the entity type directly
		if entity, ok := arg.(T); ok {
			return ds.WhereEntity(entity)
		}
		// Check if it's any pointer that we can dereference and cast
		return ds.WhereStruct(arg)
	}
	
	// Pattern 2: Where("Id", value) - field name with value
	if len(args) == 2 {
		if fieldName, ok := args[0].(string); ok {
			return ds.WhereField(fieldName, args[1])
		}
	}
	
	// Pattern 3: Where("Id = ?", value) - SQL with parameters
	if len(args) >= 2 {
		if condition, ok := args[0].(string); ok {
			translatedCondition := ds.translator.TranslateQuery(ds.tableName, condition)
			ds.LinqDbSet.db = ds.LinqDbSet.db.Where(translatedCondition, args[1:]...)
			return ds
		}
	}
	
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

// WhereEntity - static typing with entity structs like GORM: context.Users.Where(&User{Id: 1, Name: "test"})
func (ds *PostgreSQLLinqDbSet[T]) WhereEntity(entity T) *PostgreSQLLinqDbSet[T] {
	entityValue := reflect.ValueOf(entity)
	entityType := reflect.TypeOf(entity)
	
	// Handle pointer
	if entityType.Kind() == reflect.Ptr {
		if entityValue.IsNil() {
			return ds
		}
		entityValue = entityValue.Elem()
		entityType = entityType.Elem()
	}
	
	// Iterate through fields and build WHERE conditions
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		fieldValue := entityValue.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Skip zero values (unset fields)
		if fieldValue.IsZero() {
			continue
		}
		
		fieldName := field.Name
		quotedFieldName := ds.translator.GetQuotedFieldName(fieldName)
		
		// Add WHERE condition for this field
		ds.LinqDbSet.db = ds.LinqDbSet.db.Where(quotedFieldName+" = ?", fieldValue.Interface())
	}
	
	return ds
}

// WhereStruct - overloaded method that accepts entity struct
func (ds *PostgreSQLLinqDbSet[T]) WhereStruct(entity interface{}) *PostgreSQLLinqDbSet[T] {
	// Type assertion to T
	if typedEntity, ok := entity.(T); ok {
		return ds.WhereEntity(typedEntity)
	}
	
	// If it's a pointer, try to dereference and cast
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr && !entityValue.IsNil() {
		if typedEntity, ok := entityValue.Elem().Interface().(T); ok {
			return ds.WhereEntity(typedEntity)
		}
	}
	
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

// First - overloaded method that supports static typing like GORM
func (ds *PostgreSQLLinqDbSet[T]) First(args ...interface{}) (*T, error) {
	query := ds.LinqDbSet.db.Model(new(T))
	
	// If entity pattern provided, use it as WHERE condition
	if len(args) == 1 {
		if entityPtr, ok := args[0].(*T); ok {
			// Use WhereEntity logic
			entityValue := reflect.ValueOf(*entityPtr)
			entityType := reflect.TypeOf(*entityPtr)
			
			for i := 0; i < entityType.NumField(); i++ {
				field := entityType.Field(i)
				fieldValue := entityValue.Field(i)
				
				if field.PkgPath != "" || fieldValue.IsZero() {
					continue
				}
				
				fieldName := field.Name
				quotedFieldName := ds.translator.GetQuotedFieldName(fieldName)
				
				query = query.Where(quotedFieldName+" = ?", fieldValue.Interface())
			}
		}
	}
	
	var result T
	err := query.First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Save - GORM-style save that creates or updates
func (ds *PostgreSQLLinqDbSet[T]) Save(entity interface{}) error {
	return ds.LinqDbSet.db.Save(entity).Error
}

// Create - GORM-style create 
func (ds *PostgreSQLLinqDbSet[T]) Create(entity interface{}) error {
	return ds.LinqDbSet.db.Create(entity).Error
}

// Add adds an entity (EF Core style)
func (ds *PostgreSQLLinqDbSet[T]) Add(entity T) (*T, error) {
	err := ds.LinqDbSet.db.Create(&entity).Error
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Update - GORM-style update with change tracking
func (ds *PostgreSQLLinqDbSet[T]) Update(entity T) error {
	return ds.LinqDbSet.Update(entity)
}

// Or - adds OR condition with field name translation
func (ds *PostgreSQLLinqDbSet[T]) Or(condition string, args ...interface{}) *PostgreSQLLinqDbSet[T] {
	translatedCondition := ds.translator.TranslateQuery(ds.tableName, condition)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Or(translatedCondition, args...)
	return ds
}

// OrField - adds OR condition for field comparison with translation
func (ds *PostgreSQLLinqDbSet[T]) OrField(fieldName string, value interface{}) *PostgreSQLLinqDbSet[T] {
	quotedField := ds.translator.GetQuotedFieldName(fieldName)
	ds.LinqDbSet.db = ds.LinqDbSet.db.Or(quotedField+" = ?", value)
	return ds
}

// OrEntity - adds OR condition with entity struct
func (ds *PostgreSQLLinqDbSet[T]) OrEntity(entity T) *PostgreSQLLinqDbSet[T] {
	entityValue := reflect.ValueOf(entity)
	entityType := reflect.TypeOf(entity)
	
	// Handle pointer
	if entityType.Kind() == reflect.Ptr {
		if entityValue.IsNil() {
			return ds
		}
		entityValue = entityValue.Elem()
		entityType = entityType.Elem()
	}
	
	// Build OR conditions for non-zero fields
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		fieldValue := entityValue.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Skip zero values (unset fields)
		if fieldValue.IsZero() {
			continue
		}
		
		fieldName := field.Name
		quotedFieldName := ds.translator.GetQuotedFieldName(fieldName)
		
		// Add OR condition for this field
		ds.LinqDbSet.db = ds.LinqDbSet.db.Or(quotedFieldName+" = ?", fieldValue.Interface())
	}
	
	return ds
}