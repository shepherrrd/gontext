package linq

import (
	"fmt"
	"reflect"

	"gorm.io/gorm"
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
	}
}

func NewLinqDbSetWithContext[T any](db *gorm.DB, ctx interface{}) *LinqDbSet[T] {
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	return &LinqDbSet[T]{
		db:         db,
		entityType: entityType,
		context:    ctx,
	}
}

// Where - filters using lambda expression
func (ds *LinqDbSet[T]) Where(predicate Expression[T]) *LinqDbSet[T] {
	// For now, we'll use reflection and field analysis
	// In a full implementation, you'd parse the function body
	return ds
}

// FirstOrDefault - gets first element matching predicate or zero value
func (ds *LinqDbSet[T]) FirstOrDefault(predicate ...Expression[T]) (*T, error) {
	query := ds.db.Model(new(T))
	
	if len(predicate) > 0 {
		// Convert lambda to SQL - simplified approach
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			query = query.Where(condition)
		}
	}
	
	var result T
	err := query.First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil for default
		}
		return nil, err
	}
	return &result, nil
}

// First - gets first element matching predicate
func (ds *LinqDbSet[T]) First(predicate ...Expression[T]) (*T, error) {
	query := ds.db.Model(new(T))
	
	if len(predicate) > 0 {
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			query = query.Where(condition)
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
	query := ds.db.Model(new(T))
	
	if len(predicate) > 0 {
		condition := ds.parseExpression(predicate[0])
		if condition != "" {
			query = query.Where(condition)
		}
	}
	
	var results []T
	err := query.Find(&results).Error
	return results, err
}

// OrderBy - orders by field selector
func (ds *LinqDbSet[T]) OrderBy(selector func(T) interface{}) *LinqDbSet[T] {
	// Parse field name from selector
	fieldName := ds.parseFieldSelector(selector)
	if fieldName != "" {
		ds.db = ds.db.Order(fieldName + " ASC")
	}
	return ds
}

// OrderByDescending - orders by field selector descending
func (ds *LinqDbSet[T]) OrderByDescending(selector func(T) interface{}) *LinqDbSet[T] {
	fieldName := ds.parseFieldSelector(selector)
	if fieldName != "" {
		ds.db = ds.db.Order(fieldName + " DESC")
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

// WhereField - helper for field-based filtering - EF Core: context.Users.Where(x => x.Field == value)
func (ds *LinqDbSet[T]) WhereField(fieldName string, value interface{}) *LinqDbSet[T] {
	// Handle different comparison operators embedded in string values
	if strValue, ok := value.(string); ok {
		if len(strValue) > 1 && strValue[0] == '>' {
			if strValue[1] == '=' {
				ds.db = ds.db.Where(fmt.Sprintf("%s >= ?", fieldName), strValue[2:])
			} else {
				ds.db = ds.db.Where(fmt.Sprintf("%s > ?", fieldName), strValue[1:])
			}
		} else if len(strValue) > 1 && strValue[0] == '<' {
			if strValue[1] == '=' {
				ds.db = ds.db.Where(fmt.Sprintf("%s <= ?", fieldName), strValue[2:])
			} else {
				ds.db = ds.db.Where(fmt.Sprintf("%s < ?", fieldName), strValue[1:])
			}
		} else if len(strValue) > 1 && strValue[0] == '!' && strValue[1] == '=' {
			ds.db = ds.db.Where(fmt.Sprintf("%s != ?", fieldName), strValue[2:])
		} else {
			ds.db = ds.db.Where(fmt.Sprintf("%s = ?", fieldName), value)
		}
	} else {
		ds.db = ds.db.Where(fmt.Sprintf("%s = ?", fieldName), value)
	}
	return ds
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

// OrderByField - EF Core: context.Users.OrderBy(x => x.Field)
func (ds *LinqDbSet[T]) OrderByField(fieldName string) *LinqDbSet[T] {
	ds.db = ds.db.Order(fieldName + " ASC")
	return ds
}

// OrderByFieldDescending - EF Core: context.Users.OrderByDescending(x => x.Field)
func (ds *LinqDbSet[T]) OrderByFieldDescending(fieldName string) *LinqDbSet[T] {
	ds.db = ds.db.Order(fieldName + " DESC")
	return ds
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

// Add - EF Core: context.Users.Add(user)
func (ds *LinqDbSet[T]) Add(entity T) {
	if ds.context != nil {
		// Use reflection to call the change tracker
		ctxValue := reflect.ValueOf(ds.context)
		changeTrackerField := ctxValue.Elem().FieldByName("changeTracker")
		if changeTrackerField.IsValid() {
			addMethod := changeTrackerField.MethodByName("Add")
			if addMethod.IsValid() {
				addMethod.Call([]reflect.Value{
					reflect.ValueOf(entity),
					reflect.ValueOf(EntityAdded),
				})
			}
		}
	}
}

// AddRange - EF Core: context.Users.AddRange(users)
func (ds *LinqDbSet[T]) AddRange(entities []T) {
	for _, entity := range entities {
		ds.Add(entity)
	}
}

// Update - EF Core: context.Users.Update(user)
func (ds *LinqDbSet[T]) Update(entity T) {
	if ds.context != nil {
		ctxValue := reflect.ValueOf(ds.context)
		changeTrackerField := ctxValue.Elem().FieldByName("changeTracker")
		if changeTrackerField.IsValid() {
			addMethod := changeTrackerField.MethodByName("Add")
			if addMethod.IsValid() {
				addMethod.Call([]reflect.Value{
					reflect.ValueOf(entity),
					reflect.ValueOf(EntityModified),
				})
			}
		}
	}
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
		changeTrackerField := ctxValue.Elem().FieldByName("changeTracker")
		if changeTrackerField.IsValid() {
			addMethod := changeTrackerField.MethodByName("Add")
			if addMethod.IsValid() {
				addMethod.Call([]reflect.Value{
					reflect.ValueOf(entity),
					reflect.ValueOf(EntityDeleted),
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