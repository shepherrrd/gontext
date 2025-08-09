package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shepherrrd/gontext"
)

// User entity with proper GORM tags
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username  string    `gorm:"uniqueIndex;not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	FirstName string    `gorm:"not null"`
	LastName  string    `gorm:"not null"`
	Age       int
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time
}

// BlogContext - EF Core style DbContext
type BlogContext struct {
	*gontext.DbContext
	Users *gontext.LinqDbSet[User]
}

func NewBlogContext(connectionString string) (*BlogContext, error) {
	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		return nil, err
	}

	users := gontext.RegisterEntity[User](ctx)

	return &BlogContext{
		DbContext: ctx,
		Users:     users,
	}, nil
}

func main() {
	fmt.Println("🚀 GoNtext CRUD Example")
	fmt.Println("=======================")

	// Create database context
	ctx, err := NewBlogContext("postgres://postgres@localhost:5432/test_gontext?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to create context:", err)
	}
	defer ctx.Close()

	// Ensure database tables exist (like EF Core's EnsureCreated)
	fmt.Println("📋 Creating tables...")
	if err := ctx.EnsureCreated(); err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	// Demonstrate CRUD operations
	demonstrateCreate(ctx)
	demonstrateRead(ctx)
	demonstrateUpdate(ctx)
	demonstrateDelete(ctx)

	fmt.Println("\n✅ CRUD demo completed!")
}

func demonstrateCreate(ctx *BlogContext) {
	fmt.Println("\n🔨 CREATE Operations")
	fmt.Println("-------------------")

	// Create new users (like EF Core: context.Users.Add())
	users := []User{
		{ID: uuid.New(), Username: "alice", Email: "alice@example.com", FirstName: "Alice", LastName: "Smith", Age: 25},
		{ID: uuid.New(), Username: "bob", Email: "bob@example.com", FirstName: "Bob", LastName: "Jones", Age: 30},
		{ID: uuid.New(), Username: "charlie", Email: "charlie@example.com", FirstName: "Charlie", LastName: "Brown", Age: 35},
	}

	for _, user := range users {
		ctx.Users.Add(user)
		fmt.Printf("➕ Added user: %s\n", user.Username)
	}

	// Save all changes (like EF Core: context.SaveChanges())
	if err := ctx.SaveChanges(); err != nil {
		log.Fatal("Failed to save users:", err)
	}
	fmt.Println("💾 All users saved to database")
}

func demonstrateRead(ctx *BlogContext) {
	fmt.Println("\n📖 READ Operations")
	fmt.Println("-----------------")

	// Count users (like EF Core: context.Users.Count())
	count, _ := ctx.Users.Count()
	fmt.Printf("👥 Total users: %d\n", count)

	// Get all users (like EF Core: context.Users.ToList())
	allUsers, _ := ctx.Users.ToList()
	fmt.Printf("📋 Retrieved %d users\n", len(allUsers))

	// Find specific user (like EF Core: context.Users.FirstOrDefault())
	alice, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault()
	if alice != nil {
		fmt.Printf("🔎 Found Alice: %s %s (Age: %d)\n", alice.FirstName, alice.LastName, alice.Age)
	}

	// Find active users (like EF Core: context.Users.Where(x => x.IsActive))
	activeUsers, _ := ctx.Users.WhereField("is_active", true).ToList()
	fmt.Printf("✅ Active users: %d\n", len(activeUsers))
}

func demonstrateUpdate(ctx *BlogContext) {
	fmt.Println("\n✏️ UPDATE Operations")
	fmt.Println("-------------------")

	// Find user to update
	bob, _ := ctx.Users.WhereField("username", "bob").FirstOrDefault()
	if bob != nil {
		fmt.Printf("Before update: Bob is %d years old\n", bob.Age)

		// Update with change tracking (EF Core style)
		bob.Age = 31
		bob.FirstName = "Bobby"
		bob.UpdatedAt = time.Now()

		ctx.SaveChanges() // Automatically detects changes
		fmt.Println("💾 Bob updated with change tracking")

		// Verify the update
		updatedBob, _ := ctx.Users.WhereField("username", "bob").FirstOrDefault()
		fmt.Printf("After update: %s is %d years old\n", updatedBob.FirstName, updatedBob.Age)
	}
}

func demonstrateDelete(ctx *BlogContext) {
	fmt.Println("\n🗑️ DELETE Operations")
	fmt.Println("-------------------")

	// Find user to delete
	charlie, _ := ctx.Users.WhereField("username", "charlie").FirstOrDefault()
	if charlie != nil {
		fmt.Printf("Deleting user: %s\n", charlie.Username)

		// Mark for deletion (like EF Core: context.Users.Remove())
		ctx.Users.Remove(*charlie)

		// Save changes to execute deletion
		ctx.SaveChanges()
		fmt.Println("💾 Charlie deleted from database")

		// Verify deletion
		finalCount, _ := ctx.Users.Count()
		fmt.Printf("👥 Users remaining: %d\n", finalCount)
	}
}