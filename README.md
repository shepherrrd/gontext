# GoNtext - Entity Framework Core for Go

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/shepherrrd/gontext?status.svg)](https://godoc.org/github.com/shepherrrd/gontext)

GoNtext brings the familiar **Entity Framework Core** patterns and conventions to Go, providing a type-safe, LINQ-style ORM with automatic change tracking, migrations, and fluent querying capabilities.

## ✨ Features

- **🎯 EF Core-Style API**: Familiar patterns for .NET developers
- **🔍 LINQ Queries**: Type-safe querying with method chaining
- **📊 Change Tracking**: Automatic entity change detection
- **🔄 Migrations**: Code-first database migrations with Go files
- **💾 DbSets**: Type-safe entity collections with generics
- **🗃️ Multiple Databases**: PostgreSQL, MySQL, SQLite support
- **🏗️ Code First**: Define your entities in Go, generate database schema
- **⚡ GORM Backend**: Built on top of GORM for reliability and performance

## 🚀 Quick Start

### Installation

```bash
go get github.com/shepherrrd/gontext
```

### Instant CLI Usage

No build required! After installation, use GoNtext CLI directly:

```bash
# Set database connection
export DATABASE_URL="postgres://user:pass@localhost/db?sslmode=disable"

# Generate migration (like EF Core Add-Migration)
go run github.com/shepherrrd/gontext/cmd/gontext migration add InitialCreate

# Apply migrations (like EF Core Update-Database)  
go run github.com/shepherrrd/gontext/cmd/gontext database update
```

### Basic Usage

```go
package main

import (
    "github.com/google/uuid"
    "github.com/shepherrrd/gontext"
)

// Define your entities
type User struct {
    ID        uuid.UUID `gontext:"primary_key;default:gen_random_uuid()"`
    Username  string    `gontext:"unique;not_null"`
    Email     string    `gontext:"unique;not_null"`
    FirstName string    `gontext:"not_null"`
    IsActive  bool      `gontext:"not_null;default:true"`
}

// Create your DbContext
type AppContext struct {
    *gontext.DbContext
    Users *gontext.LinqDbSet[User] // EF Core: DbSet<User> Users { get; set; }
}

func NewAppContext(connectionString string) (*AppContext, error) {
    ctx, err := gontext.NewDbContext(connectionString, "postgres")
    if err != nil {
        return nil, err
    }

    // Register entities (returns LinqDbSet directly)
    users := gontext.RegisterEntity[User](ctx)

    return &AppContext{
        DbContext: ctx,
        Users:     users,
    }, nil
}
```

func main() {
    ctx, err := NewAppContext("postgres://user:pass@localhost/mydb?sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer ctx.Close()

    // Ensure database tables exist
    ctx.EnsureCreated()

    // EF Core-style operations
    user := User{
        Username:  "alice",
        Email:     "alice@example.com",
        FirstName: "Alice",
        IsActive:  true,
    }

    // Add entity to context
    ctx.Users.Add(user)
    
    // Save changes to database
    ctx.SaveChanges()

    // LINQ-style queries
    activeUsers, _ := ctx.Users.WhereField("is_active", true).ToList()
    userCount, _ := ctx.Users.Count()
    alice, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault()

    // Change tracking - modify and save
    alice.FirstName = "Alice Updated"
    ctx.SaveChanges() // Automatically detects changes!

    // Remove entities
    ctx.Users.Remove(*alice)
    ctx.SaveChanges()
}
```

## 🔍 LINQ Queries

GoNtext provides EF Core-style LINQ methods for type-safe querying:

```go
// Basic queries
users, _ := ctx.Users.ToList()
activeUsers, _ := ctx.Users.WhereField("is_active", true).ToList()
userCount, _ := ctx.Users.Count()

// Single record queries  
user, _ := ctx.Users.ById(userID)                                    // Find by ID
alice, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault() // First or nil
bob, _ := ctx.Users.WhereField("email", "bob@example.com").Single()    // Exactly one

// Existence checks
hasActiveUsers, _ := ctx.Users.WhereField("is_active", true).Any()
hasAlice, _ := ctx.Users.WhereField("username", "alice").Any()

// String operations
gmailUsers, _ := ctx.Users.WhereFieldLike("email", "gmail")           // Contains
aUsers, _ := ctx.Users.WhereFieldStartsWith("username", "a")          // Starts with
comUsers, _ := ctx.Users.WhereFieldEndsWith("email", ".com")          // Ends with

// Ordering and pagination
sortedUsers, _ := ctx.Users.OrderByField("username").ToList()
topUsers, _ := ctx.Users.OrderByFieldDescending("created_at").Take(10).ToList()
pagedUsers, _ := ctx.Users.Skip(20).Take(10).ToList()

// Range queries
adults, _ := ctx.Users.WhereFieldBetween("age", 18, 65).ToList()
youngUsers, _ := ctx.Users.WhereField("age", "<30").ToList()
```

## ⚡ CLI Usage (No Build Required!)

GoNtext provides a powerful CLI that you can run directly without building:

```bash
# Install GoNtext
go get github.com/shepherrrd/gontext

# Set database connection (environment variable or .env file)
export DATABASE_URL="postgres://user:pass@localhost/db?sslmode=disable"

# Migration commands (EF Core style)
go run github.com/shepherrrd/gontext/cmd/gontext migration add CreateUserTable
go run github.com/shepherrrd/gontext/cmd/gontext migration add AddUserAge  
go run github.com/shepherrrd/gontext/cmd/gontext migration list
go run github.com/shepherrrd/gontext/cmd/gontext migration remove

# Database commands
go run github.com/shepherrrd/gontext/cmd/gontext database update    # Apply migrations
go run github.com/shepherrrd/gontext/cmd/gontext database rollback 2  # Rollback 2 migrations  
go run github.com/shepherrrd/gontext/cmd/gontext database drop     # Drop all tables

# Get help
go run github.com/shepherrrd/gontext/cmd/gontext help
```

### Database Connection

Set your database connection using either:

**Environment Variable:**
```bash
export DATABASE_URL="postgres://user:pass@localhost/db?sslmode=disable"
```

**Or .env file in your project:**
```env
DATABASE_URL=postgres://user:pass@localhost/db?sslmode=disable
```

## 📊 Entity Operations

### EF Core-Style CRUD

```go
// CREATE
user := User{Username: "alice", Email: "alice@example.com"}
ctx.Users.Add(user)        // Add single entity
ctx.Users.AddRange(users)  // Add multiple entities
ctx.SaveChanges()          // Persist to database


// READ with change tracking
user, _ := ctx.Users.Find(userID)  // Returns tracked entity
users, _ := ctx.Users.WhereField("is_active", true).ToList()

// UPDATE with change tracking
user.FirstName = "Updated Name"    // Modify tracked entity
ctx.SaveChanges()                  // Auto-detects and saves changes

// UPDATE explicit
ctx.Users.Update(user)             // Explicitly mark for update
ctx.SaveChanges()                  // Apply changes

// DELETE
ctx.Users.Remove(user)             // Mark for deletion
ctx.Users.RemoveRange(users)       // Mark multiple for deletion
ctx.SaveChanges()                  // Execute deletions
```

### Advanced Queries

```go
// Method chaining
results, err := gontext.LINQ[User](ctx.DbContext).
    Where("age >= ?", 21).
    OrderByDescending("created_at").
    Take(10).
    Skip(5).
    ToList()

// String operations
users, err := gontext.LINQ[User](ctx.DbContext).
    StartsWith("username", "john").
    StringContains("email", "gmail").
    ToList()

// Aggregations
count, err := gontext.LINQ[User](ctx.DbContext).
    Where("is_active = ?", true).
    Count()

totalAge, err := gontext.LINQ[User](ctx.DbContext).Sum("age")
avgAge, err := gontext.LINQ[User](ctx.DbContext).Average("age")
```

### Complex Queries

```go
// Multiple conditions
users, err := gontext.LINQ[User](ctx.DbContext).
    Where("age BETWEEN ? AND ?", 18, 65).
    Where("is_active = ?", true).
    In("username", "alice", "bob", "charlie").
    OrderBy("username").
    ToList()

// Pagination
page2Users, err := gontext.LINQ[User](ctx.DbContext).
    OrderBy("created_at").
    Skip(20).        // Skip first 20
    Take(10).        // Take next 10
    ToList()

// Distinct values
uniqueAges, err := gontext.LINQ[User](ctx.DbContext).
    Distinct("age").
    ToList()
```

### Available LINQ Methods

| Method | Description | Example |
|--------|-------------|---------|
| `Where(condition, args...)` | Filter records | `.Where("age > ?", 21)` |
| `OrderBy(column)` | Sort ascending | `.OrderBy("username")` |
| `OrderByDescending(column)` | Sort descending | `.OrderByDescending("created_at")` |
| `Take(n)` | Limit results | `.Take(10)` |
| `Skip(n)` | Skip results | `.Skip(5)` |
| `First()` | Get first record | `.First()` |
| `FirstOrDefault()` | Get first or nil | `.FirstOrDefault()` |
| `Single()` | Get exactly one | `.Single()` |
| `Count()` | Count records | `.Count()` |
| `Any()` | Check if any exist | `.Any()` |
| `Sum(column)` | Sum values | `.Sum("age")` |
| `Average(column)` | Average values | `.Average("age")` |
| `Min(column)` | Minimum value | `.Min("age")` |
| `Max(column)` | Maximum value | `.Max("age")` |
| `StartsWith(col, val)` | String starts with | `.StartsWith("name", "John")` |
| `EndsWith(col, val)` | String ends with | `.EndsWith("email", ".com")` |
| `StringContains(col, val)` | String contains | `.StringContains("name", "test")` |
| `In(col, values...)` | Value in list | `.In("id", id1, id2, id3)` |
| `Between(col, start, end)` | Value in range | `.Between("age", 18, 65)` |
| `IsNull(column)` | Is null check | `.IsNull("deleted_at")` |
| `IsNotNull(column)` | Is not null check | `.IsNotNull("email")` |

## Database Operations

### Create
```go
user := &User{Username: "john", Email: "john@example.com", Age: 30}
ctx.Users.Add(user)
ctx.SaveChanges()
```

### Read
```go
// LINQ queries
users, err := gontext.LINQ[User](ctx.DbContext).
    Where("is_active = ?", true).
    OrderBy("username").
    ToList()
```

### Update
```go
user.Age = 31
ctx.Users.Update(user)
ctx.SaveChanges()
```

### Delete
```go
ctx.Users.Remove(user)
ctx.SaveChanges()
```

## 🔄 Migrations

GoNtext provides a migration system similar to EF Core. After installing GoNtext, you can use the CLI directly with `go run`:

### Installation & Setup

```bash
# Install GoNtext
go get github.com/shepherrrd/gontext

# Set your database connection
export DATABASE_URL="postgres://user:pass@localhost/db?sslmode=disable"
# Or create a .env file with DATABASE_URL=your_connection_string
```

### Migration Commands

```bash
# Generate a new migration (like EF Core's Add-Migration)
go run github.com/shepherrrd/gontext/cmd/gontext migration add InitialCreate

# Apply pending migrations (like EF Core's Update-Database)
go run github.com/shepherrrd/gontext/cmd/gontext database update

# List all migrations
go run github.com/shepherrrd/gontext/cmd/gontext migration list

# Remove the last migration
go run github.com/shepherrrd/gontext/cmd/gontext migration remove

# Drop all database tables
go run github.com/shepherrrd/gontext/cmd/gontext database drop

# Rollback migrations
go run github.com/shepherrrd/gontext/cmd/gontext database rollback 2
```

### What Gets Generated

When you run `migration add`, GoNtext creates:
- `ModelSnapshot.json` - Current database schema snapshot
- `20231201120000_initialcreate.go` - Migration file with Up/Down methods

### Migration Files

Generated migration files contain Go code:

```go
func (m *Migration20231201120000) Up(db *gorm.DB) error {
    // Create table users
    if err := db.Exec("CREATE TABLE users (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), username TEXT NOT NULL UNIQUE, email TEXT NOT NULL UNIQUE, created_at TIMESTAMP NOT NULL)").Error; err != nil {
        return err
    }
    return nil
}

func (m *Migration20231201120000) Down(db *gorm.DB) error {
    // Drop table users
    if err := db.Exec("DROP TABLE IF EXISTS users").Error; err != nil {
        return err
    }
    return nil
}
```

### Column Renames

Use `old_name` tag to rename columns without data loss:

```go
type User struct {
    ID       uuid.UUID `gontext:"primary_key"`
    Username string    `gontext:"unique;not_null;old_name:user_name"` // Renames user_name → username
    Email    string    `gontext:"unique;not_null"`
}
```

## Entity Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `primary_key` | Primary key | `gontext:"primary_key"` |
| `not_null` | NOT NULL constraint | `gontext:"not_null"` |
| `nullable` | Allow NULL | `gontext:"nullable"` |
| `unique` | UNIQUE constraint | `gontext:"unique"` |
| `default:value` | Default value | `gontext:"default:'active'"` |
| `old_name:name` | Column rename | `gontext:"old_name:old_column"` |

## 🗃️ Supported Databases

GoNtext supports multiple database backends:

```go
// PostgreSQL
ctx, _ := gontext.NewDbContext("postgres://user:pass@localhost/db?sslmode=disable", "postgres")

// MySQL
ctx, _ := gontext.NewDbContext("user:pass@tcp(localhost:3306)/db?charset=utf8mb4&parseTime=True&loc=Local", "mysql")

// SQLite
ctx, _ := gontext.NewDbContext("./database.db", "sqlite")
```

## 🤝 EF Core Comparison

| EF Core (C#) | GoNtext (Go) |
|--------------|--------------|
| `context.Users.Add(user)` | `ctx.Users.Add(user)` |
| `context.SaveChanges()` | `ctx.SaveChanges()` |
| `context.Users.Where(x => x.IsActive).ToList()` | `ctx.Users.WhereField("is_active", true).ToList()` |
| `context.Users.FirstOrDefault(x => x.Id == id)` | `ctx.Users.ById(id)` |
| `context.Users.Count(x => x.IsActive)` | `ctx.Users.WhereField("is_active", true).Count()` |
| `context.Users.Any(x => x.Username == "alice")` | `ctx.Users.WhereField("username", "alice").Any()` |
| `Add-Migration InitialCreate` | `go run github.com/shepherrrd/gontext/cmd/gontext migration add InitialCreate` |
| `Update-Database` | `go run github.com/shepherrrd/gontext/cmd/gontext database update` |
| `Remove-Migration` | `go run github.com/shepherrrd/gontext/cmd/gontext migration remove` |
| `Drop-Database` | `go run github.com/shepherrrd/gontext/cmd/gontext database drop` |

## 📖 Examples

Check out the `/example` directory for a complete working example demonstrating:

- Database context setup
- Entity registration
- Migrations
- CRUD operations with LINQ
- Change tracking
- Transaction handling

Run the example:

```bash
cd example
go run main.go
```

## Advanced Usage

### Access Underlying GORM

```go
// Get GORM database for complex queries
db := ctx.GetDB()

// Use GORM directly when needed
var result CustomResult
db.Raw("SELECT custom_field FROM complex_view WHERE condition = ?", value).Scan(&result)
```

### Transactions

```go
tx := ctx.BeginTransaction()
defer tx.Rollback()

tx.Create(&user)
tx.Create(&post)

tx.Commit()
```

### Multiple Database Support

```go
// PostgreSQL
ctx, err := gontext.NewDbContext("postgres://...", "postgres")

// MySQL
ctx, err := gontext.NewDbContext("mysql://...", "mysql") 

// SQLite
ctx, err := gontext.NewDbContext("./db.sqlite", "sqlite")
```

## Why GoNtext?

- **🎯 Familiar**: Uses EF Core patterns you already know
- **🔍 Type-Safe**: LINQ queries with compile-time checking
- **🚀 Powerful**: Full SQL capabilities via GORM integration
- **📦 Complete**: Migrations, change tracking, and DbContext patterns
- **🔄 Flexible**: Use LINQ or drop down to raw SQL when needed

GoNtext brings the best of Entity Framework Core to Go!