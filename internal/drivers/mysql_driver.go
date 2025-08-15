package drivers

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MySQLDriver struct{}

func NewMySQLDriver() *MySQLDriver {
	return &MySQLDriver{}
}

func (m *MySQLDriver) Name() string {
	return "mysql"
}

func (m *MySQLDriver) Connect(connectionString string) (*gorm.DB, error) {
	return m.ConnectWithLogger(connectionString, "silent") // Default to Silent
}

func (m *MySQLDriver) ConnectWithLogger(connectionString string, logLevel string) (*gorm.DB, error) {
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
	
	return gorm.Open(mysql.Open(connectionString), &gorm.Config{
		Logger: gormLogger,
	})
}

func (m *MySQLDriver) GetSQLDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}

func (m *MySQLDriver) SupportsTransactions() bool {
	return true
}

func (m *MySQLDriver) MapGoTypeToSQL(goType string) string {
	switch {
	case strings.Contains(goType, "uuid.UUID"):
		return "CHAR(36)"
	case strings.Contains(goType, "time.Time"):
		return "DATETIME"
	case goType == "string":
		return "TEXT"
	case goType == "int", goType == "int32":
		return "INT"
	case goType == "int64":
		return "BIGINT"
	case goType == "bool":
		return "TINYINT(1)"
	case goType == "float64":
		return "DOUBLE"
	case strings.Contains(goType, "json.RawMessage"):
		return "JSON"
	default:
		return "TEXT"
	}
}

func (m *MySQLDriver) GetSchemaInformationQuery() string {
	return `
		SELECT 
			c.COLUMN_NAME as name,
			c.DATA_TYPE as data_type,
			c.IS_NULLABLE = 'YES' as is_nullable,
			c.COLUMN_KEY = 'PRI' as is_primary,
			c.COLUMN_DEFAULT as default_value,
			c.CHARACTER_MAXIMUM_LENGTH as max_length
		FROM INFORMATION_SCHEMA.COLUMNS c
		WHERE c.TABLE_NAME = ?
			AND c.TABLE_SCHEMA = DATABASE()
		ORDER BY c.ORDINAL_POSITION`
}