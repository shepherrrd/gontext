package drivers

import (
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/shepherrrd/gontext/internal/query"
)

type PostgreSQLDriver struct{
	plugin *query.PostgreSQLPlugin
}

func NewPostgreSQLDriver() *PostgreSQLDriver {
	return &PostgreSQLDriver{
		plugin: query.NewPostgreSQLPlugin(),
	}
}

func (p *PostgreSQLDriver) Name() string {
	return "postgres"
}

func (p *PostgreSQLDriver) Connect(connectionString string) (*gorm.DB, error) {
	return p.ConnectWithLogger(connectionString, "silent") // Default to Silent
}

func (p *PostgreSQLDriver) ConnectWithLogger(connectionString string, logLevel string) (*gorm.DB, error) {
	// Create PostgreSQL naming strategy for Pascal case
	namingStrategy := query.NewPostgreSQLNamingStrategy()
	
	// Configure GORM logger based on log level
	var gormLogger logger.Interface
	switch logLevel {
	case "info": // Info level - shows SQL queries
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			logger.Config{
				SlowThreshold:             time.Second,   // Slow SQL threshold
				LogLevel:                  logger.Info,   // Log level: Info shows all SQL
				IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
				Colorful:                  true,          // Enable color
			},
		)
	case "warn": // Warn level - shows slow queries and errors
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	case "error": // Error level - shows only errors
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
	
	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{
		NamingStrategy: namingStrategy,
		Logger:         gormLogger,
	})
	
	if err != nil {
		return nil, err
	}
	
	// Install the PostgreSQL plugin
	err = db.Use(p.plugin)
	if err != nil {
		return nil, err
	}
	
	return db, nil
}

// GetPlugin returns the PostgreSQL plugin for query translation
func (p *PostgreSQLDriver) GetPlugin() *query.PostgreSQLPlugin {
	return p.plugin
}

func (p *PostgreSQLDriver) GetSQLDB(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}

func (p *PostgreSQLDriver) SupportsTransactions() bool {
	return true
}

func (p *PostgreSQLDriver) MapGoTypeToSQL(goType string) string {
	switch {
	case strings.Contains(goType, "uuid.UUID"):
		return "UUID"
	case strings.Contains(goType, "time.Time"):
		return "TIMESTAMP"
	case goType == "string":
		return "TEXT"
	case goType == "int", goType == "int32":
		return "INTEGER"
	case goType == "int64":
		return "BIGINT"
	case goType == "bool":
		return "BOOLEAN"
	case goType == "float64":
		return "DOUBLE PRECISION"
	case strings.Contains(goType, "[]string"):
		return "TEXT[]"
	case strings.Contains(goType, "json.RawMessage"):
		return "JSONB"
	default:
		return "TEXT"
	}
}

func (p *PostgreSQLDriver) GetSchemaInformationQuery() string {
	return `
		SELECT 
			c.column_name as name,
			c.data_type,
			c.is_nullable = 'YES' as is_nullable,
			CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_primary,
			c.column_default as default_value,
			c.character_maximum_length as max_length
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT kcu.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu 
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
				AND tc.table_name = $1
				AND tc.table_schema = 'public'
		) pk ON c.column_name = pk.column_name
		WHERE c.table_name = $1 
			AND c.table_schema = 'public'
		ORDER BY c.ordinal_position`
}