package gontext

import (
	"fmt"
	"reflect"

	"github.com/shepherrrd/gontext/internal/context"
	"github.com/shepherrrd/gontext/internal/drivers"
)

type DbContext = context.DbContext
type DbSet = context.DbSet

type DbContextOptions = context.DbContextOptions

func NewDbContext(connectionString string, driverType string) (*DbContext, error) {
	var driver drivers.DatabaseDriver

	switch driverType {
	case "postgres", "postgresql":
		driver = drivers.NewPostgreSQLDriver()
	case "mysql":
		driver = drivers.NewMySQLDriver()
	case "sqlite", "sqlite3":
		driver = drivers.NewSQLiteDriver()
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driverType)
	}

	options := DbContextOptions{
		ConnectionString: connectionString,
		Driver:          driver,
	}

	return context.NewDbContext(options)
}


func NewDbSet[T any](ctx *DbContext) *DbSet {
	var zero T
	return ctx.RegisterEntity(zero)
}

type Tabler interface {
	TableName() string
}

func RegisterEntity[T any](ctx *DbContext) *LinqDbSet[T] {
	var zero T
	ctx.RegisterEntity(zero) // Register with the internal context
	return NewLinqDbSet[T](ctx) // Return the LinqDbSet for EF Core-style operations
}

func GetEntityType[T any]() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}