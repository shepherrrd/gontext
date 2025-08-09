# GoNtext CRUD Operations Example

This example demonstrates basic CRUD (Create, Read, Update, Delete) operations using GoNtext with an EF Core-like API.

## 🚀 Quick Start

### Prerequisites
- PostgreSQL running on localhost:5432
- Database named `test_gontext` created

```bash
createdb test_gontext
```

### Run the Example
```bash
go mod tidy
go run .
```

## 📖 What You'll Learn

### 1. Entity Definition
```go
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
```

### 2. DbContext Setup
```go
type BlogContext struct {
    *gontext.DbContext
    Users *gontext.LinqDbSet[User]  // Like EF Core's DbSet<User>
}

func NewBlogContext(connectionString string) (*BlogContext, error) {
    ctx, err := gontext.NewDbContext(connectionString, "postgres")
    users := gontext.RegisterEntity[User](ctx)
    
    return &BlogContext{
        DbContext: ctx,
        Users:     users,
    }, nil
}
```

### 3. CRUD Operations

#### Create (INSERT)
```go
user := User{ID: uuid.New(), Username: "alice", Email: "alice@example.com"}
ctx.Users.Add(user)        // Like EF Core: context.Users.Add(user)
ctx.SaveChanges()          // Like EF Core: context.SaveChanges()
```

#### Read (SELECT)
```go
// Get all users
users, _ := ctx.Users.ToList()

// Count users
count, _ := ctx.Users.Count()

// Find specific user
alice, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault()

// Filter users
activeUsers, _ := ctx.Users.WhereField("is_active", true).ToList()
```

#### Update (UPDATE)
```go
user, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault()
user.Age = 26              // Modify the entity
ctx.SaveChanges()          // Change tracking automatically detects changes
```

#### Delete (DELETE)
```go
user, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault()
ctx.Users.Remove(*user)    // Like EF Core: context.Users.Remove(user)
ctx.SaveChanges()          // Execute the deletion
```

## 🔑 Key Features Demonstrated

### EF Core-Style API
- `ctx.Users.Add()` - Add entity to context
- `ctx.SaveChanges()` - Persist all changes
- `ctx.Users.ToList()` - Get all entities
- `ctx.Users.Count()` - Count entities
- `ctx.Users.Remove()` - Mark for deletion

### Change Tracking
- Modify entities directly
- `SaveChanges()` automatically detects changes
- No need to explicitly mark entities as modified

### Type Safety
- Generic `LinqDbSet[User]` provides compile-time safety
- Method chaining with IntelliSense support
- No string-based entity names

## 📊 Example Output

```bash
🚀 GoNtext CRUD Example
=======================
📋 Creating tables...

🔨 CREATE Operations
-------------------
➕ Added user: alice
➕ Added user: bob
➕ Added user: charlie
💾 All users saved to database

📖 READ Operations
-----------------
👥 Total users: 3
📋 Retrieved 3 users
🔎 Found Alice: Alice Smith (Age: 25)
✅ Active users: 3

✏️ UPDATE Operations
-------------------
Before update: Bob is 30 years old
💾 Bob updated with change tracking
After update: Bobby is 31 years old

🗑️ DELETE Operations
-------------------
Deleting user: charlie
💾 Charlie deleted from database
👥 Users remaining: 2

✅ CRUD demo completed!
```

## 🎯 Key Takeaways

1. **Familiar API**: If you know EF Core, you know GoNtext
2. **Change Tracking**: Automatically detects entity modifications
3. **Type Safety**: Compile-time safety with generics
4. **Simple Setup**: Minimal configuration required
5. **GORM Backend**: Full GORM power when needed

## 🔗 Next Steps

- [Migrations Example](../02-migrations/) - Learn database schema management
- [LINQ Queries Example](../03-linq/) - Advanced querying capabilities