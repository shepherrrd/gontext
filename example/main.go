package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shepherrrd/gontext"
)

// User entity - represents a user in our system
type User struct {
	ID        uuid.UUID `gontext:"primary_key;default:gen_random_uuid()" gorm:"primaryKey;default:gen_random_uuid()"`
	Username  string    `gontext:"unique;not_null" gorm:"uniqueIndex;not null"`
	Email     string    `gontext:"unique;not_null" gorm:"uniqueIndex;not null"`
	FirstName string    `gontext:"not_null" gorm:"not null"`
	LastName  string    `gontext:"not_null" gorm:"not null"`
	Age       int       `gontext:"not_null" gorm:"not null"`
	IsActive  bool      `gontext:"not_null;default:true" gorm:"not null;default:true"`
	CreatedAt time.Time `gontext:"not_null" gorm:"not null"`
	UpdatedAt time.Time `gontext:"not_null" gorm:"not null"`
}

// Post entity - represents a blog post
type Post struct {
	ID        uuid.UUID `gontext:"primary_key;default:gen_random_uuid()" gorm:"primaryKey;default:gen_random_uuid()"`
	Title     string    `gontext:"not_null" gorm:"not null"`
	Content   string    `gontext:"not_null" gorm:"not null"`
	AuthorID  uuid.UUID `gontext:"not_null" gorm:"not null"`
	Published bool      `gontext:"not_null;default:false" gorm:"not null;default:false"`
	Views     int       `gontext:"not_null;default:0" gorm:"not null;default:0"`
	CreatedAt time.Time `gontext:"not_null" gorm:"not null"`
	UpdatedAt time.Time `gontext:"not_null" gorm:"not null"`
}

// BlogContext - Our EF Core-style DbContext with typed DbSets
type BlogContext struct {
	*gontext.DbContext
	Users *gontext.LinqDbSet[User] // EF Core: DbSet<User> Users { get; set; }
	Posts *gontext.LinqDbSet[Post] // EF Core: DbSet<Post> Posts { get; set; }
}

// NewBlogContext creates a new database context (like EF Core's DbContext constructor)
func NewBlogContext(connectionString string) (*BlogContext, error) {
	// Create the underlying DbContext
	ctx, err := gontext.NewDbContext(connectionString, "postgres")
	if err != nil {
		return nil, fmt.Errorf("failed to create database context: %w", err)
	}

	// Register entities and get typed DbSets (like EF Core's DbSet registration)
	// This is equivalent to: modelBuilder.Entity<User>() in EF Core
	users := gontext.RegisterEntity[User](ctx)
	posts := gontext.RegisterEntity[Post](ctx)

	return &BlogContext{
		DbContext: ctx,
		Users:     users,
		Posts:     posts,
	}, nil
}

func main() {
	fmt.Println("🚀 GoNtext Complete Example - EF Core for Go")
	fmt.Println("===========================================")
	fmt.Println("This example demonstrates:")
	fmt.Println("• Database context setup")
	fmt.Println("• Entity registration") 
	fmt.Println("• Migrations")
	fmt.Println("• CRUD operations using LINQ")
	fmt.Println("• Change tracking")
	fmt.Println("• EF Core patterns in Go")
	fmt.Println()

	// Step 1: Create database context
	fmt.Println("📊 Step 1: Setting up Database Context")
	fmt.Println("--------------------------------------")
	
	ctx, err := NewBlogContext("postgres://postgres@localhost:5432/shbucket?sslmode=disable")
	if err != nil {
		log.Fatalf("❌ Failed to create database context: %v", err)
	}
	defer ctx.Close()
	fmt.Println("✅ Database context created successfully")

	// Step 2: Run migrations (like EF Core's Update-Database)
	fmt.Println("\n🔄 Step 2: Running Migrations")
	fmt.Println("-----------------------------")
	
	// Ensure database tables are created (like EF Core's EnsureCreated())
	err = ctx.EnsureCreated()
	if err != nil {
		log.Fatalf("❌ Failed to create database schema: %v", err)
	}
	fmt.Println("✅ Database schema created/updated successfully")

	// Step 3: CRUD Operations using LINQ
	fmt.Println("\n📝 Step 3: CRUD Operations with LINQ")
	fmt.Println("===================================")

	demonstrateCreate(ctx)
	demonstrateRead(ctx)
	demonstrateUpdate(ctx)
	demonstrateDelete(ctx)

	fmt.Println("\n🎯 Summary")
	fmt.Println("==========")
	fmt.Println("✅ GoNtext provides complete EF Core experience in Go:")
	fmt.Println("   • Type-safe DbSets with generics")
	fmt.Println("   • LINQ-style queries") 
	fmt.Println("   • Change tracking")
	fmt.Println("   • Migration system")
	fmt.Println("   • Transaction support")
	fmt.Println("   • EF Core patterns and conventions")
}

// demonstrateCreate shows EF Core-style entity creation
func demonstrateCreate(ctx *BlogContext) {
	fmt.Println("\n🔥 CREATE Operations (EF Core: context.Users.Add(user))")
	fmt.Println("-----------------------------------------------------")

	// Create users - EF Core style: context.Users.Add(user)
	users := []User{
		{
			Username:  "alice_johnson",
			Email:     "alice@example.com",
			FirstName: "Alice",
			LastName:  "Johnson", 
			Age:       28,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Username:  "bob_smith",
			Email:     "bob@example.com",
			FirstName: "Bob",
			LastName:  "Smith",
			Age:       35,
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Username:  "charlie_brown",
			Email:     "charlie@example.com",
			FirstName: "Charlie", 
			LastName:  "Brown",
			Age:       42,
			IsActive:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// EF Core pattern: Add entities to context (tracked, but not persisted yet)
	for _, user := range users {
		ctx.Users.Add(user) // Equivalent to: context.Users.Add(user) in EF Core
		fmt.Printf("✅ Added user to context: %s (%s)\n", user.Username, user.Email)
	}

	// EF Core pattern: SaveChanges persists all tracked changes in a transaction
	fmt.Println("💾 Calling ctx.SaveChanges() to persist all changes...")
	err := ctx.SaveChanges() // Equivalent to: context.SaveChanges() in EF Core
	if err != nil {
		fmt.Printf("❌ Error saving users: %v\n", err)
		return
	}
	fmt.Printf("✅ Successfully created %d users in database\n", len(users))

	// Create posts for our users
	fmt.Println("\n📝 Creating posts...")
	
	// First, get Alice's ID to create posts for her
	alice, err := ctx.Users.WhereField("username", "alice_johnson").FirstOrDefault()
	if err != nil || alice == nil {
		fmt.Println("❌ Could not find Alice to create posts")
		return
	}

	posts := []Post{
		{
			Title:     "Getting Started with GoNtext",
			Content:   "GoNtext brings Entity Framework Core patterns to Go, making database operations intuitive and type-safe.",
			AuthorID:  alice.ID,
			Published: true,
			Views:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "LINQ Queries in Go",
			Content:   "Learn how to write LINQ-style queries in Go using GoNtext's fluent API.",
			AuthorID:  alice.ID,
			Published: false,
			Views:     0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Add posts to context
	for _, post := range posts {
		ctx.Posts.Add(post)
		fmt.Printf("✅ Added post to context: %s\n", post.Title)
	}

	// Save all posts
	err = ctx.SaveChanges()
	if err != nil {
		fmt.Printf("❌ Error saving posts: %v\n", err)
		return
	}
	fmt.Printf("✅ Successfully created %d posts in database\n", len(posts))
}

// demonstrateRead shows EF Core-style LINQ queries
func demonstrateRead(ctx *BlogContext) {
	fmt.Println("\n🔍 READ Operations (EF Core LINQ Queries)")
	fmt.Println("----------------------------------------")

	// EF Core: context.Users.ToList()
	fmt.Println("Query: Get all users")
	allUsers, err := ctx.Users.ToList()
	if err != nil {
		fmt.Printf("❌ Error querying users: %v\n", err)
		return
	}
	fmt.Printf("✅ Found %d users total\n", len(allUsers))

	// EF Core: context.Users.Where(x => x.IsActive).ToList()
	fmt.Println("\nQuery: Get active users only")
	activeUsers, err := ctx.Users.WhereField("is_active", true).ToList()
	if err != nil {
		fmt.Printf("❌ Error querying active users: %v\n", err)
		return
	}
	fmt.Printf("✅ Found %d active users:\n", len(activeUsers))
	for _, user := range activeUsers {
		fmt.Printf("   • %s (%s) - Age: %d\n", user.Username, user.Email, user.Age)
	}

	// EF Core: context.Users.FirstOrDefault(x => x.Username == "alice_johnson")
	fmt.Println("\nQuery: Find specific user by username")
	alice, err := ctx.Users.WhereField("username", "alice_johnson").FirstOrDefault()
	if err != nil {
		fmt.Printf("❌ Error finding Alice: %v\n", err)
		return
	}
	if alice != nil {
		fmt.Printf("✅ Found user: %s %s (Email: %s)\n", alice.FirstName, alice.LastName, alice.Email)
	}

	// EF Core: context.Users.Count(x => x.IsActive)
	fmt.Println("\nQuery: Count active users")
	activeCount, err := ctx.Users.WhereField("is_active", true).Count()
	if err != nil {
		fmt.Printf("❌ Error counting users: %v\n", err)
		return
	}
	fmt.Printf("✅ Active users count: %d\n", activeCount)

	// EF Core: context.Users.Any(x => x.Age > 40)
	fmt.Println("\nQuery: Check if any users are over 40")
	hasOlderUsers, err := ctx.Users.WhereField("age", ">40").Any()
	if err != nil {
		fmt.Printf("❌ Error checking for older users: %v\n", err)
		return
	}
	fmt.Printf("✅ Has users over 40: %t\n", hasOlderUsers)

	// EF Core: context.Posts.Where(x => x.Published).OrderByDescending(x => x.CreatedAt).ToList()
	fmt.Println("\nQuery: Get published posts, newest first")
	publishedPosts, err := ctx.Posts.WhereField("published", true).OrderByFieldDescending("created_at").ToList()
	if err != nil {
		fmt.Printf("❌ Error querying posts: %v\n", err)
		return
	}
	fmt.Printf("✅ Found %d published posts:\n", len(publishedPosts))
	for _, post := range publishedPosts {
		fmt.Printf("   • %s (Views: %d)\n", post.Title, post.Views)
	}

	// EF Core: context.Posts.Where(x => x.Title.Contains("GoNtext")).Single()
	fmt.Println("\nQuery: Find post containing 'GoNtext' in title")
	gonTextPost, err := ctx.Posts.WhereFieldLike("title", "GoNtext").FirstOrDefault()
	if err != nil {
		fmt.Printf("❌ Error finding GoNtext post: %v\n", err)
		return
	}
	if gonTextPost != nil {
		fmt.Printf("✅ Found post: %s\n", gonTextPost.Title)
		fmt.Printf("   Content preview: %.50s...\n", gonTextPost.Content)
	}
}

// demonstrateUpdate shows EF Core-style change tracking and updates
func demonstrateUpdate(ctx *BlogContext) {
	fmt.Println("\n✏️  UPDATE Operations (EF Core Change Tracking)")
	fmt.Println("----------------------------------------------")

	// EF Core pattern: Find entity (automatically tracked)
	fmt.Println("Finding user to update...")
	user, err := ctx.Users.WhereField("username", "bob_smith").FirstOrDefault()
	if err != nil {
		fmt.Printf("❌ Error finding Bob: %v\n", err)
		return
	}
	if user == nil {
		fmt.Println("❌ Bob not found")
		return
	}

	fmt.Printf("✅ Found user: %s %s (Current age: %d)\n", user.FirstName, user.LastName, user.Age)

	// EF Core pattern: Modify tracked entity (change tracking automatically detects this)
	fmt.Println("Modifying user properties...")
	user.Age = 36 // Birthday!
	user.FirstName = "Robert" // Prefers full name now
	user.UpdatedAt = time.Now()

	fmt.Println("✅ Modified user properties:")
	fmt.Printf("   • Age: %d → %d\n", 35, user.Age)
	fmt.Printf("   • FirstName: Bob → %s\n", user.FirstName)

	// EF Core pattern: SaveChanges automatically detects changes and persists them
	// No need to call ctx.Users.Update(user) - change tracking handles it!
	fmt.Println("💾 Calling ctx.SaveChanges() to persist changes...")
	err = ctx.SaveChanges()
	if err != nil {
		fmt.Printf("❌ Error saving changes: %v\n", err)
		return
	}

	fmt.Println("✅ User updated successfully via change tracking!")

	// Verify the update
	fmt.Println("Verifying update...")
	updatedUser, err := ctx.Users.WhereField("username", "bob_smith").FirstOrDefault()
	if err != nil {
		fmt.Printf("❌ Error verifying update: %v\n", err)
		return
	}
	if updatedUser != nil {
		fmt.Printf("✅ Verified: %s %s is now %d years old\n", 
			updatedUser.FirstName, updatedUser.LastName, updatedUser.Age)
	}

	// EF Core pattern: Explicit Update for new entities
	fmt.Println("\n📝 Explicit Update example...")
	newUser := User{
		ID:        uuid.New(),
		Username:  "diana_wilson", 
		Email:     "diana@example.com",
		FirstName: "Diana",
		LastName:  "Wilson",
		Age:       31,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// EF Core: context.Users.Update(entity) - for entities not tracked
	ctx.Users.Update(newUser)
	fmt.Printf("✅ Explicitly marked user for update: %s\n", newUser.Username)

	err = ctx.SaveChanges()
	if err != nil {
		fmt.Printf("❌ Error saving explicit update: %v\n", err)
		return
	}
	fmt.Println("✅ Explicit update saved successfully!")

	// Update a post's view count
	fmt.Println("\n📊 Updating post view count...")
	post, err := ctx.Posts.WhereFieldLike("title", "GoNtext").FirstOrDefault()
	if err != nil || post == nil {
		fmt.Println("❌ Could not find post to update")
		return
	}

	originalViews := post.Views
	post.Views += 10 // Someone read the post!
	post.UpdatedAt = time.Now()

	err = ctx.SaveChanges()
	if err != nil {
		fmt.Printf("❌ Error updating post: %v\n", err)
		return
	}

	fmt.Printf("✅ Post views updated: %d → %d\n", originalViews, post.Views)
}

// demonstrateDelete shows EF Core-style entity deletion
func demonstrateDelete(ctx *BlogContext) {
	fmt.Println("\n🗑️  DELETE Operations (EF Core: context.Users.Remove(user))")
	fmt.Println("--------------------------------------------------------")

	// Find user to delete
	fmt.Println("Finding inactive user to delete...")
	inactiveUser, err := ctx.Users.WhereField("is_active", false).FirstOrDefault()
	if err != nil {
		fmt.Printf("❌ Error finding inactive user: %v\n", err)
		return
	}
	if inactiveUser == nil {
		fmt.Println("ℹ️  No inactive users found to delete")
		return
	}

	fmt.Printf("✅ Found inactive user: %s (%s)\n", inactiveUser.Username, inactiveUser.Email)

	// EF Core pattern: Remove entity from context (marks for deletion)
	fmt.Println("Marking user for deletion...")
	ctx.Users.Remove(*inactiveUser) // EF Core: context.Users.Remove(user)
	fmt.Printf("✅ User marked for deletion: %s\n", inactiveUser.Username)

	// Count users before deletion
	userCountBefore, err := ctx.Users.Count()
	if err != nil {
		fmt.Printf("❌ Error counting users before deletion: %v\n", err)
		return
	}

	// EF Core pattern: SaveChanges executes the deletion
	fmt.Println("💾 Calling ctx.SaveChanges() to execute deletion...")
	err = ctx.SaveChanges()
	if err != nil {
		fmt.Printf("❌ Error executing deletion: %v\n", err)
		return
	}

	// Verify deletion
	userCountAfter, err := ctx.Users.Count()
	if err != nil {
		fmt.Printf("❌ Error counting users after deletion: %v\n", err)
		return
	}

	fmt.Printf("✅ User deleted successfully!\n")
	fmt.Printf("   Users before: %d\n", userCountBefore)
	fmt.Printf("   Users after: %d\n", userCountAfter)

	// Try to find the deleted user (should return nil)
	fmt.Println("Verifying user is deleted...")
	deletedUser, err := ctx.Users.WhereField("username", inactiveUser.Username).FirstOrDefault()
	if err != nil {
		fmt.Printf("❌ Error verifying deletion: %v\n", err)
		return
	}
	if deletedUser == nil {
		fmt.Println("✅ Verified: User successfully deleted from database")
	} else {
		fmt.Println("❌ Warning: User still exists in database")
	}

	// Demonstrate bulk operations
	fmt.Println("\n📦 Bulk Operations...")
	
	// Create multiple test users for bulk operations
	testUsers := []User{
		{Username: "test1", Email: "test1@example.com", FirstName: "Test", LastName: "One", Age: 25, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Username: "test2", Email: "test2@example.com", FirstName: "Test", LastName: "Two", Age: 26, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{Username: "test3", Email: "test3@example.com", FirstName: "Test", LastName: "Three", Age: 27, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	// EF Core: context.Users.AddRange(users)
	ctx.Users.AddRange(testUsers)
	fmt.Printf("✅ Added %d test users with AddRange\n", len(testUsers))

	err = ctx.SaveChanges()
	if err != nil {
		fmt.Printf("❌ Error saving bulk users: %v\n", err)
		return
	}

	// Now remove them all
	testUsersToDelete, err := ctx.Users.WhereFieldStartsWith("username", "test").ToList()
	if err != nil {
		fmt.Printf("❌ Error finding test users: %v\n", err)
		return
	}

	// EF Core: context.Users.RemoveRange(users)
	ctx.Users.RemoveRange(testUsersToDelete)
	fmt.Printf("✅ Marked %d test users for deletion with RemoveRange\n", len(testUsersToDelete))

	err = ctx.SaveChanges()
	if err != nil {
		fmt.Printf("❌ Error executing bulk deletion: %v\n", err)
		return
	}

	fmt.Printf("✅ Bulk deletion completed - removed %d test users\n", len(testUsersToDelete))
}