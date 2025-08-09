package models

import (
	"reflect"
	"strings"
)

type EntityModel struct {
	Name       string
	TableName  string
	Type       reflect.Type
	Fields     map[string]FieldModel
	PrimaryKey []string
}

type FieldModel struct {
	Name         string
	ColumnName   string
	Type         string
	GoType       reflect.Type
	Tags         map[string]string
	IsPrimary    bool
	IsNullable   bool
	IsUnique     bool
	DefaultValue *string
	OldName      *string // For column renames
}

func NewEntityModel(entityType reflect.Type) *EntityModel {
	if entityType.Kind() == reflect.Ptr {
		entityType = entityType.Elem()
	}

	entity := &EntityModel{
		Name:      entityType.Name(),
		TableName: entityType.Name(), 
		Type:      entityType,
		Fields:    make(map[string]FieldModel),
	}

	for i := 0; i < entityType.NumField(); i++ {
		field := entityType.Field(i)
		if field.PkgPath != "" {
			continue
		}

		fieldModel := parseFieldModel(field)
		entity.Fields[field.Name] = fieldModel

		if fieldModel.IsPrimary {
			entity.PrimaryKey = append(entity.PrimaryKey, fieldModel.ColumnName)
		}
	}

	return entity
}

func parseFieldModel(field reflect.StructField) FieldModel {
	fieldModel := FieldModel{
		Name:       field.Name,
		ColumnName: field.Name,
		Type:       field.Type.String(),
		GoType:     field.Type,
		Tags:       make(map[string]string),
		IsNullable: isNullableType(field.Type),
	}

	gonTextTag := field.Tag.Get("gontext")
	if gonTextTag != "" {
		parseTags(gonTextTag, fieldModel.Tags)
	}

	gormTag := field.Tag.Get("gorm")
	if gormTag != "" {
		parseTags(gormTag, fieldModel.Tags)
	}

	if _, exists := fieldModel.Tags["primary_key"]; exists || strings.Contains(gonTextTag, "primary_key") {
		fieldModel.IsPrimary = true
		fieldModel.IsNullable = false
	}

	if _, exists := fieldModel.Tags["unique"]; exists {
		fieldModel.IsUnique = true
	}

	if _, exists := fieldModel.Tags["not_null"]; exists {
		fieldModel.IsNullable = false
	}

	if defaultVal, exists := fieldModel.Tags["default"]; exists {
		fieldModel.DefaultValue = &defaultVal
	}

	if oldName, exists := fieldModel.Tags["old_name"]; exists {
		fieldModel.OldName = &oldName
	}

	return fieldModel
}

func parseTags(tagStr string, tags map[string]string) {
	parts := strings.Split(tagStr, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			tags[kv[0]] = kv[1]
		} else {
			tags[part] = ""
		}
	}
}

func isNullableType(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr || 
		   t.Kind() == reflect.Interface ||
		   t.Kind() == reflect.Slice ||
		   t.Kind() == reflect.Map
}

func toSnakeCase(str string) string {
	var result strings.Builder
	for i, r := range str {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}