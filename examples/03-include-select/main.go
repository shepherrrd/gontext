package main

import (
	"fmt"
	"log"

	"github.com/shepherrrd/gontext"
)

// User entity with relationships
type User struct {
	ID       uint      `gorm:"primaryKey"`
	Name     string    `gorm:"not null"`
	Email    string    `gorm:"uniqueIndex;not null"`
	Posts    []Post    `gorm:"foreignKey:UserID"`
	Profile  *Profile  `gorm:"foreignKey:UserID"`
}

type Post struct {
	ID     uint   `gorm:"primaryKey"`
	Title  string `gorm:"not null"`
	UserID uint   `gorm:"not null"`
}

type Profile struct {
	ID     uint   `gorm:"primaryKey"`
	Bio    string
	UserID uint `gorm:"uniqueIndex;not null"`
}

func main() {
	// Create database context
	ctx, err := gontext.NewDbContext("postgres://user:password@localhost/testdb?sslmode=disable", "postgres")
	if err != nil {
		log.Fatal(err)
	}
	defer ctx.Close()

	// Register entities
	users := gontext.RegisterEntity[User](ctx)
	gontext.RegisterEntity[Post](ctx)
	gontext.RegisterEntity[Profile](ctx)

	// Example 1: Include specific relationships (type-safe)
	fmt.Println("=== Include Examples ===")

	// Load users with their posts
	usersWithPosts, _ := users.Include("Posts").ToList()
	fmt.Printf("Loaded %d users with posts\n", len(usersWithPosts))

	// Load users with multiple relationships
	usersWithAll, _ := users.Include("Posts", "Profile").ToList()
	fmt.Printf("Loaded %d users with posts and profiles\n", len(usersWithAll))

	// Example 2: Auto-include all relationships
	usersAutoInclude, _ := users.IncludeAll().ToList()
	fmt.Printf("Loaded %d users with all relationships\n", len(usersAutoInclude))

	// Example 3: Select specific fields only
	fmt.Println("\n=== Select Examples ===")

	// Load only specific fields
	usernames, _ := users.Select("ID", "Name").ToList()
	fmt.Printf("Loaded %d users with only ID and Name\n", len(usernames))

	// Exclude sensitive fields
	publicUsers, _ := users.Omit("Email").ToList()
	fmt.Printf("Loaded %d users without email\n", len(publicUsers))

	// Example 4: Combine Include and Select
	fmt.Println("\n=== Combined Examples ===")

	// Include relationships but select specific fields
	combinedQuery, _ := users.
		Include("Posts").
		Select("ID", "Name").
		Where("name LIKE ?", "John%").
		ToList()
	fmt.Printf("Loaded %d users named John* with posts, showing only ID and Name\n", len(combinedQuery))

	// Example 5: Error handling - invalid field name
	fmt.Println("\n=== Error Example ===")
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Expected error: %v\n", r)
		}
	}()

	// This will panic with: Field 'InvalidField' not found on User
	users.Include("InvalidField").ToList()
}