package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shepherrrd/gontext"
	"github.com/shepherrrd/gontext/internal/migrations"
)

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "migration":
		handleMigrationCommands()
	case "database":
		handleDatabaseCommands()
	case "help", "--help", "-h":
		showUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		showUsage()
		os.Exit(1)
	}
}

func handleMigrationCommands() {
	if len(os.Args) < 3 {
		fmt.Println("Migration command requires a subcommand")
		showMigrationUsage()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "add":
		if len(os.Args) < 4 {
			fmt.Println("Migration add requires a name")
			fmt.Println("Usage: go run github.com/shepherrrd/gontext/cmd/gontext migration add <MigrationName>")
			os.Exit(1)
		}
		migrationName := os.Args[3]
		addMigration(migrationName)
	case "list":
		listMigrations()
	case "remove":
		removeLastMigration()
	default:
		fmt.Printf("Unknown migration subcommand: %s\n\n", subcommand)
		showMigrationUsage()
		os.Exit(1)
	}
}

func handleDatabaseCommands() {
	if len(os.Args) < 3 {
		fmt.Println("Database command requires a subcommand")
		showDatabaseUsage()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "update":
		updateDatabase()
	case "drop":
		dropDatabase()
	case "rollback":
		steps := 1
		if len(os.Args) >= 4 {
			fmt.Sscanf(os.Args[3], "%d", &steps)
		}
		rollbackDatabase(steps)
	default:
		fmt.Printf("Unknown database subcommand: %s\n\n", subcommand)
		showDatabaseUsage()
		os.Exit(1)
	}
}

func addMigration(name string) {
	fmt.Printf("🔄 Adding migration: %s\n", name)

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	// Look for go.mod to find project root
	projectRoot, err := findProjectRoot(wd)
	if err != nil {
		fmt.Printf("❌ Error finding project root: %v\n", err)
		os.Exit(1)
	}

	// Create migrations directory if it doesn't exist
	migrationsDir := filepath.Join(projectRoot, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		fmt.Printf("❌ Error creating migrations directory: %v\n", err)
		os.Exit(1)
	}

	// Find database connection from environment or config
	connectionString := getDatabaseConnection()
	if connectionString == "" {
		fmt.Println("❌ Database connection not found. Please set DATABASE_URL environment variable or ensure .env file exists")
		os.Exit(1)
	}

	// Create context and migration manager
	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		fmt.Printf("❌ Error creating database context: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	migrationManager := migrations.NewMigrationManager(ctx, migrationsDir, "migrations")

	// Add the migration
	if err := migrationManager.AddMigration(name); err != nil {
		fmt.Printf("❌ Error adding migration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Migration '%s' added successfully!\n", name)
	fmt.Println("📁 Files created:")
	fmt.Println("   • ModelSnapshot.json - Database schema snapshot")
	fmt.Printf("   • %s_<name>.go - Migration file with Up/Down methods\n", getCurrentTimestamp())
}

func updateDatabase() {
	fmt.Println("🔄 Updating database...")

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	projectRoot, err := findProjectRoot(wd)
	if err != nil {
		fmt.Printf("❌ Error finding project root: %v\n", err)
		os.Exit(1)
	}

	migrationsDir := filepath.Join(projectRoot, "migrations")
	connectionString := getDatabaseConnection()

	if connectionString == "" {
		fmt.Println("❌ Database connection not found")
		os.Exit(1)
	}

	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		fmt.Printf("❌ Error creating database context: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	migrationManager := migrations.NewMigrationManager(ctx, migrationsDir, "migrations")

	if err := migrationManager.UpdateDatabase(); err != nil {
		fmt.Printf("❌ Error updating database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Database updated successfully!")
}

func listMigrations() {
	fmt.Println("📋 Listing migrations...")

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	projectRoot, err := findProjectRoot(wd)
	if err != nil {
		fmt.Printf("❌ Error finding project root: %v\n", err)
		os.Exit(1)
	}

	migrationsDir := filepath.Join(projectRoot, "migrations")
	connectionString := getDatabaseConnection()

	if connectionString == "" {
		fmt.Println("❌ Database connection not found")
		os.Exit(1)
	}

	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		fmt.Printf("❌ Error creating database context: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	migrationManager := migrations.NewMigrationManager(ctx, migrationsDir, "migrations")

	if err := migrationManager.ListMigrations(); err != nil {
		fmt.Printf("❌ Error listing migrations: %v\n", err)
		os.Exit(1)
	}
}

func removeLastMigration() {
	fmt.Println("🗑️  Removing last migration...")

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	projectRoot, err := findProjectRoot(wd)
	if err != nil {
		fmt.Printf("❌ Error finding project root: %v\n", err)
		os.Exit(1)
	}

	migrationsDir := filepath.Join(projectRoot, "migrations")
	connectionString := getDatabaseConnection()

	if connectionString == "" {
		fmt.Println("❌ Database connection not found")
		os.Exit(1)
	}

	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		fmt.Printf("❌ Error creating database context: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	migrationManager := migrations.NewMigrationManager(ctx, migrationsDir, "migrations")

	if err := migrationManager.RemoveLastMigration(); err != nil {
		fmt.Printf("❌ Error removing migration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Last migration removed successfully!")
}

func dropDatabase() {
	fmt.Println("🗑️  Dropping database...")

	connectionString := getDatabaseConnection()
	if connectionString == "" {
		fmt.Println("❌ Database connection not found")
		os.Exit(1)
	}

	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		fmt.Printf("❌ Error creating database context: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	projectRoot, err := findProjectRoot(wd)
	if err != nil {
		fmt.Printf("❌ Error finding project root: %v\n", err)
		os.Exit(1)
	}

	migrationsDir := filepath.Join(projectRoot, "migrations")
	migrationManager := migrations.NewMigrationManager(ctx, migrationsDir, "migrations")

	if err := migrationManager.DropDatabase(); err != nil {
		fmt.Printf("❌ Error dropping database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Database dropped successfully!")
}

func rollbackDatabase(steps int) {
	fmt.Printf("↩️  Rolling back %d migration(s)...\n", steps)

	connectionString := getDatabaseConnection()
	if connectionString == "" {
		fmt.Println("❌ Database connection not found")
		os.Exit(1)
	}

	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		fmt.Printf("❌ Error creating database context: %v\n", err)
		os.Exit(1)
	}
	defer ctx.Close()

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	projectRoot, err := findProjectRoot(wd)
	if err != nil {
		fmt.Printf("❌ Error finding project root: %v\n", err)
		os.Exit(1)
	}

	migrationsDir := filepath.Join(projectRoot, "migrations")
	migrationManager := migrations.NewMigrationManager(ctx, migrationsDir, "migrations")

	if err := migrationManager.RollbackDatabase(steps); err != nil {
		fmt.Printf("❌ Error rolling back database: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Rolled back %d migration(s) successfully!\n", steps)
}

func findProjectRoot(startPath string) (string, error) {
	currentPath := startPath
	for {
		goModPath := filepath.Join(currentPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentPath, nil
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			return "", fmt.Errorf("could not find go.mod file")
		}
		currentPath = parent
	}
}

func getDatabaseConnection() string {
	// Check environment variable first
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return dbURL
	}

	// Try to read from .env file
	if envContent, err := os.ReadFile(".env"); err == nil {
		lines := strings.Split(string(envContent), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "DATABASE_URL=") {
				return strings.TrimPrefix(line, "DATABASE_URL=")
			}
		}
	}

	return ""
}

func getCurrentTimestamp() string {
	return "YYYYMMDDHHMMSS"
}

func showUsage() {
	fmt.Println("🚀 GoNtext CLI - Entity Framework Core for Go")
	fmt.Println("===========================================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run github.com/shepherrrd/gontext/cmd/gontext <command> [arguments]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println()
	showMigrationUsage()
	fmt.Println()
	showDatabaseUsage()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run github.com/shepherrrd/gontext/cmd/gontext migration add InitialCreate")
	fmt.Println("  go run github.com/shepherrrd/gontext/cmd/gontext database update")
	fmt.Println("  go run github.com/shepherrrd/gontext/cmd/gontext migration list")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  DATABASE_URL - Database connection string (required)")
	fmt.Println("                 Example: postgres://user:pass@localhost/db?sslmode=disable")
	fmt.Println()
}

func showMigrationUsage() {
	fmt.Println("Migration Commands:")
	fmt.Println("  migration add <name>    Create a new migration")
	fmt.Println("  migration list          List all migrations")
	fmt.Println("  migration remove        Remove the last migration")
}

func showDatabaseUsage() {
	fmt.Println("Database Commands:")
	fmt.Println("  database update         Apply pending migrations")
	fmt.Println("  database drop           Drop all tables")
	fmt.Println("  database rollback [n]   Rollback n migrations (default: 1)")
}