package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shepherrrd/gontext"
)

// Initial entities
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username  string    `gorm:"uniqueIndex;not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	FirstName string    `gorm:"not null"`
	LastName  string    `gorm:"not null"`
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

type Post struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string    `gorm:"not null"`
	Content     string
	AuthorID    uuid.UUID `gorm:"type:uuid;not null"`
	IsPublished bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// BlogContext
type BlogContext struct {
	*gontext.DbContext
	Users *gontext.LinqDbSet[User]
	Posts *gontext.LinqDbSet[Post]
}

func NewBlogContext(connectionString string) (*BlogContext, error) {
	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		return nil, err
	}

	users := gontext.RegisterEntity[User](ctx)
	posts := gontext.RegisterEntity[Post](ctx)

	return &BlogContext{
		DbContext: ctx,
		Users:     users,
		Posts:     posts,
	}, nil
}

// CRITICAL: Design-time context for migrations
func CreateDesignTimeContext() (*gontext.DbContext, error) {
	connectionString := os.Getenv("DATABASE_URL")
	if connectionString == "" {
		connectionString = "postgres://postgres@localhost:5432/test_migrations?sslmode=disable"
	}

	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		return nil, err
	}

	// Register ALL entities for migration generation
	gontext.RegisterEntity[User](ctx)
	gontext.RegisterEntity[Post](ctx)

	return ctx, nil
}

func main() {
	if len(os.Args) > 1 {
		handleMigrationCommands()
		return
	}

	fmt.Println("🚀 GoNtext Migrations Example")
	fmt.Println("=============================")
	fmt.Println()
	fmt.Println("Migration Commands:")
	fmt.Println("  go run . migrate:add <name>     Create a new migration")
	fmt.Println("  go run . migrate:update         Apply pending migrations")
	fmt.Println("  go run . migrate:list           List migration files")
	fmt.Println("  go run . migrate:status         Show migration status")
	fmt.Println()
	fmt.Println("Example workflow:")
	fmt.Println("  go run . migrate:add InitialCreate")
	fmt.Println("  go run . migrate:update")
	fmt.Println("  go run . migrate:status")
}

func handleMigrationCommands() {
	if len(os.Args) < 2 {
		fmt.Println("Please specify a migration command")
		return
	}

	command := os.Args[1]

	switch command {
	case "migrate:add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run . migrate:add <MigrationName>")
			os.Exit(1)
		}
		if err := AddMigration(os.Args[2]); err != nil {
			log.Fatal("Migration failed:", err)
		}

	case "migrate:update":
		if err := UpdateDatabase(); err != nil {
			log.Fatal("Database update failed:", err)
		}

	case "migrate:list":
		if err := ListMigrations(); err != nil {
			log.Fatal("List failed:", err)
		}

	case "migrate:status":
		if err := ShowMigrationStatus(); err != nil {
			log.Fatal("Status failed:", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: migrate:add, migrate:update, migrate:list, migrate:status")
		os.Exit(1)
	}
}