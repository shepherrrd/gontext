package gontext

import (
	"github.com/shepherrrd/gontext/internal/linq"
)

// LinqDbSet provides EF Core-style LINQ methods with type safety
type LinqDbSet[T any] = linq.LinqDbSet[T]

// NewLinqDbSet creates a new type-safe LINQ DbSet
func NewLinqDbSet[T any](ctx *DbContext) *LinqDbSet[T] {
	return linq.NewLinqDbSetWithContext[T](ctx.GetDB(), ctx)
}

// Expression represents a LINQ lambda expression
type Expression[T any] = linq.Expression[T]

// Helper functions for creating expressions

// ById creates an expression to find by ID
func ById[T any](id interface{}) func(*LinqDbSet[T]) (*T, error) {
	return func(ds *LinqDbSet[T]) (*T, error) {
		return ds.ById(id)
	}
}

// WhereField creates an expression for field comparison
func WhereField[T any](fieldName string, value interface{}) func(*LinqDbSet[T]) *LinqDbSet[T] {
	return func(ds *LinqDbSet[T]) *LinqDbSet[T] {
		return ds.WhereField(fieldName, value)
	}
}