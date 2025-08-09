package discovery

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// DbContextInfo holds information about a discovered DbContext
type DbContextInfo struct {
	Name        string
	PackageName string
	FilePath    string
	Entities    []EntityInfo
}

// EntityInfo holds information about an entity discovered in a DbContext
type EntityInfo struct {
	Name     string
	TypeName string
	Package  string
}

// ContextScanner scans Go source files to find DbContext structs and their entities
type ContextScanner struct {
	projectRoot string
}

// NewContextScanner creates a new context scanner
func NewContextScanner(projectRoot string) *ContextScanner {
	return &ContextScanner{projectRoot: projectRoot}
}

// ScanForContexts scans the project for GoNtext DbContext structs
func (cs *ContextScanner) ScanForContexts() ([]DbContextInfo, error) {
	var contexts []DbContextInfo

	// Walk through all .go files in the project
	err := filepath.Walk(cs.projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and vendor/test directories
		if !strings.HasSuffix(path, ".go") || 
		   strings.Contains(path, "vendor/") || 
		   strings.Contains(path, "_test.go") {
			return nil
		}

		// Parse the Go file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err // Skip files with parse errors
		}

		// Look for structs that embed gontext.DbContext
		contexts = append(contexts, cs.findContextsInFile(node, path)...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan project: %w", err)
	}

	return contexts, nil
}

// findContextsInFile finds DbContext structs in a single file
func (cs *ContextScanner) findContextsInFile(file *ast.File, filePath string) []DbContextInfo {
	var contexts []DbContextInfo

	ast.Inspect(file, func(n ast.Node) bool {
		// Look for struct type declarations
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				// Check if this struct embeds gontext.DbContext
				if cs.isDbContext(structType) {
					context := DbContextInfo{
						Name:        typeSpec.Name.Name,
						PackageName: file.Name.Name,
						FilePath:    filePath,
						Entities:    cs.extractEntitiesFromStruct(structType),
					}
					contexts = append(contexts, context)
				}
			}
		}
		return true
	})

	return contexts
}

// isDbContext checks if a struct embeds gontext.DbContext
func (cs *ContextScanner) isDbContext(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		// Check for embedded gontext.DbContext
		if len(field.Names) == 0 { // Embedded field
			if starExpr, ok := field.Type.(*ast.StarExpr); ok {
				if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
					if ident, ok := selectorExpr.X.(*ast.Ident); ok {
						if ident.Name == "gontext" && selectorExpr.Sel.Name == "DbContext" {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// extractEntitiesFromStruct extracts entity types from LinqDbSet fields
func (cs *ContextScanner) extractEntitiesFromStruct(structType *ast.StructType) []EntityInfo {
	var entities []EntityInfo

	for _, field := range structType.Fields.List {
		// Look for fields of type *gontext.LinqDbSet[EntityType]
		if len(field.Names) > 0 { // Named field (not embedded)
			entityType := cs.extractEntityFromLinqDbSet(field.Type)
			if entityType != "" {
				entities = append(entities, EntityInfo{
					Name:     field.Names[0].Name,
					TypeName: entityType,
					Package:  "", // We'll need to resolve this
				})
			}
		}
	}

	return entities
}

// extractEntityFromLinqDbSet extracts the entity type from *gontext.LinqDbSet[EntityType]
func (cs *ContextScanner) extractEntityFromLinqDbSet(fieldType ast.Expr) string {
	// Handle *gontext.LinqDbSet[EntityType]
	if starExpr, ok := fieldType.(*ast.StarExpr); ok {
		if indexExpr, ok := starExpr.X.(*ast.IndexExpr); ok {
			// Check if it's gontext.LinqDbSet
			if selectorExpr, ok := indexExpr.X.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					if ident.Name == "gontext" && selectorExpr.Sel.Name == "LinqDbSet" {
						// Extract the generic type parameter
						if entityIdent, ok := indexExpr.Index.(*ast.Ident); ok {
							return entityIdent.Name
						}
					}
				}
			}
		}
	}
	return ""
}

// FindDefaultContext finds the first DbContext in the project
func (cs *ContextScanner) FindDefaultContext() (*DbContextInfo, error) {
	contexts, err := cs.ScanForContexts()
	if err != nil {
		return nil, err
	}

	if len(contexts) == 0 {
		return nil, fmt.Errorf("no DbContext found in project")
	}

	// Return the first context found
	return &contexts[0], nil
}

// FindContextByName finds a specific DbContext by name
func (cs *ContextScanner) FindContextByName(name string) (*DbContextInfo, error) {
	contexts, err := cs.ScanForContexts()
	if err != nil {
		return nil, err
	}

	for _, ctx := range contexts {
		if ctx.Name == name {
			return &ctx, nil
		}
	}

	return nil, fmt.Errorf("DbContext '%s' not found", name)
}