package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shepherrrd/gontext"
)

// Entities for LINQ demonstrations
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Username  string    `gorm:"uniqueIndex;not null"`
	Email     string    `gorm:"uniqueIndex;not null"`
	FirstName string    `gorm:"not null"`
	LastName  string    `gorm:"not null"`
	Age       int
	Salary    float64
	IsActive  bool      `gorm:"not null;default:true"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time
}

type Post struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Title       string    `gorm:"not null"`
	Content     string
	AuthorID    uuid.UUID `gorm:"type:uuid;not null"`
	ViewCount   int       `gorm:"not null;default:0"`
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

func main() {
	fmt.Println("🚀 GoNtext LINQ Queries Example")
	fmt.Println("===============================")

	// Create database context
	ctx, err := NewBlogContext("postgres://postgres@localhost:5432/test_linq?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to create context:", err)
	}
	defer ctx.Close()

	// Setup database and sample data
	setupDatabase(ctx)
	
	// Demonstrate all LINQ capabilities
	demonstrateBasicQueries(ctx)
	demonstrateFiltering(ctx)
	demonstrateStringOperations(ctx)
	demonstrateOrdering(ctx)
	demonstratePagination(ctx)
	demonstrateAggregations(ctx)
	demonstrateComplexQueries(ctx)
	demonstrateExistenceChecks(ctx)

	fmt.Println("\n✅ LINQ demo completed!")
}

func setupDatabase(ctx *BlogContext) {
	fmt.Println("📋 Setting up database and sample data...")
	
	// Ensure tables exist
	if err := ctx.EnsureCreated(); err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	// Create sample users
	users := []User{
		{ID: uuid.New(), Username: "alice", Email: "alice@example.com", FirstName: "Alice", LastName: "Smith", Age: 25, Salary: 50000, IsActive: true},
		{ID: uuid.New(), Username: "bob", Email: "bob@gmail.com", FirstName: "Bob", LastName: "Johnson", Age: 30, Salary: 60000, IsActive: true},
		{ID: uuid.New(), Username: "charlie", Email: "charlie@yahoo.com", FirstName: "Charlie", LastName: "Brown", Age: 35, Salary: 70000, IsActive: false},
		{ID: uuid.New(), Username: "diana", Email: "diana@gmail.com", FirstName: "Diana", LastName: "Prince", Age: 28, Salary: 55000, IsActive: true},
		{ID: uuid.New(), Username: "eve", Email: "eve@example.com", FirstName: "Eve", LastName: "Adams", Age: 22, Salary: 45000, IsActive: true},
	}

	for _, user := range users {
		ctx.Users.Add(user)
	}

	if err := ctx.SaveChanges(); err != nil {
		log.Fatal("Failed to save users:", err)
	}

	// Get user IDs for posts
	allUsers, _ := ctx.Users.ToList()
	if len(allUsers) < 2 {
		log.Fatal("Not enough users created")
	}

	// Create sample posts
	posts := []Post{
		{ID: uuid.New(), Title: "Getting Started with Go", Content: "Go is amazing!", AuthorID: allUsers[0].ID, ViewCount: 150, IsPublished: true},
		{ID: uuid.New(), Title: "Advanced Go Patterns", Content: "Learn advanced techniques", AuthorID: allUsers[0].ID, ViewCount: 89, IsPublished: true},
		{ID: uuid.New(), Title: "Database Design", Content: "How to design databases", AuthorID: allUsers[1].ID, ViewCount: 234, IsPublished: true},
		{ID: uuid.New(), Title: "Draft Post", Content: "This is unpublished", AuthorID: allUsers[1].ID, ViewCount: 0, IsPublished: false},
		{ID: uuid.New(), Title: "Popular Tutorial", Content: "Very popular content", AuthorID: allUsers[0].ID, ViewCount: 567, IsPublished: true},
	}

	for _, post := range posts {
		ctx.Posts.Add(post)
	}

	if err := ctx.SaveChanges(); err != nil {
		log.Fatal("Failed to save posts:", err)
	}

	fmt.Printf("✅ Created %d users and %d posts\n", len(users), len(posts))
}

func demonstrateBasicQueries(ctx *BlogContext) {
	fmt.Println("\n📖 Basic Queries")
	fmt.Println("----------------")

	// Get all users (like EF Core: context.Users.ToList())
	allUsers, _ := ctx.Users.ToList()
	fmt.Printf("All users: %d\n", len(allUsers))

	// Count users (like EF Core: context.Users.Count())
	userCount, _ := ctx.Users.Count()
	fmt.Printf("User count: %d\n", userCount)

	// First user (like EF Core: context.Users.First())
	firstUser, _ := ctx.Users.OrderByField("username").FirstOrDefault()
	if firstUser != nil {
		fmt.Printf("First user: %s\n", firstUser.Username)
	}

	// Find by ID (like EF Core: context.Users.Find(id))
	if len(allUsers) > 0 {
		userByID, _ := ctx.Users.ById(allUsers[0].ID)
		if userByID != nil {
			fmt.Printf("Found by ID: %s\n", userByID.Username)
		}
	}
}

func demonstrateFiltering(ctx *BlogContext) {
	fmt.Println("\n🔍 Filtering (WHERE Clauses)")
	fmt.Println("----------------------------")

	// Simple equality (like EF Core: context.Users.Where(x => x.IsActive))
	activeUsers, _ := ctx.Users.WhereField("is_active", true).ToList()
	fmt.Printf("Active users: %d\n", len(activeUsers))

	// Age comparison (like EF Core: context.Users.Where(x => x.Age >= 30))
	olderUsers, _ := ctx.Users.WhereField("age", ">=30").ToList()
	fmt.Printf("Users 30 or older: %d\n", len(olderUsers))

	// Range queries (like EF Core: context.Users.Where(x => x.Age >= 25 && x.Age <= 35))
	ageRangeUsers, _ := ctx.Users.WhereFieldBetween("age", 25, 35).ToList()
	fmt.Printf("Users aged 25-35: %d\n", len(ageRangeUsers))

	// Multiple conditions
	youngActiveUsers, _ := ctx.Users.WhereField("is_active", true).WhereField("age", "<30").ToList()
	fmt.Printf("Young active users: %d\n", len(youngActiveUsers))
}

func demonstrateStringOperations(ctx *BlogContext) {
	fmt.Println("\n🔤 String Operations")
	fmt.Println("-------------------")

	// Contains (like EF Core: context.Users.Where(x => x.Email.Contains("gmail")))
	gmailUsers, _ := ctx.Users.WhereFieldLike("email", "gmail").ToList()
	fmt.Printf("Gmail users: %d\n", len(gmailUsers))

	// Starts with (like EF Core: context.Users.Where(x => x.FirstName.StartsWith("A")))
	aNameUsers, _ := ctx.Users.WhereFieldStartsWith("first_name", "A").ToList()
	fmt.Printf("Names starting with A: %d\n", len(aNameUsers))

	// Ends with (like EF Core: context.Users.Where(x => x.Email.EndsWith(".com")))
	comUsers, _ := ctx.Users.WhereFieldEndsWith("email", ".com").ToList()
	fmt.Printf("Email ending with .com: %d\n", len(comUsers))

	// Specific string match
	alice, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault()
	if alice != nil {
		fmt.Printf("Found specific user: %s %s\n", alice.FirstName, alice.LastName)
	}
}

func demonstrateOrdering(ctx *BlogContext) {
	fmt.Println("\n📊 Ordering")
	fmt.Println("-----------")

	// Order by username (like EF Core: context.Users.OrderBy(x => x.Username))
	usersByName, _ := ctx.Users.OrderByField("username").ToList()
	fmt.Printf("Users by username: %s, %s, %s...\n", 
		usersByName[0].Username, usersByName[1].Username, usersByName[2].Username)

	// Order by age descending (like EF Core: context.Users.OrderByDescending(x => x.Age))
	usersByAge, _ := ctx.Users.OrderByFieldDescending("age").ToList()
	fmt.Printf("Users by age (oldest first): %d, %d, %d...\n", 
		usersByAge[0].Age, usersByAge[1].Age, usersByAge[2].Age)

	// Order posts by view count
	popularPosts, _ := ctx.Posts.OrderByFieldDescending("view_count").Take(3).ToList()
	fmt.Printf("Top 3 posts by views: %d, %d, %d views\n", 
		popularPosts[0].ViewCount, popularPosts[1].ViewCount, popularPosts[2].ViewCount)
}

func demonstratePagination(ctx *BlogContext) {
	fmt.Println("\n📄 Pagination")
	fmt.Println("-------------")

	// Take first 3 (like EF Core: context.Users.Take(3))
	firstThree, _ := ctx.Users.OrderByField("username").Take(3).ToList()
	fmt.Printf("First 3 users: %d\n", len(firstThree))

	// Skip 2, take 2 (like EF Core: context.Users.Skip(2).Take(2))
	page2, _ := ctx.Users.OrderByField("username").Skip(2).Take(2).ToList()
	fmt.Printf("Page 2 (skip 2, take 2): %d users\n", len(page2))

	// Most popular posts (top 3)
	topPosts, _ := ctx.Posts.WhereField("is_published", true).OrderByFieldDescending("view_count").Take(3).ToList()
	fmt.Printf("Top 3 published posts: %d found\n", len(topPosts))
}

func demonstrateAggregations(ctx *BlogContext) {
	fmt.Println("\n🔢 Aggregations")
	fmt.Println("---------------")

	// Count with conditions (like EF Core: context.Users.Count(x => x.IsActive))
	activeCount, _ := ctx.Users.WhereField("is_active", true).Count()
	fmt.Printf("Active users count: %d\n", activeCount)

	// Count published posts
	publishedCount, _ := ctx.Posts.WhereField("is_published", true).Count()
	fmt.Printf("Published posts: %d\n", publishedCount)

	// Count posts by author
	allUsers, _ := ctx.Users.ToList()
	if len(allUsers) > 0 {
		userPosts, _ := ctx.Posts.WhereField("author_id", allUsers[0].ID).Count()
		fmt.Printf("%s has %d posts\n", allUsers[0].Username, userPosts)
	}
}

func demonstrateComplexQueries(ctx *BlogContext) {
	fmt.Println("\n🧠 Complex Queries")
	fmt.Println("------------------")

	// Chained conditions (like EF Core method chaining)
	complexQuery, _ := ctx.Users.
		WhereField("is_active", true).
		WhereField("age", ">=25").
		WhereFieldLike("email", "gmail").
		OrderByField("age").
		Take(5).
		ToList()
	fmt.Printf("Complex query result: %d users\n", len(complexQuery))

	// High-value posts (published with many views)
	popularContent, _ := ctx.Posts.
		WhereField("is_published", true).
		WhereField("view_count", ">100").
		OrderByFieldDescending("view_count").
		ToList()
	fmt.Printf("Popular content (>100 views): %d posts\n", len(popularContent))

	// Young, high-earning, active users
	premiumUsers, _ := ctx.Users.
		WhereField("is_active", true).
		WhereField("age", "<30").
		WhereField("salary", ">50000").
		ToList()
	fmt.Printf("Premium young users: %d\n", len(premiumUsers))
}

func demonstrateExistenceChecks(ctx *BlogContext) {
	fmt.Println("\n✅ Existence Checks")
	fmt.Println("-------------------")

	// Check if any users exist (like EF Core: context.Users.Any())
	hasUsers, _ := ctx.Users.Any()
	fmt.Printf("Has any users: %t\n", hasUsers)

	// Check if specific conditions exist (like EF Core: context.Users.Any(x => x.Age > 40))
	hasOldUsers, _ := ctx.Users.WhereField("age", ">40").Any()
	fmt.Printf("Has users over 40: %t\n", hasOldUsers)

	// Check for gmail users
	hasGmail, _ := ctx.Users.WhereFieldLike("email", "gmail").Any()
	fmt.Printf("Has Gmail users: %t\n", hasGmail)

	// Check for unpublished posts
	hasDrafts, _ := ctx.Posts.WhereField("is_published", false).Any()
	fmt.Printf("Has draft posts: %t\n", hasDrafts)

	// Check for high-view posts
	hasPopular, _ := ctx.Posts.WhereField("view_count", ">200").Any()
	fmt.Printf("Has posts with >200 views: %t\n", hasPopular)
}