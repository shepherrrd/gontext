# GoNtext Migrations & Schema Management

This example demonstrates how to properly set up and use GoNtext migrations, including model snapshots and schema evolution - the features that were missing!

## 🚀 Quick Start

### Prerequisites
- PostgreSQL running on localhost:5432  
- Database named `test_migrations` created

```bash
createdb test_migrations
```

### Run the Example
```bash
go mod tidy

# See available commands
go run .

# Create your first migration
go run . migrate:add InitialCreate

# Apply migrations to database
go run . migrate:update

# Check migration status
go run . migrate:status

# List all migration files
go run . migrate:list
```

## 📖 What You'll Learn

### 1. Design-Time Context Setup (CRITICAL)

The key to making migrations work is the `CreateDesignTimeContext()` function:

```go
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
    // MUST register every entity you want in migrations

    return ctx, nil
}
```

### 2. Model Snapshots (Like EF Core)

GoNtext generates `ModelSnapshot.json` files that track your schema:

```json
{
  "version": "1.0",
  "timestamp": "2024-01-01T12:00:00Z",
  "entities": {
    "User": {
      "name": "User",
      "tableName": "users",
      "fields": {
        "ID": {
          "name": "id",
          "type": "uuid.UUID", 
          "nullable": false
        },
        "Username": {
          "name": "username",
          "type": "string",
          "nullable": false
        }
      }
    }
  }
}
```

### 3. Migration Generation Process

1. **Current Snapshot**: Generated from registered entities
2. **Previous Snapshot**: Loaded from `ModelSnapshot.json`  
3. **Comparison**: Detects changes between snapshots
4. **Migration File**: Generated with Up/Down methods
5. **Snapshot Update**: Current snapshot saved for next migration

### 4. Custom Migration Commands

```bash
# Create a new migration
go run . migrate:add AddAgeField

# Apply migrations (creates/updates tables)
go run . migrate:update

# Check what's tracked
go run . migrate:status

# List all migration files
go run . migrate:list
```

## 📊 Example Migration Workflow

### Step 1: Create Initial Migration
```bash
$ go run . migrate:add InitialCreate
🔄 Adding migration: InitialCreate
📊 Detected 2 schema changes
  • Create table: User
  • Create table: Post
📁 Generated: ./migrations/20240101120000_InitialCreate.go
✅ Migration 'InitialCreate' created successfully!
```

### Step 2: Check Status
```bash
$ go run . migrate:status
📊 Migration Status
==================
✅ Model snapshot: 2024-01-01T12:00:00Z
📊 Tracked entities: 2
   • User (users) - 6 fields
   • Post (posts) - 5 fields
📁 Migration files: 1
```

### Step 3: Apply Migration
```bash
$ go run . migrate:update
🔄 Updating database...
✅ Database updated successfully!
```

## 🗂️ Generated Files

### Migration File Structure
```
migrations/
├── ModelSnapshot.json          # Current schema snapshot
├── 20240101120000_InitialCreate.go   # Migration with Up/Down methods
└── 20240102130000_AddAgeField.go     # Next migration
```

### Generated Migration File
```go
// Code generated migration. DO NOT EDIT.
package migrations

import "gorm.io/gorm"

type Migration20240101120000 struct{}

func (m *Migration20240101120000) ID() string {
    return "20240101120000_InitialCreate"
}

func (m *Migration20240101120000) Up(db *gorm.DB) error {
    // Create table users
    if err := db.AutoMigrate(&User{}); err != nil {
        return err
    }
    
    // Create table posts  
    if err := db.AutoMigrate(&Post{}); err != nil {
        return err
    }
    
    return nil
}

func (m *Migration20240101120000) Down(db *gorm.DB) error {
    // Drop table users
    if err := db.Exec("DROP TABLE IF EXISTS users CASCADE").Error; err != nil {
        return err
    }
    
    // Drop table posts
    if err := db.Exec("DROP TABLE IF EXISTS posts CASCADE").Error; err != nil {
        return err
    }
    
    return nil
}
```

## 🔧 Schema Evolution Example

### Add a New Field
1. **Modify your entity**:
```go
type User struct {
    ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Username  string    `gorm:"uniqueIndex;not null"`
    Email     string    `gorm:"uniqueIndex;not null"`
    Age       int       // NEW FIELD
    CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}
```

2. **Generate migration**:
```bash
go run . migrate:add AddAgeField
```

3. **Migration automatically detects**:
```
📊 Detected 1 schema changes
  • Add field: User.Age
```

## 🎯 Key Features

### ✅ What Works
- **Model Snapshots**: Track schema changes over time
- **Change Detection**: Automatically detects entity modifications  
- **Migration Files**: Generates actual .go migration files
- **Up/Down Methods**: Proper migration rollback support
- **Entity Registration**: Explicit entity discovery

### 🔄 Migration Commands
| Command | Purpose | EF Core Equivalent |
|---------|---------|-------------------|
| `migrate:add <name>` | Create migration | `Add-Migration` |
| `migrate:update` | Apply migrations | `Update-Database` |
| `migrate:status` | Show current state | N/A (custom) |
| `migrate:list` | List migration files | N/A (custom) |

## 💡 Best Practices

### 1. Always Register All Entities
```go
func CreateDesignTimeContext() (*gontext.DbContext, error) {
    // Register EVERY entity you want in migrations
    gontext.RegisterEntity[User](ctx)
    gontext.RegisterEntity[Post](ctx)
    gontext.RegisterEntity[Category](ctx)
    // Missing entities = broken migrations!
}
```

### 2. Use Proper GORM Tags
```go
type User struct {
    ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Username string    `gorm:"uniqueIndex;not null"`
    // GORM tags are essential for proper schema generation
}
```

### 3. Test Your Migrations
```bash
# Always test the complete workflow
go run . migrate:add TestMigration
go run . migrate:update
go run . migrate:status
```

## 🔗 Next Steps

- [CRUD Operations](../01-crud/) - Basic database operations
- [LINQ Queries](../03-linq/) - Advanced querying capabilities

This example shows the **correct way** to implement GoNtext migrations with proper snapshots and change detection!