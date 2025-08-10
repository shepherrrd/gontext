package query

import (
	"regexp"
	"strings"
)

// PostgreSQLQueryTranslator handles automatic translation of field names to quoted PostgreSQL identifiers
type PostgreSQLQueryTranslator struct {
	entityFieldMap map[string][]string // entityType -> field names
}

// NewPostgreSQLQueryTranslator creates a new translator
func NewPostgreSQLQueryTranslator() *PostgreSQLQueryTranslator {
	return &PostgreSQLQueryTranslator{
		entityFieldMap: make(map[string][]string),
	}
}

// RegisterEntityFields registers field names for an entity type
func (t *PostgreSQLQueryTranslator) RegisterEntityFields(entityName string, fieldNames []string) {
	t.entityFieldMap[entityName] = fieldNames
}

// TranslateQuery translates a WHERE condition to use proper PostgreSQL quoted identifiers
func (t *PostgreSQLQueryTranslator) TranslateQuery(entityName, condition string) string {
	if fieldNames, exists := t.entityFieldMap[entityName]; exists {
		return t.translateCondition(condition, fieldNames)
	}
	return condition
}

// translateCondition translates field names in a condition to quoted identifiers
func (t *PostgreSQLQueryTranslator) translateCondition(condition string, fieldNames []string) string {
	result := condition
	
	// Sort field names by length (descending) to match longer names first
	// This prevents issues where "Name" might match part of "Username"
	sortedFields := make([]string, len(fieldNames))
	copy(sortedFields, fieldNames)
	for i := 0; i < len(sortedFields)-1; i++ {
		for j := i + 1; j < len(sortedFields); j++ {
			if len(sortedFields[i]) < len(sortedFields[j]) {
				sortedFields[i], sortedFields[j] = sortedFields[j], sortedFields[i]
			}
		}
	}
	
	for _, fieldName := range sortedFields {
		// Skip if already quoted
		if strings.Contains(result, "\""+fieldName+"\"") {
			continue
		}
		
		// Pattern to match field names in various SQL contexts
		patterns := []string{
			// Basic comparisons with flexible spacing: fieldName = ? or fieldName= ?
			`\b` + regexp.QuoteMeta(fieldName) + `\s*(=|!=|<>|<|>|<=|>=)\s*`,
			// LIKE/ILIKE with flexible spacing: fieldName LIKE ? or fieldName LIKE ?
			`\b` + regexp.QuoteMeta(fieldName) + `\s+(LIKE|ILIKE)\s+`,
			// IN/NOT IN with flexible spacing: fieldName IN ? or fieldName NOT IN ?
			`\b` + regexp.QuoteMeta(fieldName) + `\s+(IN|NOT\s+IN)\s+`,
			// IS NULL/IS NOT NULL: fieldName IS NULL
			`\b` + regexp.QuoteMeta(fieldName) + `(\s+IS\s+(NOT\s+)?NULL)`,
			// BETWEEN: fieldName BETWEEN
			`\b` + regexp.QuoteMeta(fieldName) + `(\s+BETWEEN\s)`,
			// ORDER BY: ORDER BY fieldName
			`(ORDER\s+BY\s+)` + regexp.QuoteMeta(fieldName) + `(\s|$)`,
			// GROUP BY: GROUP BY fieldName
			`(GROUP\s+BY\s+)` + regexp.QuoteMeta(fieldName) + `(\s|$)`,
			// SELECT: SELECT fieldName
			`(SELECT\s+)` + regexp.QuoteMeta(fieldName) + `(\s|,|$)`,
			// Functions: COUNT(fieldName), SUM(fieldName), etc.
			`(COUNT\s*\(\s*)` + regexp.QuoteMeta(fieldName) + `(\s*\))`,
			`(SUM\s*\(\s*)` + regexp.QuoteMeta(fieldName) + `(\s*\))`,
			`(AVG\s*\(\s*)` + regexp.QuoteMeta(fieldName) + `(\s*\))`,
			`(MIN\s*\(\s*)` + regexp.QuoteMeta(fieldName) + `(\s*\))`,
			`(MAX\s*\(\s*)` + regexp.QuoteMeta(fieldName) + `(\s*\))`,
		}
		
		for _, pattern := range patterns {
			re := regexp.MustCompile(`(?i)` + pattern)
			result = re.ReplaceAllStringFunc(result, func(match string) string {
				return strings.ReplaceAll(match, fieldName, `"`+fieldName+`"`)
			})
		}
	}
	
	return result
}

// TranslateComplexQuery handles complex WHERE queries with AND, OR, parentheses
func (t *PostgreSQLQueryTranslator) TranslateComplexQuery(entityName, condition string) string {
	if fieldNames, exists := t.entityFieldMap[entityName]; exists {
		return t.translateComplexCondition(condition, fieldNames)
	}
	return condition
}

// translateComplexCondition handles complex queries with logical operators
func (t *PostgreSQLQueryTranslator) translateComplexCondition(condition string, fieldNames []string) string {
	// First, handle simple field references
	result := t.translateCondition(condition, fieldNames)
	
	// Handle complex cases with AND/OR/parentheses
	// Split by logical operators while preserving them
	parts := t.splitPreservingDelimiters(result, []string{" AND ", " OR ", "(", ")"})
	
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && part != "AND" && part != "OR" && part != "(" && part != ")" {
			parts[i] = t.translateCondition(part, fieldNames)
		}
	}
	
	return strings.Join(parts, "")
}

// splitPreservingDelimiters splits a string by delimiters while keeping the delimiters
func (t *PostgreSQLQueryTranslator) splitPreservingDelimiters(text string, delimiters []string) []string {
	if len(delimiters) == 0 {
		return []string{text}
	}
	
	result := []string{text}
	
	for _, delimiter := range delimiters {
		var newResult []string
		for _, part := range result {
			if strings.Contains(part, delimiter) {
				split := strings.Split(part, delimiter)
				for i, s := range split {
					if i > 0 {
						newResult = append(newResult, delimiter)
					}
					if s != "" {
						newResult = append(newResult, s)
					}
				}
			} else {
				newResult = append(newResult, part)
			}
		}
		result = newResult
	}
	
	return result
}

// GetQuotedFieldName returns a field name with PostgreSQL quotes
func (t *PostgreSQLQueryTranslator) GetQuotedFieldName(fieldName string) string {
	return `"` + fieldName + `"`
}