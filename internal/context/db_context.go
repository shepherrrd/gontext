package context

import (
	"fmt"
	"log"
	"reflect"
	"sync"

	"gorm.io/gorm"
	"github.com/shepherrrd/gontext/internal/drivers"
	"github.com/shepherrrd/gontext/internal/models"
	"github.com/shepherrrd/gontext/internal/query"
)

// typeKey converts a reflect.Type to a string key for map storage
func typeKey(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}

type DbContext struct {
	db            *gorm.DB
	driver        drivers.DatabaseDriver
	entities      map[string]*models.EntityModel  // Use string keys instead of reflect.Type
	entityTypes   map[string]reflect.Type         // Map to store the actual reflect.Type for each key
	dbSets        map[string]interface{}          // Use string keys instead of reflect.Type  
	mu            sync.RWMutex
	changeTracker *ChangeTracker
	pgPlugin      *query.PostgreSQLPlugin
}

type DbContextOptions struct {
	ConnectionString string
	Driver          drivers.DatabaseDriver
	LogLevel        int
}

func NewDbContext(options DbContextOptions) (*DbContext, error) {
	db, err := options.Driver.Connect(options.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	ctx := &DbContext{
		db:            db,
		driver:        options.Driver,
		entities:      make(map[string]*models.EntityModel),
		entityTypes:   make(map[string]reflect.Type),
		dbSets:        make(map[string]interface{}),
		changeTracker: NewChangeTracker(),
	}
	
	// Check if this is PostgreSQL - we'll get the plugin differently
	if options.Driver.Name() == "postgres" {
		// For now, we'll store a reference to check later
		// The actual plugin registration happens in the driver
	}

	return ctx, nil
}

func (ctx *DbContext) RegisterEntity(entity interface{}) *DbSet {
	entityType := reflect.TypeOf(entity)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	key := typeKey(entityType)

	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if _, exists := ctx.entities[key]; exists {
		return ctx.dbSets[key].(*DbSet)
	}

	entityModel := models.NewEntityModel(entityType)
	ctx.entities[key] = entityModel
	ctx.entityTypes[key] = entityType  // Store the reflect.Type for later retrieval

	dbSet := NewDbSet(ctx, entityType, entityModel)
	ctx.dbSets[key] = dbSet

	return dbSet
}

func (ctx *DbContext) GetDbSet(entityType reflect.Type) *DbSet {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	key := typeKey(entityType)
	if dbSet, exists := ctx.dbSets[key]; exists {
		return dbSet.(*DbSet)
	}

	return nil
}

func (ctx *DbContext) SaveChanges() error {
	return ctx.db.Transaction(func(tx *gorm.DB) error {
		for _, changes := range ctx.changeTracker.GetChanges() {
			entity := changes.Entity
			
			// Ensure we have a pointer for GORM operations
			entityValue := reflect.ValueOf(entity)
			if entityValue.Kind() != reflect.Ptr {
				// Create a pointer to the entity
				entityPtr := reflect.New(entityValue.Type())
				entityPtr.Elem().Set(entityValue)
				entity = entityPtr.Interface()
			}
			
			switch changes.State {
			case EntityAdded:
				if err := tx.Create(entity).Error; err != nil {
					return err
				}
			case EntityModified:
				if err := tx.Save(entity).Error; err != nil {
					return err
				}
			case EntityDeleted:
				if err := tx.Delete(entity).Error; err != nil {
					return err
				}
			}
		}
		ctx.changeTracker.Clear()
		return nil
	})
}

func (ctx *DbContext) BeginTransaction() *gorm.DB {
	return ctx.db.Begin()
}

// GetDB returns the underlying GORM database instance
// DEPRECATED: Use LINQ methods instead of GetDB().Model() patterns
// OLD DEPRECATED PATTERN: ctx.GetDB().Model(&Entity{}).Select("SUM(field)").Scan(&result)
// NEW CORRECT PATTERN: result, err := ctx.EntitySet.SumField("field")
// Only use GetDB() for operations not yet supported by LINQ methods
func (ctx *DbContext) GetDB() *gorm.DB {
	return ctx.db
}

func (ctx *DbContext) GetDriver() drivers.DatabaseDriver {
	return ctx.driver
}

func (ctx *DbContext) GetEntityModels() map[string]*models.EntityModel {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	result := make(map[string]*models.EntityModel)
	for key, entityModel := range ctx.entities {
		result[key] = entityModel
	}
	return result
}

func (ctx *DbContext) Close() error {
	sqlDB, err := ctx.driver.GetSQLDB(ctx.db)
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (ctx *DbContext) EnsureCreated() error {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	for _, entity := range ctx.entities {
		if err := ctx.db.AutoMigrate(reflect.New(entity.Type).Interface()); err != nil {
			log.Printf("Warning: AutoMigrate failed for %s: %v", entity.Name, err)
		}
	}
	return nil
}

// AddEntity adds an entity to the change tracker
func (ctx *DbContext) AddEntity(entity interface{}) {
	ctx.changeTracker.Add(entity, EntityAdded)
}

// UpdateEntity marks an entity as modified
func (ctx *DbContext) UpdateEntity(entity interface{}) {
	ctx.changeTracker.Add(entity, EntityModified)
}

// RemoveEntity marks an entity for deletion
func (ctx *DbContext) RemoveEntity(entity interface{}) {
	ctx.changeTracker.Add(entity, EntityDeleted)
}