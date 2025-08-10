# GoNtext - Entity Framework Core for Go

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![GoDoc](https://godoc.org/github.com/shepherrrd/gontext?status.svg)](https://godoc.org/github.com/shepherrrd/gontext)

GoNtext brings the familiar **Entity Framework Core** patterns to Go, providing a type-safe, LINQ-style ORM with automatic change tracking, migrations, and fluent querying capabilities.

## ✨ Features

- **🎯 EF Core-Style API**: Familiar patterns for .NET developers
- **🔍 LINQ Queries**: Type-safe querying with method chaining
- **📊 Change Tracking**: Automatic entity change detection
- **🔄 Migrations**: Code-first database migrations with Go files
- **💾 DbSets**: Type-safe entity collections with generics
- **🗃️ Multiple Databases**: PostgreSQL, MySQL, SQLite support
- **🐘 PostgreSQL Pascal Case**: Automatic field name translation with quoted identifiers
- **🚀 Zero Configuration**: Automatic database-specific optimizations

## 🚀 Quick Start

### Installation

```bash
go get github.com/shepherrrd/gontext
```

### Basic Setup

```go
package main

import (
    "github.com/google/uuid"
    "github.com/shepherrrd/gontext"
)

// Define your entities
type User struct {
    ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Username string    `gorm:"uniqueIndex;not null"`
    Email    string    `gorm:"uniqueIndex;not null"`
    Name     string    `gorm:"not null"`
}

// Create your DbContext
type AppContext struct {
    *gontext.DbContext
    Users *gontext.LinqDbSet[User]
}

func NewAppContext(connectionString string) (*AppContext, error) {
    ctx, err := gontext.NewDbContext(connectionString, "postgres")
    if err != nil {
        return nil, err
    }

    users := gontext.RegisterEntity[User](ctx)

    return &AppContext{
        DbContext: ctx,
        Users:     users,
    }, nil
}
```

## 🐘 PostgreSQL Pascal Case Support

**GoNtext automatically handles PostgreSQL case-sensitive identifiers!** No manual configuration required.

### ✨ How It Works

When using PostgreSQL, GoNtext automatically:
- **Uses struct names as table names**: `User` struct → `"User"` table (not `users`)
- **Uses field names as column names**: `Username` field → `"Username"` column  
- **Quotes all identifiers**: Proper PostgreSQL case-sensitive handling
- **Translates all queries**: WHERE, ORDER BY, SELECT - everything works automatically

### 📝 Example

```go
type User struct {
    ID           uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Username     string    `gorm:"uniqueIndex;not null"`
    Email        string    `gorm:"uniqueIndex;not null"`
    PasswordHash string    `gorm:"not null"`
    IsActive     bool      `gorm:"not null;default:true"`
    CreatedAt    time.Time `gorm:"autoCreateTime"`
}

// Setup - Zero configuration needed!
ctx, _ := gontext.NewDbContext("postgres://user:pass@localhost/db", "postgres")
userSet := gontext.RegisterEntity[User](ctx)

// All queries automatically use quoted PostgreSQL identifiers:
user, _ := userSet.WhereField("Username", "john").FirstOrDefault()
// SQL: SELECT * FROM "User" WHERE "Username" = 'john'

users, _ := userSet.OrderByField("Email").ToList()  
// SQL: SELECT * FROM "User" ORDER BY "Email" ASC

userSet.WhereField("IsActive", true).Delete()
// SQL: DELETE FROM "User" WHERE "IsActive" = true
```

### 🎯 What You Get

- ✅ **Table Names**: `User` struct becomes `"User"` table (Pascal case)
- ✅ **Column Names**: `Username` field becomes `"Username"` column (Pascal case)
- ✅ **All Query Types**: INSERT, SELECT, UPDATE, DELETE - all automatically translated
- ✅ **Complex Queries**: WHERE with AND/OR/parentheses, LIKE, IN - all supported
- ✅ **Zero Boilerplate**: No `TableName()` methods needed, no manual quoting

### 🚫 No More TableName() Methods

**OLD WAY** (not needed anymore):
```go
func (User) TableName() string {
    return "User" // ❌ Don't do this anymore!
}
```

**NEW WAY** (automatic):
```go
type User struct {
    // Just define your struct - GoNtext handles the rest! ✅
    Username string
    Email    string
}
```

GoNtext automatically uses the struct name (`User`) as the table name with proper PostgreSQL quoting.

## 🏷️ Custom Table Names

**You can override the default table naming by implementing the `TableName()` method:**

### 📝 Example

```go
type User struct {
    Id       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Username string    `gorm:"uniqueIndex;not null"`
    Email    string    `gorm:"uniqueIndex;not null"`
}

// Custom table name - overrides default "User" 
func (User) TableName() string {
    return "app_users" // Will create "app_users" table instead of "User"
}

type Product struct {
    Id   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Name string    `gorm:"not null"`
}

// No TableName() method - uses default "Product" table name
```

### ✨ How It Works

GoNtext respects the `TableName()` method across **all operations**:

- ✅ **CRUD Operations**: `ctx.Users.Add()`, `ctx.Users.Find()`, etc. use custom table name
- ✅ **LINQ Queries**: `ctx.Users.WhereField().ToList()` uses custom table name  
- ✅ **Migrations**: Migration files generate SQL for custom table names
- ✅ **PostgreSQL**: Automatically quotes custom table names: `"app_users"`

```go
// Setup with custom table names
ctx, _ := gontext.NewDbContext("postgres://user:pass@localhost/db", "postgres")
userSet := gontext.RegisterEntity[User](ctx)   // Uses "app_users" table
productSet := gontext.RegisterEntity[Product](ctx) // Uses "Product" table

// All operations use the custom table names automatically
user, _ := userSet.WhereField("Username", "john").FirstOrDefault()
// SQL: SELECT * FROM "app_users" WHERE "Username" = 'john'

product, _ := productSet.WhereField("Name", "laptop").FirstOrDefault()
// SQL: SELECT * FROM "Product" WHERE "Name" = 'laptop'
```

### 🎯 When to Use Custom Table Names

- **Legacy databases**: Match existing table names
- **Naming conventions**: Follow company/team standards (e.g., `tbl_users`, `app_users`)
- **Multi-tenant**: Different table prefixes per tenant
- **Database conventions**: Snake_case, plural names, etc.

## 📚 Examples

Choose what you want to learn:

### 🔨 [Basic CRUD Operations](./examples/01-crud/)
Learn the fundamentals:
- Creating entities
- Saving changes  
- Querying data
- Updates and deletes

### 🗃️ [Migrations & Schema](./examples/02-migrations/)
Database schema management:
- Creating migrations
- Model snapshots
- Schema evolution
- Database updates

### 🔍 [LINQ Queries](./examples/03-linq/)
Advanced querying:
- Where conditions
- Ordering and pagination
- String operations
- Aggregations
- Method chaining

## ⚠️ Important: Migration Setup

**The built-in CLI has limitations**. For proper migrations, you need to set up entity registration:

### Step 1: Create Design-Time Context
```go
// File: migrations_context.go
func CreateDesignTimeContext() (*gontext.DbContext, error) {
    ctx, err := gontext.NewDbContext("your-db-url", "postgres")
    if err != nil {
        return nil, err
    }

    // Register ALL your entities
    gontext.RegisterEntity[User](ctx)
    gontext.RegisterEntity[Post](ctx)
    // ... register every entity

    return ctx, nil
}
```

### Step 2: Add Migration Commands
```go
// Add to your main.go
func handleMigrations() {
    if len(os.Args) > 1 && os.Args[1] == "migrate:add" {
        // Use your design-time context for migrations
        ctx, _ := CreateDesignTimeContext()
        // Generate migration files
    }
}
```

**See [Migrations Example](./examples/02-migrations/) for complete setup.**

## 🎯 Why GoNtext?

- **🎯 Familiar**: Uses EF Core patterns you already know
- **🔍 Type-Safe**: LINQ queries with compile-time checking
- **🚀 Powerful**: Full SQL capabilities via GORM integration
- **📦 Complete**: Migrations, change tracking, and DbContext patterns
- **🔄 Flexible**: Use LINQ or drop down to raw SQL when needed

GoNtext brings the best of Entity Framework Core to Go!

## 🤝 EF Core Comparison

| EF Core (C#) | GoNtext (Go) |
|--------------|--------------|
| `context.Users.Add(user)` | `ctx.Users.Add(user); ctx.SaveChanges()` |
| `context.SaveChanges()` | `ctx.SaveChanges()` |
| `context.Users.Where(x => x.IsActive).ToList()` | `ctx.Users.WhereField("IsActive", true).ToList()` |
| `context.Users.FirstOrDefault(x => x.Id == id)` | `ctx.Users.ById(id)` |
| `context.Users.OrderBy(x => x.Email)` | `ctx.Users.OrderByField("Email")` |
| Pascal case tables (`Users`) | Pascal case tables (`"User"`) ✨ |
| Pascal case columns (`IsActive`) | Pascal case columns (`"IsActive"`) ✨ |
| `[Table("app_users")] class User` | `func (User) TableName() string { return "app_users" }` ✨ |
| `Add-Migration InitialCreate` | Custom migration commands |
| `Update-Database` | Custom migration commands |

## 📖 Documentation

- [CRUD Operations Guide](./examples/01-crud/README.md)
- [Migrations Setup Guide](./examples/02-migrations/README.md)  
- [LINQ Queries Guide](./examples/03-linq/README.md)

## 🏃‍♂️ Quick Test

```bash
# Clone and test
git clone https://github.com/shepherrrd/gontext
cd gontext/examples/01-crud
go mod tidy
createdb test_gontext
go run .
```