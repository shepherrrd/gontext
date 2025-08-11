# GoNtext API Reference

Complete reference for all GoNtext LINQ methods with examples and use cases.

## 🏗️ Core Setup Methods

### DbContext Creation
```go
// Create new database context
ctx, err := gontext.NewDbContext("connection_string", "postgres")

// Register entities (creates typed DbSets)
userSet := gontext.RegisterEntity[User](ctx)
fileSet := gontext.RegisterEntity[File](ctx)
```

## 🔍 Query Methods

### Basic Retrieval
```go
// Get first record or nil
user, err := ctx.Users.FirstOrDefault()

// Get first record or error if not found
user, err := ctx.Users.First()

// Get exactly one record (error if 0 or >1)
user, err := ctx.Users.Single()

// Get all records
users, err := ctx.Users.ToList()

// Find by primary key
user, err := ctx.Users.Find(userId)
```

### Where Conditions
```go
// Field-based filtering
users, err := ctx.Users.Where("Email", "john@example.com").ToList()

// SQL-style conditions
users, err := ctx.Users.Where("Age > ?", 18).ToList()

// Struct-based filtering (GORM-style)
users, err := ctx.Users.Where(&User{Role: "admin"}).ToList()

// Multiple conditions
users, err := ctx.Users.
    Where("IsActive", true).
    Where("Age > ?", 18).
    ToList()
```

### OR Conditions
```go
// Field-based OR
user, err := ctx.Users.
    Where("Email", email).
    OrField("Username", username).
    FirstOrDefault()

// Struct-based OR
users, err := ctx.Users.
    Where(&User{Role: "admin"}).
    OrEntity(User{Role: "manager"}).
    ToList()
```

### Field Comparison Helpers
```go
// IN queries
users, err := ctx.Users.WhereIn("Role", []string{"admin", "manager"}).ToList()

// LIKE queries
users, err := ctx.Users.WhereLike("Username", "%john%").ToList()

// NULL checks
users, err := ctx.Users.WhereNull("DeletedAt").ToList()
users, err := ctx.Users.WhereNotNull("Email").ToList()

// Between ranges
users, err := ctx.Users.WhereBetween("Age", 18, 65).ToList()
```

## 🔢 Aggregation Methods

### Counting
```go
// Count all records
total, err := ctx.Users.Count()

// Count with conditions
activeCount, err := ctx.Users.Where("IsActive", true).Count()

// Count non-null values in field
emailCount, err := ctx.Users.CountField("Email")

// Count distinct values
uniqueRoles, err := ctx.Users.CountDistinctField("Role")
```

### Numeric Aggregations
```go
// Sum of field values
totalSize, err := ctx.Files.SumField("Size")

// Average of field values
avgSize, err := ctx.Files.AverageField("Size")

// Minimum value
oldestFile, err := ctx.Files.MinField("CreatedAt")

// Maximum value
newestFile, err := ctx.Files.MaxField("UpdatedAt")
```

### Custom Aggregations with Scan
```go
// Complex custom queries
var stats struct {
    TotalFiles int64   `json:"total_files"`
    TotalSize  int64   `json:"total_size"`
    AvgSize    float64 `json:"avg_size"`
}

err := ctx.Files.Select(`
    COUNT(*) as total_files,
    COALESCE(SUM("Size"), 0) as total_size,
    COALESCE(AVG("Size"), 0) as avg_size
`).Scan(&stats)
```

## 📋 Field Selection

### Select Specific Fields
```go
// Load only specific fields
users, err := ctx.Users.Select("Id", "Username", "Email").ToList()

// Exclude sensitive fields
publicUsers, err := ctx.Users.Omit("PasswordHash").ToList()
```

## 🔗 Relationship Loading

### Include Related Data
```go
// Include single relationship
users, err := ctx.Users.Include("Posts").ToList()

// Include multiple relationships
users, err := ctx.Users.Include("Posts", "Profile").ToList()

// Auto-include all relationships
users, err := ctx.Users.IncludeAll().ToList()

// Combine with other operations
users, err := ctx.Users.
    Include("Posts").
    Where("IsActive", true).
    OrderBy("Username").
    ToList()
```

## 📊 Sorting and Pagination

### Ordering
```go
// Order by field ascending
users, err := ctx.Users.OrderByField("Username").ToList()

// Order by field descending
users, err := ctx.Users.OrderByFieldDescending("CreatedAt").ToList()

// Multiple ordering
users, err := ctx.Users.
    OrderByField("Role").
    ThenByField("Username").
    ToList()
```

### Pagination
```go
// Skip and take (offset and limit)
users, err := ctx.Users.
    OrderByField("Username").
    Skip(20).
    Take(10).
    ToList()

// Page-based pagination helper
func GetUserPage(page, pageSize int) ([]User, error) {
    return ctx.Users.
        OrderByField("Username").
        Skip((page - 1) * pageSize).
        Take(pageSize).
        ToList()
}
```

## 🔄 CRUD Operations

### Creating Records
```go
// Add single entity (EF Core style)
ctx.Users.Add(newUser)
err := ctx.SaveChanges()

// Create directly (GORM style)
err := ctx.Users.Create(&newUser)

// Add multiple entities
ctx.Users.AddRange([]User{user1, user2, user3})
err := ctx.SaveChanges()
```

### Updating Records
```go
// Update with change tracking
user.Email = "new@example.com"
ctx.Users.Update(user)
err := ctx.SaveChanges()

// Direct save (GORM style)
user.Email = "new@example.com"
err := ctx.Users.Save(&user)
```

### Deleting Records
```go
// Mark for deletion with change tracking
ctx.Users.Remove(user)
err := ctx.SaveChanges()

// Delete with conditions
err := ctx.Users.Where("IsActive", false).Delete()
```

## 🧪 Existence Checking

### Check if Records Exist
```go
// Check if any records match condition
hasAdmin, err := ctx.Users.Where("Role", "admin").Any()

// Check if specific user exists
userExists, err := ctx.Users.Where("Email", email).Any()
```

## 📝 Advanced Query Examples

### Complex Filtering
```go
// Multiple conditions with OR
adminUsers, err := ctx.Users.
    Where("Role", "admin").
    OrField("Role", "superadmin").
    Where("IsActive", true).
    ToList()

// Date range filtering
recentFiles, err := ctx.Files.
    WhereBetween("CreatedAt", startDate, endDate).
    OrderByFieldDescending("CreatedAt").
    ToList()

// Text search
searchResults, err := ctx.Users.
    WhereLike("Username", "%"+searchTerm+"%").
    OrField("Email", "%"+searchTerm+"%").
    ToList()
```

### Statistical Queries
```go
// User activity statistics
var userStats struct {
    TotalUsers   int64   `json:"total_users"`
    ActiveUsers  int64   `json:"active_users"`
    AvgAge      float64 `json:"avg_age"`
    NewestUser   string  `json:"newest_user"`
}

err := ctx.Users.Select(`
    COUNT(*) as total_users,
    COUNT(CASE WHEN "IsActive" THEN 1 END) as active_users,
    AVG("Age") as avg_age,
    MAX("Username") as newest_user
`).Scan(&userStats)
```

### Grouped Aggregations
```go
// Count users by role
var roleCounts []struct {
    Role  string `json:"role"`
    Count int64  `json:"count"`
}

err := ctx.Users.Select(`
    "Role" as role,
    COUNT(*) as count
`).GroupBy("Role").Scan(&roleCounts)
```

## ⚠️ Error Handling Patterns

### Always Handle Errors
```go
// ❌ WRONG - ignoring errors
user := ctx.Users.FirstOrDefault()

// ✅ CORRECT - proper error handling
user, err := ctx.Users.FirstOrDefault()
if err != nil {
    return fmt.Errorf("database error: %w", err)
}
if user == nil {
    return fmt.Errorf("user not found")
}
```

### Validation with Any()
```go
// Check if user exists before operations
exists, err := ctx.Users.Where("Email", email).Any()
if err != nil {
    return fmt.Errorf("failed to check user existence: %w", err)
}
if !exists {
    return fmt.Errorf("user with email %s not found", email)
}
```

## 🚫 Deprecated Patterns (Don't Use)

```go
// ❌ OLD DEPRECATED PATTERNS - DO NOT USE
ctx.GetDB().Model(&User{}).Where("email = ?", email).First(&user)
ctx.GetDB().Model(&User{}).Count(&count)
ctx.GetDB().Raw("SELECT COUNT(*) FROM users").Scan(&count)

// ✅ USE THESE INSTEAD
user, err := ctx.Users.Where("Email", email).FirstOrDefault()
count, err := ctx.Users.Count()
err := ctx.Users.Select("COUNT(*)").Scan(&count)
```

## 🎯 Best Practices

### 1. Always Handle Errors
```go
result, err := ctx.EntitySet.Method()
if err != nil {
    // Handle error appropriately
    return fmt.Errorf("operation failed: %w", err)
}
```

### 2. Use Typed Aggregations When Possible
```go
// ✅ PREFERRED - Type-safe
total, err := ctx.Files.SumField("Size")

// ✅ ACCEPTABLE - Custom queries
var total float64
err := ctx.Files.Select(`SUM("Size")`).Scan(&total)
```

### 3. Combine Operations Efficiently
```go
// Chain operations for better performance
results, err := ctx.Users.
    Include("Posts").
    Where("IsActive", true).
    OrderByField("Username").
    Skip(offset).
    Take(limit).
    ToList()
```

### 4. Use Appropriate Return Methods
- `FirstOrDefault()` - when record might not exist
- `First()` - when record must exist (error if not found)
- `Single()` - when exactly one record expected
- `ToList()` - for multiple records
- `Any()` - just to check existence
- `Count()` - just for counting
