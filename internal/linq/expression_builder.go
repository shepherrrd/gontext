package linq

import (
	"fmt"
	"reflect"
	"log"
	"gorm.io/gorm"
	"github.com/shepherrrd/gontext/internal/query"
)

// Expression represents a LINQ expression
type Expression[T any] func(T) bool

// EntityState constants to match the context package
const (
	EntityUnchanged = 0
	EntityAdded     = 1
	EntityModified  = 2
	EntityDeleted   = 3
)

// LinqDbSet provides LINQ methods that accept lambda expressions
type LinqDbSet[T any] struct {
	db         *gorm.DB
	entityType reflect.Type
	context    interface{} // Will hold the DbContext
	translator *query.PostgreSQLQueryTranslator // For automatic PostgreSQL translation
	tableName  string // Entity table name
}

func NewLinqDbSet[T any](db *gorm.DB) *LinqDbSet[T] {
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	return &LinqDbSet[T]{
		db:         db,
		entityType: entityType,
		context:    nil, // Will be set when created from DbContext
		translator: nil, // Will be set if PostgreSQL
		tableName:  entityType.Name(),
	}
}

func NewLinqDbSetWithContext[T any](db *gorm.DB, ctx interface{}) *LinqDbSet[T] {
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	// Check if this is a PostgreSQL database and set up automatic translation
	var translator *query.PostgreSQLQueryTranslator
	tableName := entityType.Name()
	
	// Get table name (check for TableName method)
	if tabler, ok := interface{}(zero).(interface{ TableName() string }); ok {
		tableName = tabler.TableName()
	}
	
	// Detect PostgreSQL by checking the driver name
	if db.Dialector.Name() == "postgres" {
		translator = query.NewPostgreSQLQueryTranslator()
		
		// Register field names
		var fieldNames []string
		for i := 0; i < entityType.NumField(); i++ {
			field := entityType.Field(i)
			if field.PkgPath == "" { // exported field
				fieldNames = append(fieldNames, field.Name)
			}
		}
		translator.RegisterEntityFields(tableName, fieldNames)
	}

	return &LinqDbSet[T]{
		db:         db,
		entityType: entityType,
		context:    ctx,
		translator: translator,
		tableName:  tableName,
	}
}

// Where - overloaded method that supports multiple patterns:
// 1. Where("Id = ?", value) - SQL with parameters
// 2. Where("Id", value) - field name with value
// 3. Where(&User{Id: 1}) - struct pointer like GORM
func (ds *LinqDbSet[T]) Where(args ...interface{}) *LinqDbSet[T] {
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
			quotedFieldName := condition
			if ds.translator != nil {
				quotedFieldName = ds.translator.TranslateQuery(ds.tableName, condition)
			}
			ds.db = ds.db.Where(quotedFieldName, args[1:]...)
			return ds
		}
	}
	
	return ds
}

// FirstOrDefault - gets first element matching predicate or zero value
// IMPORTANT: Returns (*T, error) - you MUST handle both return values in your code
// DEPRECATED OLD PATTERN: user := h.dbContext.Files.FirstOrDefault() - WRONG! Missing error handling
// CORRECT NEW PATTERN: user, err := h.dbContext.Files.FirstOrDefault(); if err != nil { ... }
func (ds *LinqDbSet[T]) FirstOrDefault(predicate ...Expression[T]) (*T, error) {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].FirstOrDefault called with %d predicates", *new(T), len(predicate))
	
	query := ds.db.Model(new(T))
	log.Printf("[GONTEXT DEBUG] Created base query for table: %s", ds.tableName)
	
	if len(predicate) > 0 {
		// Convert lambda to SQL - simplified approach
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			log.Printf("[GONTEXT DEBUG] Adding predicate condition: %s", condition)
			query = query.Where(condition)
		} else {
			log.Printf("[GONTEXT DEBUG] Warning: Could not parse predicate expression")
		}
	}
	
	// Log the SQL query - use a more direct approach
	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		var dummy T
		return tx.First(&dummy)
	})
	log.Printf("[GONTEXT DEBUG] Generated SQL: %s", sql)
	
	// Also log the table name being queried
	log.Printf("[GONTEXT DEBUG] Querying table: %s", ds.tableName)
	
	// Log database driver and session info
	log.Printf("[GONTEXT DEBUG] Database driver: %s", ds.db.Dialector.Name())
	if ds.translator != nil {
		log.Printf("[GONTEXT DEBUG] Using PostgreSQL translator for table: %s", ds.tableName)
	}
	
	// Log any existing clauses
	if len(query.Statement.Clauses) > 0 {
		log.Printf("[GONTEXT DEBUG] Query has %d clauses", len(query.Statement.Clauses))
		for name, clause := range query.Statement.Clauses {
			log.Printf("[GONTEXT DEBUG] Clause: %s = %+v", name, clause)
		}
	} else {
		log.Printf("[GONTEXT DEBUG] No WHERE clauses - will return first record from table")
	}
	
	var result T
	log.Printf("[GONTEXT DEBUG] Executing First() query...")
	err := query.First(&result).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[GONTEXT DEBUG] No record found (ErrRecordNotFound), returning nil")
			return nil, nil // Return nil for default
		}
		log.Printf("[GONTEXT DEBUG] Error occurred during query: %v", err)
		return nil, err
	}
	
	log.Printf("[GONTEXT DEBUG] Record found successfully: %+v", result)
	return &result, nil
}

// First - overloaded method that supports multiple patterns:
// 1. First() - get first element
// 2. First(&Entity{Field: value}) - find by entity pattern (like GORM)
func (ds *LinqDbSet[T]) First(args ...interface{}) (*T, error) {
	query := ds.db.Model(new(T))
	
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
				if ds.translator != nil {
					fieldName = ds.translator.GetQuotedFieldName(fieldName)
				}
				
				query = query.Where(fmt.Sprintf("%s = ?", fieldName), fieldValue.Interface())
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

// Single - gets exactly one element matching predicate
func (ds *LinqDbSet[T]) Single(predicate ...Expression[T]) (*T, error) {
	query := ds.db.Model(new(T))
	
	if len(predicate) > 0 {
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			query = query.Where(condition)
		}
	}
	
	var results []T
	err := query.Limit(2).Find(&results).Error
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(results) > 1 {
		return nil, fmt.Errorf("sequence contains more than one element")
	}
	
	return &results[0], nil
}

// Any - checks if any element matches predicate
func (ds *LinqDbSet[T]) Any(predicate ...Expression[T]) (bool, error) {
	query := ds.db.Model(new(T))
	
	if len(predicate) > 0 {
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			query = query.Where(condition)
		}
	}
	
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

// Count - counts elements matching predicate
func (ds *LinqDbSet[T]) Count(predicate ...Expression[T]) (int64, error) {
	query := ds.db.Model(new(T))
	
	if len(predicate) > 0 {
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			query = query.Where(condition)
		}
	}
	
	var count int64
	err := query.Count(&count).Error
	return count, err
}

// ToList - gets all elements matching predicate
func (ds *LinqDbSet[T]) ToList(predicate ...Expression[T]) ([]T, error) {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].ToList called with %d predicates", *new(T), len(predicate))
	
	query := ds.db.Model(new(T))
	
	if len(predicate) > 0 {
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			log.Printf("[GONTEXT DEBUG] Adding predicate condition: %s", condition)
			query = query.Where(condition)
		} else {
			log.Printf("[GONTEXT DEBUG] Warning: Could not parse predicate expression in ToList")
		}
	}
	
	// Log the SQL query
	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Find(new([]T))
	})
	log.Printf("[GONTEXT DEBUG] Generated SQL for ToList: %s", sql)
	
	// Log existing clauses
	if len(query.Statement.Clauses) > 0 {
		log.Printf("[GONTEXT DEBUG] ToList query has %d clauses", len(query.Statement.Clauses))
		for name, clause := range query.Statement.Clauses {
			log.Printf("[GONTEXT DEBUG] ToList Clause: %s = %+v", name, clause)
		}
	} else {
		log.Printf("[GONTEXT DEBUG] ToList: No WHERE clauses - will return all records from table")
	}
	
	var results []T
	log.Printf("[GONTEXT DEBUG] Executing Find() query...")
	err := query.Find(&results).Error
	
	if err != nil {
		log.Printf("[GONTEXT DEBUG] ToList error: %v", err)
		return results, err
	}
	
	log.Printf("[GONTEXT DEBUG] ToList found %d records", len(results))
	return results, err
}

// OrderBy - overloaded method that supports multiple patterns:
// 1. OrderBy(func(T) interface{}) - field selector function
// 2. OrderBy("fieldName") - field name string
func (ds *LinqDbSet[T]) OrderBy(args ...interface{}) *LinqDbSet[T] {
	if len(args) == 0 {
		return ds
	}
	
	// Pattern 1: Function selector OrderBy(func(T) interface{})
	if len(args) == 1 {
		if selector, ok := args[0].(func(T) interface{}); ok {
			fieldName := ds.parseFieldSelector(selector)
			if fieldName != "" {
				quotedFieldName := fieldName
				if ds.translator != nil {
					quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
				}
				ds.db = ds.db.Order(quotedFieldName + " ASC")
			}
			return ds
		}
		
		// Pattern 2: String field name OrderBy("fieldName")
		if fieldName, ok := args[0].(string); ok {
			log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].OrderBy called with field name: %s", *new(T), fieldName)
			
			quotedFieldName := fieldName
			if ds.translator != nil {
				quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
				log.Printf("[GONTEXT DEBUG] Field name translated: %s -> %s", fieldName, quotedFieldName)
			}
			
			orderClause := quotedFieldName + " ASC"
			log.Printf("[GONTEXT DEBUG] Adding ORDER BY: %s", orderClause)
			ds.db = ds.db.Order(orderClause)
			return ds
		}
	}
	
	return ds
}

// OrderByDescending - overloaded method that supports multiple patterns:
// 1. OrderByDescending(func(T) interface{}) - field selector function
// 2. OrderByDescending("fieldName") - field name string
func (ds *LinqDbSet[T]) OrderByDescending(args ...interface{}) *LinqDbSet[T] {
	if len(args) == 0 {
		return ds
	}
	
	// Pattern 1: Function selector OrderByDescending(func(T) interface{})
	if len(args) == 1 {
		if selector, ok := args[0].(func(T) interface{}); ok {
			fieldName := ds.parseFieldSelector(selector)
			if fieldName != "" {
				quotedFieldName := fieldName
				if ds.translator != nil {
					quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
				}
				ds.db = ds.db.Order(quotedFieldName + " DESC")
			}
			return ds
		}
		
		// Pattern 2: String field name OrderByDescending("fieldName")
		if fieldName, ok := args[0].(string); ok {
			log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].OrderByDescending called with field name: %s", *new(T), fieldName)
			
			quotedFieldName := fieldName
			if ds.translator != nil {
				quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
				log.Printf("[GONTEXT DEBUG] Field name translated: %s -> %s", fieldName, quotedFieldName)
			}
			
			orderClause := quotedFieldName + " DESC"
			log.Printf("[GONTEXT DEBUG] Adding ORDER BY: %s", orderClause)
			ds.db = ds.db.Order(orderClause)
			return ds
		}
	}
	
	return ds
}

// Take - takes specified number of elements
func (ds *LinqDbSet[T]) Take(count int) *LinqDbSet[T] {
	ds.db = ds.db.Limit(count)
	return ds
}

// Skip - skips specified number of elements
func (ds *LinqDbSet[T]) Skip(count int) *LinqDbSet[T] {
	ds.db = ds.db.Offset(count)
	return ds
}

// parseExpression attempts to parse the lambda expression
// This is a simplified version - in production, you'd want a proper expression parser
func (ds *LinqDbSet[T]) parseExpression(expr Expression[T]) string {
	// For this implementation, we'll use a simplified approach
	// In reality, you'd need to parse the function's AST or use code generation
	
	// This is a placeholder - real implementation would parse the lambda
	// For now, return empty string to indicate no parsing
	return ""
}

// parseFieldSelector attempts to parse field selector
func (ds *LinqDbSet[T]) parseFieldSelector(selector func(T) interface{}) string {
	// This would require AST parsing or code generation in a real implementation
	// For now, return empty string
	return ""
}

// Helper methods for common patterns - EF Core style

// ById - shorthand for finding by ID - EF Core: context.Users.FirstOrDefault(x => x.Id == id)
func (ds *LinqDbSet[T]) ById(id interface{}) (*T, error) {
	var result T
	err := ds.db.Where("id = ?", id).First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// WhereEntity - static typing with entity structs with comparison operator support
// Supports: context.Users.Where(&User{Id: 1, Name: "test"}) for equality
// Supports: context.Users.Where(&User{Age: ">18"}) for comparison operators
func (ds *LinqDbSet[T]) WhereEntity(entity T) *LinqDbSet[T] {
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
		quotedFieldName := fieldName
		if ds.translator != nil {
			quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
		}
		
		// Check if the value is a string with comparison operators
		value := fieldValue.Interface()
		if strValue, ok := value.(string); ok {
			// Parse operator from string value
			operator, actualValue := ds.parseOperator(strValue)
			condition := fmt.Sprintf("%s %s ?", quotedFieldName, operator)
			log.Printf("[GONTEXT DEBUG] Adding entity WHERE condition: %s with value: %s", condition, actualValue)
			ds.db = ds.db.Where(condition, actualValue)
		} else {
			// Default equality comparison
			condition := fmt.Sprintf("%s = ?", quotedFieldName)
			log.Printf("[GONTEXT DEBUG] Adding entity WHERE condition: %s with value: %+v", condition, value)
			ds.db = ds.db.Where(condition, value)
		}
	}
	
	return ds
}

// Where - overloaded method that accepts either entity struct or function
func (ds *LinqDbSet[T]) WhereStruct(entity interface{}) *LinqDbSet[T] {
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

// WhereField - helper for field-based filtering with comparison operators
// DEPRECATED: Use the overloaded Where method instead: Where("fieldName", value) or Where(&Entity{Field: value})
// Supports: WhereField("Age", 25), WhereField("Age", ">25"), WhereField("Age", ">=18"), etc.
func (ds *LinqDbSet[T]) WhereField(fieldName string, value interface{}) *LinqDbSet[T] {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].WhereField called: field=%s, value=%+v", *new(T), fieldName, value)
	
	// Apply PostgreSQL translation if available
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
		log.Printf("[GONTEXT DEBUG] Field name translated: %s -> %s", fieldName, quotedFieldName)
	}
	
	return ds.addComparisonCondition(quotedFieldName, value, "WHERE")
}

// addComparisonCondition - helper to add comparison conditions with operator support
func (ds *LinqDbSet[T]) addComparisonCondition(quotedFieldName string, value interface{}, conditionType string) *LinqDbSet[T] {
	// Handle comparison operators for numeric and string types
	switch v := value.(type) {
	case string:
		// Check for operator prefixes in string values
		operator, actualValue := ds.parseOperator(v)
		condition := fmt.Sprintf("%s %s ?", quotedFieldName, operator)
		log.Printf("[GONTEXT DEBUG] Adding %s condition: %s with value: %s", conditionType, condition, actualValue)
		
		if conditionType == "WHERE" {
			ds.db = ds.db.Where(condition, actualValue)
		} else {
			ds.db = ds.db.Or(condition, actualValue)
		}
		
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		// For numeric types, support direct comparison
		condition := fmt.Sprintf("%s = ?", quotedFieldName)
		log.Printf("[GONTEXT DEBUG] Adding %s condition: %s with value: %+v", conditionType, condition, value)
		
		if conditionType == "WHERE" {
			ds.db = ds.db.Where(condition, value)
		} else {
			ds.db = ds.db.Or(condition, value)
		}
		
	default:
		// Default equality comparison
		condition := fmt.Sprintf("%s = ?", quotedFieldName)
		log.Printf("[GONTEXT DEBUG] Adding %s condition: %s with value: %+v", conditionType, condition, value)
		
		if conditionType == "WHERE" {
			ds.db = ds.db.Where(condition, value)
		} else {
			ds.db = ds.db.Or(condition, value)
		}
	}
	
	log.Printf("[GONTEXT DEBUG] Current query has %d clauses after %s", len(ds.db.Statement.Clauses), conditionType)
	return ds
}

// parseOperator - parses operator from string value
func (ds *LinqDbSet[T]) parseOperator(strValue string) (operator string, actualValue string) {
	if len(strValue) == 0 {
		return "=", strValue
	}
	
	// Check for two-character operators first
	if len(strValue) >= 2 {
		switch strValue[:2] {
		case ">=":
			return ">=", strValue[2:]
		case "<=":
			return "<=", strValue[2:]
		case "!=", "<>":
			return "!=", strValue[2:]
		}
	}
	
	// Check for single-character operators
	switch strValue[0] {
	case '>':
		return ">", strValue[1:]
	case '<':
		return "<", strValue[1:]
	case '=':
		return "=", strValue[1:]
	default:
		return "=", strValue
	}
}

// WhereFieldIn - helper for IN queries - EF Core: context.Users.Where(x => values.Contains(x.Field))
func (ds *LinqDbSet[T]) WhereFieldIn(fieldName string, values []interface{}) *LinqDbSet[T] {
	ds.db = ds.db.Where(fmt.Sprintf("%s IN ?", fieldName), values)
	return ds
}

// WhereFieldLike - helper for LIKE queries - EF Core: context.Users.Where(x => x.Field.Contains(pattern))
func (ds *LinqDbSet[T]) WhereFieldLike(fieldName string, pattern string) *LinqDbSet[T] {
	ds.db = ds.db.Where(fmt.Sprintf("%s LIKE ?", fieldName), "%"+pattern+"%")
	return ds
}

// WhereFieldStartsWith - EF Core: context.Users.Where(x => x.Field.StartsWith(prefix))
func (ds *LinqDbSet[T]) WhereFieldStartsWith(fieldName string, prefix string) *LinqDbSet[T] {
	ds.db = ds.db.Where(fmt.Sprintf("%s LIKE ?", fieldName), prefix+"%")
	return ds
}

// WhereFieldEndsWith - EF Core: context.Users.Where(x => x.Field.EndsWith(suffix))
func (ds *LinqDbSet[T]) WhereFieldEndsWith(fieldName string, suffix string) *LinqDbSet[T] {
	ds.db = ds.db.Where(fmt.Sprintf("%s LIKE ?", fieldName), "%"+suffix)
	return ds
}

// WhereFieldBetween - EF Core: context.Users.Where(x => x.Field >= min && x.Field <= max)
func (ds *LinqDbSet[T]) WhereFieldBetween(fieldName string, min, max interface{}) *LinqDbSet[T] {
	ds.db = ds.db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", fieldName), min, max)
	return ds
}

// Or - overloaded method that supports multiple patterns like Where:
// 1. Or("email = ?", value) - SQL with parameters
// 2. Or("Email", value) - field name with value
// 3. Or(&User{Email: "test"}) - entity struct
func (ds *LinqDbSet[T]) Or(args ...interface{}) *LinqDbSet[T] {
	if len(args) == 0 {
		return ds
	}
	
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].Or called with %d args", *new(T), len(args))
	
	// Pattern 1: Entity struct like GORM Or(&User{Email: "test"})
	if len(args) == 1 {
		arg := args[0]
		// Check if it's a pointer to our entity type
		if entityPtr, ok := arg.(*T); ok {
			return ds.OrEntity(*entityPtr)
		}
		// Check if it's the entity type directly
		if entity, ok := arg.(T); ok {
			return ds.OrEntity(entity)
		}
		// Check if it's any pointer that we can dereference and cast
		return ds.OrStruct(arg)
	}
	
	// Pattern 2: Or("Email", value) - field name with value
	if len(args) == 2 {
		if fieldName, ok := args[0].(string); ok {
			return ds.OrField(fieldName, args[1])
		}
	}
	
	// Pattern 3: Or("email = ?", value) - SQL with parameters
	if len(args) >= 2 {
		if condition, ok := args[0].(string); ok {
			quotedCondition := condition
			if ds.translator != nil {
				quotedCondition = ds.translator.TranslateQuery(ds.tableName, condition)
			}
			log.Printf("[GONTEXT DEBUG] Adding OR condition: %s", quotedCondition)
			ds.db = ds.db.Or(quotedCondition, args[1:]...)
			return ds
		}
	}
	
	return ds
}

// OrStruct - helper method to handle Or with any struct type
func (ds *LinqDbSet[T]) OrStruct(entity interface{}) *LinqDbSet[T] {
	// Type assertion to T
	if typedEntity, ok := entity.(T); ok {
		return ds.OrEntity(typedEntity)
	}
	
	// If it's a pointer, try to dereference and cast
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() == reflect.Ptr && !entityValue.IsNil() {
		if typedEntity, ok := entityValue.Elem().Interface().(T); ok {
			return ds.OrEntity(typedEntity)
		}
	}
	
	return ds
}

// OrField - adds OR condition for field comparison with operator support
// DEPRECATED: Use the overloaded Or method instead: Or("fieldName", value) or Or(&Entity{Field: value})
// Supports: OrField("Age", 25), OrField("Age", ">25"), OrField("Age", ">=18"), etc.
func (ds *LinqDbSet[T]) OrField(fieldName string, value interface{}) *LinqDbSet[T] {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].OrField called: field=%s, value=%+v", *new(T), fieldName, value)
	
	// Apply PostgreSQL translation if available
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
		log.Printf("[GONTEXT DEBUG] Field name translated: %s -> %s", fieldName, quotedFieldName)
	}
	
	return ds.addComparisonCondition(quotedFieldName, value, "OR")
}

// OrEntity - adds OR condition with entity struct with comparison operator support
// Supports: Where(&User{Email: email}).Or(&User{Username: username}) for equality
// Supports: Where(&User{Age: ">18"}).Or(&User{Role: "admin"}) for comparison operators
func (ds *LinqDbSet[T]) OrEntity(entity T) *LinqDbSet[T] {
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
		quotedFieldName := fieldName
		if ds.translator != nil {
			quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
		}
		
		// Check if the value is a string with comparison operators
		value := fieldValue.Interface()
		if strValue, ok := value.(string); ok {
			// Parse operator from string value
			operator, actualValue := ds.parseOperator(strValue)
			condition := fmt.Sprintf("%s %s ?", quotedFieldName, operator)
			log.Printf("[GONTEXT DEBUG] Adding entity OR condition: %s with value: %s", condition, actualValue)
			ds.db = ds.db.Or(condition, actualValue)
		} else {
			// Default equality comparison
			condition := fmt.Sprintf("%s = ?", quotedFieldName)
			log.Printf("[GONTEXT DEBUG] Adding entity OR condition: %s with value: %+v", condition, value)
			ds.db = ds.db.Or(condition, value)
		}
	}
	
	return ds
}

// WhereFieldNull - EF Core: context.Users.Where(x => x.Field == null)
func (ds *LinqDbSet[T]) WhereFieldNull(fieldName string) *LinqDbSet[T] {
	ds.db = ds.db.Where(fmt.Sprintf("%s IS NULL", fieldName))
	return ds
}

// WhereFieldNotNull - EF Core: context.Users.Where(x => x.Field != null)
func (ds *LinqDbSet[T]) WhereFieldNotNull(fieldName string) *LinqDbSet[T] {
	ds.db = ds.db.Where(fmt.Sprintf("%s IS NOT NULL", fieldName))
	return ds
}

// OrderByField - EF Core: context.Users.OrderBy("Field")
// DEPRECATED: Use the overloaded OrderBy method instead: OrderBy("fieldName") or OrderBy(func(T) interface{})
func (ds *LinqDbSet[T]) OrderByField(fieldName string) *LinqDbSet[T] {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].OrderByField called (DEPRECATED): field=%s", *new(T), fieldName)
	
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
		log.Printf("[GONTEXT DEBUG] Field name translated: %s -> %s", fieldName, quotedFieldName)
	}
	
	orderClause := quotedFieldName + " ASC"
	log.Printf("[GONTEXT DEBUG] Adding ORDER BY: %s", orderClause)
	ds.db = ds.db.Order(orderClause)
	return ds
}

// OrderByFieldDescending - EF Core: context.Users.OrderByDescending("Field")
// DEPRECATED: Use the overloaded OrderByDescending method instead: OrderByDescending("fieldName") or OrderByDescending(func(T) interface{})
func (ds *LinqDbSet[T]) OrderByFieldDescending(fieldName string) *LinqDbSet[T] {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].OrderByFieldDescending called (DEPRECATED): field=%s", *new(T), fieldName)
	
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
		log.Printf("[GONTEXT DEBUG] Field name translated: %s -> %s", fieldName, quotedFieldName)
	}
	
	orderClause := quotedFieldName + " DESC"
	log.Printf("[GONTEXT DEBUG] Adding ORDER BY: %s", orderClause)
	ds.db = ds.db.Order(orderClause)
	return ds
}

// OrderByAscending - Entity-based ordering: context.Users.OrderByAscending(&User{CreatedAt: time.Now()})
// Only works with fields that have values set in the entity (non-zero values)
func (ds *LinqDbSet[T]) OrderByAscending(entity T) *LinqDbSet[T] {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].OrderByAscending called with entity", *new(T))
	fieldName := ds.getFirstNonZeroFieldName(entity)
	if fieldName != "" {
		return ds.OrderByField(fieldName)
	}
	log.Printf("[GONTEXT DEBUG] Warning: No non-zero fields found in entity for OrderByAscending")
	return ds
}

// OrderByDescendingEntity - Entity-based descending ordering: context.Users.OrderByDescendingEntity(&User{CreatedAt: time.Now()})
// Only works with fields that have values set in the entity (non-zero values)  
func (ds *LinqDbSet[T]) OrderByDescendingEntity(entity T) *LinqDbSet[T] {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].OrderByDescendingEntity called with entity", *new(T))
	fieldName := ds.getFirstNonZeroFieldName(entity)
	if fieldName != "" {
		return ds.OrderByFieldDescending(fieldName)
	}
	log.Printf("[GONTEXT DEBUG] Warning: No non-zero fields found in entity for OrderByDescendingEntity")
	return ds
}

// getFirstNonZeroFieldName - helper to extract field name from entity with non-zero value
func (ds *LinqDbSet[T]) getFirstNonZeroFieldName(entity T) string {
	entityValue := reflect.ValueOf(entity)
	entityType := reflect.TypeOf(entity)
	
	// Handle pointer
	if entityType.Kind() == reflect.Ptr {
		if entityValue.IsNil() {
			return ""
		}
		entityValue = entityValue.Elem()
		entityType = entityType.Elem()
	}
	
	// Find the first non-zero field
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		fieldValue := entityValue.Field(i)
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Return the first non-zero field
		if !fieldValue.IsZero() {
			log.Printf("[GONTEXT DEBUG] Found non-zero field for ordering: %s", field.Name)
			return field.Name
		}
	}
	
	return ""
}

// ThenByField - EF Core: context.Users.OrderBy(x => x.Field1).ThenBy(x => x.Field2)
func (ds *LinqDbSet[T]) ThenByField(fieldName string) *LinqDbSet[T] {
	ds.db = ds.db.Order(fieldName + " ASC")
	return ds
}

// ThenByFieldDescending - EF Core: context.Users.OrderBy(x => x.Field1).ThenByDescending(x => x.Field2)
func (ds *LinqDbSet[T]) ThenByFieldDescending(fieldName string) *LinqDbSet[T] {
	ds.db = ds.db.Order(fieldName + " DESC")
	return ds
}

// EF Core-style CRUD Operations

// Add - EF Core style: context.Users.Add(user) - Creates entity in database immediately
// Returns the created entity and error (if any)
func (ds *LinqDbSet[T]) Add(entity T) (*T, error) {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].Add called", *new(T))
	
	// Create the entity in the database
	err := ds.db.Create(&entity).Error
	if err != nil {
		log.Printf("[GONTEXT DEBUG] Add failed: %v", err)
		return nil, err
	}
	
	log.Printf("[GONTEXT DEBUG] Entity added successfully: %+v", entity)
	
	// Also track in change tracker if context is available
	if ds.context != nil {
		ctxValue := reflect.ValueOf(ds.context)
		if ctxValue.Kind() == reflect.Ptr {
			addEntityMethod := ctxValue.MethodByName("AddEntity")
			if addEntityMethod.IsValid() {
				addEntityMethod.Call([]reflect.Value{
					reflect.ValueOf(entity),
				})
			}
		}
	}
	
	return &entity, nil
}

// AddRange - EF Core: context.Users.AddRange(users)
// Returns slice of created entities and any errors encountered
func (ds *LinqDbSet[T]) AddRange(entities []T) ([]*T, error) {
	log.Printf("[GONTEXT DEBUG] LinqDbSet[%T].AddRange called with %d entities", *new(T), len(entities))
	
	var addedEntities []*T
	for _, entity := range entities {
		added, err := ds.Add(entity)
		if err != nil {
			log.Printf("[GONTEXT DEBUG] AddRange failed on entity: %v", err)
			return addedEntities, err
		}
		addedEntities = append(addedEntities, added)
	}
	
	log.Printf("[GONTEXT DEBUG] AddRange completed successfully: %d entities added", len(addedEntities))
	return addedEntities, nil
}

// Update - EF Core: context.Users.Update(user) with GORM-style support
func (ds *LinqDbSet[T]) Update(entity T) error {
	if ds.context != nil {
		// Use change tracking when available
		ctxValue := reflect.ValueOf(ds.context)
		if ctxValue.Kind() == reflect.Ptr {
			updateEntityMethod := ctxValue.MethodByName("UpdateEntity")
			if updateEntityMethod.IsValid() {
				updateEntityMethod.Call([]reflect.Value{
					reflect.ValueOf(entity),
				})
				saveChangesMethod := ctxValue.MethodByName("SaveChanges")
				if saveChangesMethod.IsValid() {
					results := saveChangesMethod.Call([]reflect.Value{})
					if len(results) > 0 && !results[0].IsNil() {
						return results[0].Interface().(error)
					}
					return nil
				}
			}
		}
	}
	return ds.db.Save(&entity).Error
}

// UpdateRange - EF Core: context.Users.UpdateRange(users)
func (ds *LinqDbSet[T]) UpdateRange(entities []T) {
	for _, entity := range entities {
		ds.Update(entity)
	}
}

// Remove - EF Core: context.Users.Remove(user)
func (ds *LinqDbSet[T]) Remove(entity T) {
	if ds.context != nil {
		ctxValue := reflect.ValueOf(ds.context)
		if ctxValue.Kind() == reflect.Ptr {
			removeEntityMethod := ctxValue.MethodByName("RemoveEntity")
			if removeEntityMethod.IsValid() {
				removeEntityMethod.Call([]reflect.Value{
					reflect.ValueOf(entity),
				})
			}
		}
	}
}

// RemoveRange - EF Core: context.Users.RemoveRange(users)
func (ds *LinqDbSet[T]) RemoveRange(entities []T) {
	for _, entity := range entities {
		ds.Remove(entity)
	}
}

// Find - EF Core: context.Users.Find(id) - returns tracked entity
func (ds *LinqDbSet[T]) Find(id interface{}) (*T, error) {
	var result T
	err := ds.db.Where("id = ?", id).First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	// Entity is now tracked for changes
	return &result, nil
}

// HasChanges returns true if there are pending changes
func (ds *LinqDbSet[T]) HasChanges() bool {
	if ds.context != nil {
		ctxValue := reflect.ValueOf(ds.context)
		changeTrackerField := ctxValue.Elem().FieldByName("changeTracker")
		if changeTrackerField.IsValid() {
			hasChangesMethod := changeTrackerField.MethodByName("HasChanges")
			if hasChangesMethod.IsValid() {
				results := hasChangesMethod.Call([]reflect.Value{})
				if len(results) > 0 {
					return results[0].Bool()
				}
			}
		}
	}
	return false
}

// GORM-style database operations with change tracking

// Save - GORM-style save that creates or updates
func (ds *LinqDbSet[T]) Save(entity interface{}) error {
	if ds.context != nil {
		// Use change tracking when available
		ctxValue := reflect.ValueOf(ds.context)
		if ctxValue.Kind() == reflect.Ptr {
			saveChangesMethod := ctxValue.MethodByName("SaveChanges")
			if saveChangesMethod.IsValid() {
				results := saveChangesMethod.Call([]reflect.Value{})
				if len(results) > 0 && !results[0].IsNil() {
					return results[0].Interface().(error)
				}
				return nil
			}
		}
	}
	return ds.db.Save(entity).Error
}

// Create - GORM-style create - DEPRECATED: Use Add instead for EF Core consistency
func (ds *LinqDbSet[T]) Create(entity interface{}) error {
	log.Printf("[GONTEXT DEBUG] Create method called (DEPRECATED - use Add instead)")
	return ds.db.Create(entity).Error
}


// Delete deletes records matching the current query filters
func (ds *LinqDbSet[T]) Delete() error {
	return ds.db.Delete(new(T)).Error
}

// Scan - Execute query and scan results into destination
// Example: var total int64; err := ctx.Files.Select("COALESCE(SUM(size), 0)").Scan(&total)
func (ds *LinqDbSet[T]) Scan(dest interface{}) error {
	return ds.db.Scan(dest).Error
}

// Sum - overloaded method that supports multiple patterns:
// 1. Sum(func(T) interface{}) - field selector function
// 2. Sum(&entities.Entity{Field: 0}) - entity with field to sum
func (ds *LinqDbSet[T]) Sum(args ...interface{}) (float64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("Sum requires at least one argument")
	}
	
	// Pattern 1: Function selector Sum(func(T) interface{})
	if len(args) == 1 {
		if selector, ok := args[0].(func(T) interface{}); ok {
			fieldName := ds.parseFieldSelector(selector)
			if fieldName == "" {
				return 0, fmt.Errorf("unable to parse field selector for Sum")
			}
			
			var result float64
			quotedFieldName := fieldName
			if ds.translator != nil {
				quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
			}
			
			err := ds.db.Model(new(T)).Select(fmt.Sprintf("COALESCE(SUM(%s), 0)", quotedFieldName)).Scan(&result).Error
			return result, err
		}
		
		// Pattern 2: Entity with field to sum Sum(&entities.File{Size: 0})
		if entityPtr, ok := args[0].(*T); ok {
			fieldName := ds.getFirstNonZeroFieldName(*entityPtr)
			if fieldName == "" {
				return 0, fmt.Errorf("no non-zero field found in entity for Sum")
			}
			return ds.SumField(fieldName)
		}
		
		// Check if it's the entity type directly
		if entity, ok := args[0].(T); ok {
			fieldName := ds.getFirstNonZeroFieldName(entity)
			if fieldName == "" {
				return 0, fmt.Errorf("no non-zero field found in entity for Sum")
			}
			return ds.SumField(fieldName)
		}
	}
	
	return 0, fmt.Errorf("unsupported argument type for Sum")
}

// SumField - Calculate sum using field name: ctx.Files.SumField("Size")
// PREFER: Use the overloaded Sum method instead: Sum(&Entity{Field: 0}) or Sum(func(T) interface{})
func (ds *LinqDbSet[T]) SumField(fieldName string) (float64, error) {
	var result float64
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
	}
	
	err := ds.db.Model(new(T)).Select(fmt.Sprintf("COALESCE(SUM(%s), 0)", quotedFieldName)).Scan(&result).Error
	return result, err
}

// Average - overloaded method that supports multiple patterns:
// 1. Average(func(T) interface{}) - field selector function
// 2. Average(&entities.Entity{Field: 0}) - entity with field to average
func (ds *LinqDbSet[T]) Average(args ...interface{}) (float64, error) {
	if len(args) == 0 {
		return 0, fmt.Errorf("Average requires at least one argument")
	}
	
	// Pattern 1: Function selector Average(func(T) interface{})
	if len(args) == 1 {
		if selector, ok := args[0].(func(T) interface{}); ok {
			fieldName := ds.parseFieldSelector(selector)
			if fieldName == "" {
				return 0, fmt.Errorf("unable to parse field selector for Average")
			}
			return ds.AverageField(fieldName)
		}
		
		// Pattern 2: Entity with field to average Average(&entities.File{Size: 0})
		if entityPtr, ok := args[0].(*T); ok {
			fieldName := ds.getFirstNonZeroFieldName(*entityPtr)
			if fieldName == "" {
				return 0, fmt.Errorf("no non-zero field found in entity for Average")
			}
			return ds.AverageField(fieldName)
		}
		
		// Check if it's the entity type directly
		if entity, ok := args[0].(T); ok {
			fieldName := ds.getFirstNonZeroFieldName(entity)
			if fieldName == "" {
				return 0, fmt.Errorf("no non-zero field found in entity for Average")
			}
			return ds.AverageField(fieldName)
		}
	}
	
	return 0, fmt.Errorf("unsupported argument type for Average")
}

// AverageField - Calculate average using field name: ctx.Files.AverageField("Size")
// PREFER: Use the overloaded Average method instead: Average(&Entity{Field: 0}) or Average(func(T) interface{})
func (ds *LinqDbSet[T]) AverageField(fieldName string) (float64, error) {
	var result float64
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
	}
	
	err := ds.db.Model(new(T)).Select(fmt.Sprintf("COALESCE(AVG(%s), 0)", quotedFieldName)).Scan(&result).Error
	return result, err
}

// Min - overloaded method that supports multiple patterns:
// 1. Min(func(T) interface{}) - field selector function
// 2. Min(&entities.Entity{Field: 0}) - entity with field to find minimum
func (ds *LinqDbSet[T]) Min(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("Min requires at least one argument")
	}
	
	// Pattern 1: Function selector Min(func(T) interface{})
	if len(args) == 1 {
		if selector, ok := args[0].(func(T) interface{}); ok {
			fieldName := ds.parseFieldSelector(selector)
			if fieldName == "" {
				return nil, fmt.Errorf("unable to parse field selector for Min")
			}
			return ds.MinField(fieldName)
		}
		
		// Pattern 2: Entity with field to find min Min(&entities.File{Size: 0})
		if entityPtr, ok := args[0].(*T); ok {
			fieldName := ds.getFirstNonZeroFieldName(*entityPtr)
			if fieldName == "" {
				return nil, fmt.Errorf("no non-zero field found in entity for Min")
			}
			return ds.MinField(fieldName)
		}
		
		// Check if it's the entity type directly
		if entity, ok := args[0].(T); ok {
			fieldName := ds.getFirstNonZeroFieldName(entity)
			if fieldName == "" {
				return nil, fmt.Errorf("no non-zero field found in entity for Min")
			}
			return ds.MinField(fieldName)
		}
	}
	
	return nil, fmt.Errorf("unsupported argument type for Min")
}

// MinField - Find minimum value using field name: ctx.Files.MinField("Size")
// PREFER: Use the overloaded Min method instead: Min(&Entity{Field: 0}) or Min(func(T) interface{})
func (ds *LinqDbSet[T]) MinField(fieldName string) (interface{}, error) {
	var result interface{}
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
	}
	
	err := ds.db.Model(new(T)).Select(fmt.Sprintf("MIN(%s)", quotedFieldName)).Scan(&result).Error
	return result, err
}

// Max - overloaded method that supports multiple patterns:
// 1. Max(func(T) interface{}) - field selector function
// 2. Max(&entities.Entity{Field: 0}) - entity with field to find maximum
func (ds *LinqDbSet[T]) Max(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("Max requires at least one argument")
	}
	
	// Pattern 1: Function selector Max(func(T) interface{})
	if len(args) == 1 {
		if selector, ok := args[0].(func(T) interface{}); ok {
			fieldName := ds.parseFieldSelector(selector)
			if fieldName == "" {
				return nil, fmt.Errorf("unable to parse field selector for Max")
			}
			return ds.MaxField(fieldName)
		}
		
		// Pattern 2: Entity with field to find max Max(&entities.File{Size: 0})
		if entityPtr, ok := args[0].(*T); ok {
			fieldName := ds.getFirstNonZeroFieldName(*entityPtr)
			if fieldName == "" {
				return nil, fmt.Errorf("no non-zero field found in entity for Max")
			}
			return ds.MaxField(fieldName)
		}
		
		// Check if it's the entity type directly
		if entity, ok := args[0].(T); ok {
			fieldName := ds.getFirstNonZeroFieldName(entity)
			if fieldName == "" {
				return nil, fmt.Errorf("no non-zero field found in entity for Max")
			}
			return ds.MaxField(fieldName)
		}
	}
	
	return nil, fmt.Errorf("unsupported argument type for Max")
}

// MaxField - Find maximum value using field name: ctx.Files.MaxField("Size")
// PREFER: Use the overloaded Max method instead: Max(&Entity{Field: 0}) or Max(func(T) interface{})
func (ds *LinqDbSet[T]) MaxField(fieldName string) (interface{}, error) {
	var result interface{}
	quotedFieldName := fieldName
	if ds.translator != nil {
		quotedFieldName = ds.translator.GetQuotedFieldName(fieldName)
	}
	
	err := ds.db.Model(new(T)).Select(fmt.Sprintf("MAX(%s)", quotedFieldName)).Scan(&result).Error
	return result, err
}

// Include - Type-safe Include with field name validation: query.Include("Buckets", "Sessions")
// Validates field names exist on the entity type and panics with clear error if not
func (ds *LinqDbSet[T]) Include(fieldNames ...string) *LinqDbSet[T] {
	// Validate all field names exist on the entity type
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	
	for _, fieldName := range fieldNames {
		if _, found := entityType.FieldByName(fieldName); !found {
			panic(fmt.Sprintf("Field '%s' not found on %s", fieldName, entityType.Name()))
		}
	}
	
	// Apply GORM preloading
	newDb := ds.db
	for _, association := range fieldNames {
		newDb = newDb.Preload(association)
	}
	
	return &LinqDbSet[T]{
		db:         newDb,
		entityType: ds.entityType,
		context:    ds.context,
		translator: ds.translator,
		tableName:  ds.tableName,
	}
}


// IncludeAll - Load all relationships automatically by detecting GORM foreign key tags
func (ds *LinqDbSet[T]) IncludeAll() *LinqDbSet[T] {
	var zero T
	value := reflect.ValueOf(zero)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	entityType := value.Type()
	
	newDb := ds.db
	
	// Find all relationship fields by looking for slices and struct references
	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		fieldType := field.Type
		
		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}
		
		// Check for slice relationships (e.g., []Bucket)
		if fieldType.Kind() == reflect.Slice {
			elemType := fieldType.Elem()
			if elemType.Kind() == reflect.Struct {
				// This is likely a relationship - use field name for preload
				newDb = newDb.Preload(field.Name)
			}
		}
		
		// Check for single struct relationships (e.g., User in Bucket.Owner)
		if fieldType.Kind() == reflect.Struct && fieldType.PkgPath() != "" {
			// This might be a belongs-to relationship
			newDb = newDb.Preload(field.Name)
		}
		
		// Check for pointer to struct relationships (e.g., *User)
		if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
			newDb = newDb.Preload(field.Name)
		}
	}
	
	return &LinqDbSet[T]{
		db:         newDb,
		entityType: ds.entityType,
		context:    ds.context,
		translator: ds.translator,
		tableName:  ds.tableName,
	}
}

// Select - Choose specific fields to load: context.Users.Select("Id", "Username", "Email")
// For aggregations, chain with Scan(): ctx.Files.Select("COALESCE(SUM(size), 0)").Scan(&total)
// For typed aggregations, use: ctx.Files.SumField("Size") or ctx.Files.Sum(func(f File) interface{} { return f.Size })
func (ds *LinqDbSet[T]) Select(fields ...string) *LinqDbSet[T] {
	newDb := ds.db.Select(fields)
	
	return &LinqDbSet[T]{
		db:         newDb,
		entityType: ds.entityType,
		context:    ds.context,
		translator: ds.translator,
		tableName:  ds.tableName,
	}
}

// Omit - Exclude specific fields from loading: context.Users.Omit("PasswordHash")
func (ds *LinqDbSet[T]) Omit(fields ...string) *LinqDbSet[T] {
	newDb := ds.db.Omit(fields...)
	
	return &LinqDbSet[T]{
		db:         newDb,
		entityType: ds.entityType,
		context:    ds.context,
		translator: ds.translator,
		tableName:  ds.tableName,
	}
}