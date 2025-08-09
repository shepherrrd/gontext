package discovery

import (
	"fmt"
	"go/build"
	"reflect"

	"github.com/shepherrrd/gontext"
)

// EntityLoader loads and registers entities from a discovered DbContext
type EntityLoader struct {
	contextInfo *DbContextInfo
	projectRoot string
}

// NewEntityLoader creates a new entity loader
func NewEntityLoader(contextInfo *DbContextInfo, projectRoot string) *EntityLoader {
	return &EntityLoader{
		contextInfo: contextInfo,
		projectRoot: projectRoot,
	}
}

// LoadEntitiesIntoContext loads all entities from the DbContext into a gontext.DbContext
func (el *EntityLoader) LoadEntitiesIntoContext(ctx *gontext.DbContext) error {
	// Try to load the package and get entity types
	pkg, err := el.loadPackage()
	if err != nil {
		return fmt.Errorf("failed to load package: %w", err)
	}

	// Register each entity found in the DbContext
	for _, entityInfo := range el.contextInfo.Entities {
		entityType, err := el.getEntityType(pkg, entityInfo.TypeName)
		if err != nil {
			return fmt.Errorf("failed to get entity type %s: %w", entityInfo.TypeName, err)
		}

		// Register the entity with the context
		ctx.RegisterEntity(reflect.New(entityType).Interface())
	}

	return nil
}

// loadPackage attempts to load the Go package containing the entities
func (el *EntityLoader) loadPackage() (*build.Package, error) {
	// Use go/build to load package information
	pkg, err := build.ImportDir(el.projectRoot, build.ImportComment)
	if err != nil {
		return nil, fmt.Errorf("failed to import package: %w", err)
	}
	return pkg, nil
}

// getEntityType gets the reflect.Type for an entity by name
func (el *EntityLoader) getEntityType(pkg *build.Package, entityName string) (reflect.Type, error) {
	// This is tricky in Go - we need to use reflection or code generation
	// For now, we'll implement a registry pattern where entities self-register
	
	// Check if the entity type is registered in a global registry
	if entityType := GetRegisteredEntityType(entityName); entityType != nil {
		return entityType, nil
	}

	return nil, fmt.Errorf("entity type %s not found in registry", entityName)
}

// Global entity registry (simple implementation)
var entityRegistry = make(map[string]reflect.Type)

// RegisterEntityType registers an entity type globally
func RegisterEntityType[T any]() {
	var zero T
	entityType := reflect.TypeOf(zero)
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}
	entityRegistry[entityType.Name()] = entityType
}

// GetRegisteredEntityType gets a registered entity type by name
func GetRegisteredEntityType(name string) reflect.Type {
	return entityRegistry[name]
}

// InitializeEntityRegistry should be called by projects to register their entities
func InitializeEntityRegistry() {
	// This would be called by the user's project to register all entity types
	// Example usage in user code:
	// discovery.RegisterEntityType[User]()
	// discovery.RegisterEntityType[Post]()
}