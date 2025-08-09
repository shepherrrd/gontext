package discovery

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/shepherrrd/gontext"
)

// DesignTimeContextFinder looks for design-time context functions
type DesignTimeContextFinder struct {
	projectRoot string
}

// NewDesignTimeContextFinder creates a new design-time context finder
func NewDesignTimeContextFinder(projectRoot string) *DesignTimeContextFinder {
	return &DesignTimeContextFinder{projectRoot: projectRoot}
}

// FindDesignTimeContext looks for CreateDesignTimeContext function
func (dtf *DesignTimeContextFinder) FindDesignTimeContext() (string, error) {
	var designTimeFile string

	// Look for files with CreateDesignTimeContext function
	err := filepath.Walk(dtf.projectRoot, func(path string, info os.FileInfo, err error) error {
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
			return nil // Skip files with parse errors
		}

		// Look for CreateDesignTimeContext function
		if dtf.hasCreateDesignTimeContext(node) {
			designTimeFile = path
			return filepath.SkipDir // Found it, stop searching
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to scan project: %w", err)
	}

	if designTimeFile == "" {
		return "", fmt.Errorf("CreateDesignTimeContext function not found")
	}

	return designTimeFile, nil
}

// hasCreateDesignTimeContext checks if a file has CreateDesignTimeContext function
func (dtf *DesignTimeContextFinder) hasCreateDesignTimeContext(file *ast.File) bool {
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == "CreateDesignTimeContext" {
				// Check return type is (*gontext.DbContext, error)
				if dtf.hasCorrectSignature(funcDecl) {
					return true
				}
			}
		}
	}
	return false
}

// hasCorrectSignature checks if CreateDesignTimeContext has correct signature
func (dtf *DesignTimeContextFinder) hasCorrectSignature(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 2 {
		return false
	}

	// Should return (*gontext.DbContext, error)
	results := funcDecl.Type.Results.List

	// First return type should be *gontext.DbContext
	if starExpr, ok := results[0].Type.(*ast.StarExpr); ok {
		if selectorExpr, ok := starExpr.X.(*ast.SelectorExpr); ok {
			if ident, ok := selectorExpr.X.(*ast.Ident); ok {
				if ident.Name == "gontext" && selectorExpr.Sel.Name == "DbContext" {
					// Second return type should be error
					if ident, ok := results[1].Type.(*ast.Ident); ok {
						return ident.Name == "error"
					}
				}
			}
		}
	}

	return false
}

// CreateContextFromDesignTimeFactory creates a context using the discovered design-time factory
// This requires dynamic loading which is complex in Go, so we'll provide instructions instead
func (dtf *DesignTimeContextFinder) CreateContextFromDesignTimeFactory() (*gontext.DbContext, error) {
	designTimeFile, err := dtf.FindDesignTimeContext()
	if err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("found CreateDesignTimeContext in %s - please run: go run . --design-time-context", designTimeFile)
}