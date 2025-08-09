package gontext

import (
	"github.com/shepherrrd/gontext/internal/linq"
)

type LinqQuery[T any] = linq.LinqQuery[T]

// LINQ creates a new LINQ query for the specified type
func LINQ[T any](ctx *DbContext) *LinqQuery[T] {
	return linq.NewLinqQuery[T](ctx.GetDB())
}

// Query creates a LINQ query for a DbSet (type-safe)
func Query[T any](ctx *DbContext) *LinqQuery[T] {
	return linq.NewLinqQuery[T](ctx.GetDB())
}