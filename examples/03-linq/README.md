# GoNtext LINQ Queries Example

This example demonstrates the full power of GoNtext's LINQ-style query API, showing all the ways you can query data with familiar EF Core patterns.

## 🚀 Quick Start

### Prerequisites
- PostgreSQL running on localhost:5432
- Database named `test_linq` created

```bash
createdb test_linq
```

### Run the Example
```bash
go mod tidy
go run .
```

## 📖 What You'll Learn

### 1. Basic Queries

```go
// Get all users (EF Core: context.Users.ToList())
allUsers, _ := ctx.Users.ToList()

// Count users (EF Core: context.Users.Count())
userCount, _ := ctx.Users.Count()

// First user (EF Core: context.Users.First())
firstUser, _ := ctx.Users.OrderByField("username").FirstOrDefault()

// Find by ID (EF Core: context.Users.Find(id))
userByID, _ := ctx.Users.ById(userID)
```

### 2. Filtering with WHERE

```go
// Simple equality (EF Core: x => x.IsActive)
activeUsers, _ := ctx.Users.WhereField("is_active", true).ToList()

// Comparisons (EF Core: x => x.Age >= 30)
olderUsers, _ := ctx.Users.WhereField("age", ">=30").ToList()

// Range queries (EF Core: x => x.Age >= 25 && x.Age <= 35)
ageRangeUsers, _ := ctx.Users.WhereFieldBetween("age", 25, 35).ToList()

// Multiple conditions
youngActiveUsers, _ := ctx.Users.
    WhereField("is_active", true).
    WhereField("age", "<30").
    ToList()
```

### 3. String Operations

```go
// Contains (EF Core: x => x.Email.Contains("gmail"))
gmailUsers, _ := ctx.Users.WhereFieldLike("email", "gmail").ToList()

// Starts with (EF Core: x => x.FirstName.StartsWith("A"))
aNameUsers, _ := ctx.Users.WhereFieldStartsWith("first_name", "A").ToList()

// Ends with (EF Core: x => x.Email.EndsWith(".com"))
comUsers, _ := ctx.Users.WhereFieldEndsWith("email", ".com").ToList()

// Exact match
alice, _ := ctx.Users.WhereField("username", "alice").FirstOrDefault()
```

### 4. Ordering

```go
// Ascending (EF Core: context.Users.OrderBy(x => x.Username))
usersByName, _ := ctx.Users.OrderByField("username").ToList()

// Descending (EF Core: context.Users.OrderByDescending(x => x.Age))
usersByAge, _ := ctx.Users.OrderByFieldDescending("age").ToList()

// Multiple sorting
sortedUsers, _ := ctx.Users.
    OrderByField("age").
    OrderByField("username").
    ToList()
```

### 5. Pagination

```go
// Take first N (EF Core: context.Users.Take(3))
firstThree, _ := ctx.Users.OrderByField("username").Take(3).ToList()

// Skip and Take (EF Core: context.Users.Skip(10).Take(5))
page2, _ := ctx.Users.OrderByField("username").Skip(10).Take(5).ToList()

// Top N with conditions
topPosts, _ := ctx.Posts.
    WhereField("is_published", true).
    OrderByFieldDescending("view_count").
    Take(10).
    ToList()
```

### 6. Aggregations

```go
// Count with conditions (EF Core: context.Users.Count(x => x.IsActive))
activeCount, _ := ctx.Users.WhereField("is_active", true).Count()

// Count related data
userPostCount, _ := ctx.Posts.WhereField("author_id", userID).Count()

// Existence checks (EF Core: context.Users.Any(x => x.Age > 40))
hasOldUsers, _ := ctx.Users.WhereField("age", ">40").Any()
```

### 7. Method Chaining (The Power of LINQ)

```go
// Complex query with method chaining
results, _ := ctx.Users.
    WhereField("is_active", true).           // Filter active users
    WhereField("age", ">=25").               // Age 25 or older
    WhereFieldLike("email", "gmail").        // Gmail users only
    OrderByFieldDescending("salary").        // Highest paid first
    Skip(5).                                 // Skip first 5
    Take(10).                                // Take next 10
    ToList()                                 // Execute query

// Popular content query
popularContent, _ := ctx.Posts.
    WhereField("is_published", true).        // Published posts only
    WhereField("view_count", ">100").        // High view count
    OrderByFieldDescending("view_count").    // Most popular first
    Take(5).                                 // Top 5
    ToList()
```

## 📊 Complete Method Reference

### Basic Operations
| Method | Purpose | EF Core Equivalent |
|--------|---------|-------------------|
| `ToList()` | Get all results | `ToList()` |
| `Count()` | Count records | `Count()` |
| `Any()` | Check if any exist | `Any()` |
| `FirstOrDefault()` | Get first or nil | `FirstOrDefault()` |
| `ById(id)` | Find by primary key | `Find(id)` |

### Filtering
| Method | Purpose | EF Core Equivalent |
|--------|---------|-------------------|
| `WhereField("field", value)` | Exact match | `Where(x => x.Field == value)` |
| `WhereField("field", ">10")` | Comparison | `Where(x => x.Field > 10)` |
| `WhereFieldBetween("field", 1, 10)` | Range | `Where(x => x.Field >= 1 && x.Field <= 10)` |
| `WhereFieldLike("field", "text")` | Contains | `Where(x => x.Field.Contains("text"))` |
| `WhereFieldStartsWith("field", "A")` | Starts with | `Where(x => x.Field.StartsWith("A"))` |
| `WhereFieldEndsWith("field", ".com")` | Ends with | `Where(x => x.Field.EndsWith(".com"))` |

### Ordering & Pagination
| Method | Purpose | EF Core Equivalent |
|--------|---------|-------------------|
| `OrderByField("field")` | Sort ascending | `OrderBy(x => x.Field)` |
| `OrderByFieldDescending("field")` | Sort descending | `OrderByDescending(x => x.Field)` |
| `Take(n)` | Limit results | `Take(n)` |
| `Skip(n)` | Skip results | `Skip(n)` |

### Comparison Operators
| Operator | Example | Purpose |
|----------|---------|---------|
| `"=value"` or `value` | `WhereField("age", 25)` | Exact match |
| `">value"` | `WhereField("age", ">25")` | Greater than |
| `">=value"` | `WhereField("age", ">=25")` | Greater or equal |
| `"<value"` | `WhereField("age", "<25")` | Less than |
| `"<=value"` | `WhereField("age", "<=25")` | Less or equal |
| `"<>value"` | `WhereField("age", "<>25")` | Not equal |

## 🎯 Example Output

```bash
🚀 GoNtext LINQ Queries Example
===============================
📋 Setting up database and sample data...
✅ Created 5 users and 5 posts

📖 Basic Queries
----------------
All users: 5
User count: 5
First user: alice
Found by ID: alice

🔍 Filtering (WHERE Clauses)
----------------------------
Active users: 4
Users 30 or older: 2
Users aged 25-35: 3
Young active users: 3

🔤 String Operations
-------------------
Gmail users: 2
Names starting with A: 1
Email ending with .com: 2
Found specific user: Alice Smith

📊 Ordering
-----------
Users by username: alice, bob, charlie...
Users by age (oldest first): 35, 30, 28...
Top 3 posts by views: 567, 234, 150 views

📄 Pagination
-------------
First 3 users: 3
Page 2 (skip 2, take 2): 2 users
Top 3 published posts: 3 found

🔢 Aggregations
---------------
Active users count: 4
Published posts: 4
alice has 3 posts

🧠 Complex Queries
------------------
Complex query result: 1 users
Popular content (>100 views): 3 posts
Premium young users: 2

✅ Existence Checks
-------------------
Has any users: true
Has users over 40: false
Has Gmail users: true
Has draft posts: true
Has posts with >200 views: true

✅ LINQ demo completed!
```

## 💡 Best Practices

### 1. Method Chaining
```go
// Good: Chain methods for complex queries
users, _ := ctx.Users.
    WhereField("is_active", true).
    WhereField("age", ">25").
    OrderByField("username").
    Take(10).
    ToList()

// Avoid: Multiple separate queries
activeUsers, _ := ctx.Users.WhereField("is_active", true).ToList()
// Then filtering in Go code - inefficient!
```

### 2. Use Appropriate Methods
```go
// Check existence - use Any() (efficient)
hasUsers, _ := ctx.Users.Any()

// Don't use Count() > 0 (less efficient)
count, _ := ctx.Users.Count()
hasUsers := count > 0
```

### 3. Pagination Best Practices
```go
// Always use OrderBy with Skip/Take for consistent results
page1, _ := ctx.Users.
    OrderByField("id").  // Consistent ordering
    Skip(0).
    Take(10).
    ToList()

page2, _ := ctx.Users.
    OrderByField("id").  // Same ordering
    Skip(10).
    Take(10).
    ToList()
```

## 🎮 Try It Yourself

Modify the example to try:
1. **Custom filtering**: Add your own WHERE conditions
2. **Complex sorting**: Try multiple OrderBy clauses
3. **Advanced pagination**: Implement page-based navigation
4. **Custom aggregations**: Count specific conditions
5. **String patterns**: Try different LIKE patterns

## 🔗 Next Steps

- [CRUD Operations](../01-crud/) - Basic database operations
- [Migrations Example](../02-migrations/) - Schema management

This example shows the full power of GoNtext's LINQ implementation!