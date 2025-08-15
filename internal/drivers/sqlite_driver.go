package drivers

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SQLiteDriver struct{}

func NewSQLiteDriver() *SQLiteDriver {
	return &SQLiteDriver{}
}

func (s *SQLiteDriver) Name() string {
	return "sqlite"
}

func (s *SQLiteDriver) Connect(connectionString string) (*gorm.DB, error) {
	return s.ConnectWithLogger(connectionString, "silent") // Default to Silent
}

func (s *SQLiteDriver) ConnectWithLogger(connectionString string, logLevel string) (*gorm.DB, error) {
	// Configure GORM logger based on log level
	var gormLogger logger.Interface
	switch logLevel {
	case "info": // Info level - shows SQL queries
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	case "warn": // Warn level
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	case "error": // Error level
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Error,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	default: // Silent
		gormLogger = logger.Default.LogMode(logger.Silent)
	}
	
	return gorm.Open(sqlite.Open(connectionString), &gorm.Config{
		Logger: gormLogger,
	})
}

func (s *SQLiteDriver) GetSQLDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}

func (s *SQLiteDriver) SupportsTransactions() bool {
	return true
}

func (s *SQLiteDriver) MapGoTypeToSQL(goType string) string {
	switch {
	case strings.Contains(goType, "uuid.UUID"):
		return "TEXT"
	case strings.Contains(goType, "time.Time"):
		return "DATETIME"
	case goType == "string":
		return "TEXT"
	case goType == "int", goType == "int32", goType == "int64":
		return "INTEGER"
	case goType == "bool":
		return "BOOLEAN"
	case goType == "float64":
		return "REAL"
	case strings.Contains(goType, "json.RawMessage"):
		return "TEXT"
	default:
		return "TEXT"
	}
}

func (s *SQLiteDriver) GetSchemaInformationQuery() string {
	return `
		SELECT 
			name,
			type as data_type,
			"notnull" = 0 as is_nullable,
			pk > 0 as is_primary,
			dflt_value as default_value,
			NULL as max_length
		FROM pragma_table_info(?)`
}