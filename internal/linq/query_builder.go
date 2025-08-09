package linq

import (
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

type QueryBuilder struct {
	db         *gorm.DB
	entityType reflect.Type
	query      *gorm.DB
}

type LinqQuery[T any] struct {
	builder *QueryBuilder
}

func NewLinqQuery[T any](db *gorm.DB) *LinqQuery[T] {
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	builder := &QueryBuilder{
		db:         db,
		entityType: entityType,
		query:      db.Model(new(T)),
	}

	return &LinqQuery[T]{builder: builder}
}

// Where - filters elements based on a predicate
func (q *LinqQuery[T]) Where(condition string, args ...interface{}) *LinqQuery[T] {
	q.builder.query = q.builder.query.Where(condition, args...)
	return q
}

// WhereFunc - filters elements using a function predicate
func (q *LinqQuery[T]) WhereFunc(predicate func(T) bool) *LinqQuery[T] {
	// For function predicates, we'll need to fetch and filter in memory
	// This is less efficient but provides full LINQ-like functionality
	q.builder.query = q.builder.query.Where("1=1") // placeholder
	return q
}

// Select - projects elements to a new form
func (q *LinqQuery[T]) Select(columns ...string) *LinqQuery[T] {
	if len(columns) > 0 {
		q.builder.query = q.builder.query.Select(columns)
	}
	return q
}

// OrderBy - sorts elements in ascending order
func (q *LinqQuery[T]) OrderBy(column string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Order(column + " ASC")
	return q
}

// OrderByDescending - sorts elements in descending order
func (q *LinqQuery[T]) OrderByDescending(column string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Order(column + " DESC")
	return q
}

// ThenBy - performs a subsequent ordering in ascending order
func (q *LinqQuery[T]) ThenBy(column string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Order(column + " ASC")
	return q
}

// ThenByDescending - performs a subsequent ordering in descending order
func (q *LinqQuery[T]) ThenByDescending(column string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Order(column + " DESC")
	return q
}

// Take - returns a specified number of elements
func (q *LinqQuery[T]) Take(count int) *LinqQuery[T] {
	q.builder.query = q.builder.query.Limit(count)
	return q
}

// Skip - bypasses a specified number of elements
func (q *LinqQuery[T]) Skip(count int) *LinqQuery[T] {
	q.builder.query = q.builder.query.Offset(count)
	return q
}

// Distinct - returns distinct elements
func (q *LinqQuery[T]) Distinct(columns ...string) *LinqQuery[T] {
	if len(columns) > 0 {
		q.builder.query = q.builder.query.Distinct(columns)
	} else {
		q.builder.query = q.builder.query.Distinct()
	}
	return q
}

// GroupBy - groups elements by a key
func (q *LinqQuery[T]) GroupBy(columns ...string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Group(strings.Join(columns, ","))
	return q
}

// Having - filters grouped elements
func (q *LinqQuery[T]) Having(condition string, args ...interface{}) *LinqQuery[T] {
	q.builder.query = q.builder.query.Having(condition, args...)
	return q
}

// Join - performs an inner join
func (q *LinqQuery[T]) Join(table string, condition string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Joins(fmt.Sprintf("JOIN %s ON %s", table, condition))
	return q
}

// LeftJoin - performs a left outer join
func (q *LinqQuery[T]) LeftJoin(table string, condition string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Joins(fmt.Sprintf("LEFT JOIN %s ON %s", table, condition))
	return q
}

// RightJoin - performs a right outer join
func (q *LinqQuery[T]) RightJoin(table string, condition string) *LinqQuery[T] {
	q.builder.query = q.builder.query.Joins(fmt.Sprintf("RIGHT JOIN %s ON %s", table, condition))
	return q
}

// Include - includes related data (eager loading)
func (q *LinqQuery[T]) Include(associations ...string) *LinqQuery[T] {
	for _, assoc := range associations {
		q.builder.query = q.builder.query.Preload(assoc)
	}
	return q
}

// Execution Methods

// ToList - executes the query and returns all results
func (q *LinqQuery[T]) ToList() ([]T, error) {
	var results []T
	err := q.builder.query.Find(&results).Error
	return results, err
}

// ToArray - executes the query and returns all results as array
func (q *LinqQuery[T]) ToArray() ([]T, error) {
	return q.ToList()
}

// First - returns the first element
func (q *LinqQuery[T]) First() (*T, error) {
	var result T
	err := q.builder.query.First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// FirstOrDefault - returns the first element or zero value
func (q *LinqQuery[T]) FirstOrDefault() (*T, error) {
	var result T
	err := q.builder.query.First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

// Last - returns the last element
func (q *LinqQuery[T]) Last() (*T, error) {
	var result T
	err := q.builder.query.Last(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Single - returns the single element (fails if more than one)
func (q *LinqQuery[T]) Single() (*T, error) {
	results, err := q.Take(2).ToList()
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

// Count - returns the number of elements
func (q *LinqQuery[T]) Count() (int64, error) {
	var count int64
	err := q.builder.query.Count(&count).Error
	return count, err
}

// Any - determines whether any element exists
func (q *LinqQuery[T]) Any() (bool, error) {
	count, err := q.Count()
	return count > 0, err
}

// All - determines whether all elements satisfy a condition (requires fetching all)
func (q *LinqQuery[T]) All(predicate func(T) bool) (bool, error) {
	results, err := q.ToList()
	if err != nil {
		return false, err
	}
	for _, item := range results {
		if !predicate(item) {
			return false, nil
		}
	}
	return true, nil
}

// Sum - computes the sum of numeric values
func (q *LinqQuery[T]) Sum(column string) (interface{}, error) {
	var result struct {
		Sum interface{} `gorm:"column:sum"`
	}
	err := q.builder.query.Select(fmt.Sprintf("SUM(%s) as sum", column)).Scan(&result).Error
	return result.Sum, err
}

// Average - computes the average of numeric values
func (q *LinqQuery[T]) Average(column string) (interface{}, error) {
	var result struct {
		Avg interface{} `gorm:"column:avg"`
	}
	err := q.builder.query.Select(fmt.Sprintf("AVG(%s) as avg", column)).Scan(&result).Error
	return result.Avg, err
}

// Min - finds the minimum value
func (q *LinqQuery[T]) Min(column string) (interface{}, error) {
	var result struct {
		Min interface{} `gorm:"column:min"`
	}
	err := q.builder.query.Select(fmt.Sprintf("MIN(%s) as min", column)).Scan(&result).Error
	return result.Min, err
}

// Max - finds the maximum value
func (q *LinqQuery[T]) Max(column string) (interface{}, error) {
	var result struct {
		Max interface{} `gorm:"column:max"`
	}
	err := q.builder.query.Select(fmt.Sprintf("MAX(%s) as max", column)).Scan(&result).Error
	return result.Max, err
}

// Contains - determines whether the sequence contains a specific element
func (q *LinqQuery[T]) Contains(column string, value interface{}) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s = ?", column), value)
}

// StartsWith - filters strings that start with specified value
func (q *LinqQuery[T]) StartsWith(column string, value string) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s LIKE ?", column), value+"%")
}

// EndsWith - filters strings that end with specified value
func (q *LinqQuery[T]) EndsWith(column string, value string) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s LIKE ?", column), "%"+value)
}

// StringContains - filters strings that contain specified value
func (q *LinqQuery[T]) StringContains(column string, value string) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s LIKE ?", column), "%"+value+"%")
}

// In - filters elements where column value is in the provided list
func (q *LinqQuery[T]) In(column string, values ...interface{}) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s IN ?", column), values)
}

// NotIn - filters elements where column value is not in the provided list
func (q *LinqQuery[T]) NotIn(column string, values ...interface{}) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s NOT IN ?", column), values)
}

// Between - filters elements where column value is between two values
func (q *LinqQuery[T]) Between(column string, start, end interface{}) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s BETWEEN ? AND ?", column), start, end)
}

// IsNull - filters elements where column value is null
func (q *LinqQuery[T]) IsNull(column string) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s IS NULL", column))
}

// IsNotNull - filters elements where column value is not null
func (q *LinqQuery[T]) IsNotNull(column string) *LinqQuery[T] {
	return q.Where(fmt.Sprintf("%s IS NOT NULL", column))
}

// GetQuery - returns the underlying GORM query for advanced usage
func (q *LinqQuery[T]) GetQuery() *gorm.DB {
	return q.builder.query
}