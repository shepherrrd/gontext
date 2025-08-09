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
| `context.Users.Add(user)` | `ctx.Users.Add(user)` |
| `context.SaveChanges()` | `ctx.SaveChanges()` |
| `context.Users.Where(x => x.IsActive).ToList()` | `ctx.Users.WhereField("is_active", true).ToList()` |
| `context.Users.FirstOrDefault(x => x.Id == id)` | `ctx.Users.ById(id)` |
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