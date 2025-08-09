package drivers

import (
	"database/sql"
	"gorm.io/gorm"
)

type DatabaseDriver interface {
	Name() string
	Connect(connectionString string) (*gorm.DB, error)
	GetSQLDB(db *gorm.DB) (*sql.DB, error)
	MapGoTypeToSQL(goType string) string
	SupportsTransactions() bool
	GetSchemaInformationQuery() string
}

type ColumnInfo struct {
	Name         string
	DataType     string
	IsNullable   bool
	IsPrimary    bool
	DefaultValue *string
	MaxLength    *int
}