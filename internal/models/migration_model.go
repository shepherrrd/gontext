package models

import (
	"time"
)

type Migration struct {
	Id          string    `gontext:"primary_key"`
	Name        string    `gontext:"not_null"`
	AppliedAt   time.Time `gontext:"not_null"`
	Version     int       `gontext:"not_null"`
	Checksum    string    `gontext:"not_null"`
	DependsOn   *string   `gontext:"nullable"` // ID of the migration this depends on
}

type MigrationOperation struct {
	Type       MigrationOperationType
	EntityName string
	Details    interface{}
}

type MigrationOperationType int

const (
	CreateTable MigrationOperationType = iota
	DropTable
	AddColumn
	DropColumn
	RenameColumn
	ModifyColumn
	AddIndex
	DropIndex
	AddForeignKey
	DropForeignKey
	RawSQL
)

type CreateTableOperation struct {
	TableName string
	Columns   []ColumnDefinition
	Indexes   []IndexDefinition
}

type DropTableOperation struct {
	TableName string
}

type AddColumnOperation struct {
	TableName string
	Column    ColumnDefinition
}

type DropColumnOperation struct {
	TableName  string
	ColumnName string
}

type RenameColumnOperation struct {
	TableName   string
	OldName     string
	NewName     string
}

type ModifyColumnOperation struct {
	TableName string
	Column    ColumnDefinition
}

type ColumnDefinition struct {
	Name         string
	Type         string
	IsNullable   bool
	IsPrimary    bool
	IsUnique     bool
	DefaultValue *string
	References   *ForeignKeyReference
}

type IndexDefinition struct {
	Name      string
	Columns   []string
	IsUnique  bool
}

type ForeignKeyReference struct {
	Table  string
	Column string
	OnDelete string
	OnUpdate string
}