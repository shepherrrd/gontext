package context

import (
	"fmt"
	"reflect"
	"log"

	"gorm.io/gorm"
	"github.com/shepherrrd/gontext/internal/models"
	"github.com/shepherrrd/gontext/internal/linq"
)

type DbSet struct {
	context     *DbContext
	entityType  reflect.Type
	entityModel *models.EntityModel
}

func NewDbSet(ctx *DbContext, entityType reflect.Type, entityModel *models.EntityModel) *DbSet {
	return &DbSet{
		context:     ctx,
		entityType:  entityType,
		entityModel: entityModel,
	}
}

func (ds *DbSet) Add(entity interface{}) {
	ds.context.changeTracker.Add(entity, EntityAdded)
}

func (ds *DbSet) Update(entity interface{}) {
	ds.context.changeTracker.Add(entity, EntityModified)
}

func (ds *DbSet) Remove(entity interface{}) {
	ds.context.changeTracker.Add(entity, EntityDeleted)
}

func (ds *DbSet) Find(dest interface{}, conditions ...interface{}) error {
	return ds.context.db.Find(dest, conditions...).Error
}

func (ds *DbSet) FirstEntity(dest interface{}, conditions ...interface{}) error {
	return ds.context.db.First(dest, conditions...).Error
}

func (ds *DbSet) Where(query interface{}, args ...interface{}) *gorm.DB {
	return ds.context.db.Model(reflect.New(ds.entityType).Interface()).Where(query, args...)
}

func (ds *DbSet) Create(value interface{}) error {
	return ds.context.db.Create(value).Error
}

func (ds *DbSet) Delete(value interface{}, conditions ...interface{}) error {
	return ds.context.db.Delete(value, conditions...).Error
}

func (ds *DbSet) Count(count *int64) error {
	return ds.context.db.Model(reflect.New(ds.entityType).Interface()).Count(count).Error
}

func (ds *DbSet) Preload(column string, conditions ...interface{}) *gorm.DB {
	return ds.context.db.Model(reflect.New(ds.entityType).Interface()).Preload(column, conditions...)
}

func (ds *DbSet) Raw(sql string, values ...interface{}) *gorm.DB {
	return ds.context.db.Raw(sql, values...)
}

func (ds *DbSet) GetEntityType() reflect.Type {
	return ds.entityType
}

func (ds *DbSet) GetEntityModel() *models.EntityModel {
	return ds.entityModel
}

// LINQ - returns a LINQ query builder for type-safe queries
func (ds *DbSet) LINQ() interface{} {
	// This needs to be called with proper type information
	// The actual implementation will be in the public API
	return linq.NewLinqQuery[interface{}](ds.context.db)
}

// FirstOrDefault - EF Core style method with predicate support
func (ds *DbSet) FirstOrDefault(conditions ...interface{}) (interface{}, error) {
	log.Printf("[GONTEXT DEBUG] DbSet.FirstOrDefault called for entity type: %s", ds.entityType.Name())
	
	var result interface{}
	query := ds.context.db.Model(reflect.New(ds.entityType).Interface())
	
	if len(conditions) > 0 {
		log.Printf("[GONTEXT DEBUG] Adding conditions: %+v", conditions)
		query = query.Where(conditions[0], conditions[1:]...)
	}
	
	// Log the SQL query
	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Limit(1).First(&result)
	})
	log.Printf("[GONTEXT DEBUG] Generated SQL: %s", sql)
	
	// Log any existing clauses
	if len(query.Statement.Clauses) > 0 {
		log.Printf("[GONTEXT DEBUG] Query has %d clauses", len(query.Statement.Clauses))
		for name, clause := range query.Statement.Clauses {
			log.Printf("[GONTEXT DEBUG] Clause: %s = %+v", name, clause)
		}
	}
	
	log.Printf("[GONTEXT DEBUG] Executing First() query...")
	err := query.First(&result).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Printf("[GONTEXT DEBUG] No record found, returning nil")
			return nil, nil
		}
		log.Printf("[GONTEXT DEBUG] Error occurred: %v", err)
		return nil, err
	}
	
	log.Printf("[GONTEXT DEBUG] Record found: %+v", result)
	
	// Automatically track the loaded entity for change detection
	ds.context.changeTracker.TrackLoaded(result)
	
	return result, nil
}

// First - EF Core style method
func (ds *DbSet) First(conditions ...interface{}) (interface{}, error) {
	var result interface{}
	query := ds.context.db.Model(reflect.New(ds.entityType).Interface())
	
	if len(conditions) > 0 {
		query = query.Where(conditions[0], conditions[1:]...)
	}
	
	err := query.First(&result).Error
	if err == nil {
		// Automatically track the loaded entity for change detection
		ds.context.changeTracker.TrackLoaded(result)
	}
	return result, err
}

// Single - EF Core style method
func (ds *DbSet) Single(conditions ...interface{}) (interface{}, error) {
	query := ds.context.db.Model(reflect.New(ds.entityType).Interface())
	
	if len(conditions) > 0 {
		query = query.Where(conditions[0], conditions[1:]...)
	}
	
	var results []interface{}
	err := query.Limit(2).Find(&results).Error
	if err != nil {
		return nil, err
	}
	
	if len(results) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(results) > 1 {
		return nil, fmt.Errorf("sequence contains more than one element")
	}
	
	result := results[0]
	// Automatically track the loaded entity for change detection
	ds.context.changeTracker.TrackLoaded(result)
	
	return result, nil
}

// Any - EF Core style method
func (ds *DbSet) Any(conditions ...interface{}) (bool, error) {
	query := ds.context.db.Model(reflect.New(ds.entityType).Interface())
	
	if len(conditions) > 0 {
		query = query.Where(conditions[0], conditions[1:]...)
	}
	
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}