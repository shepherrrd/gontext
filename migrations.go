package gontext

import "github.com/shepherrrd/gontext/internal/migrations"

type MigrationManager = migrations.MigrationManager

func NewMigrationManager(ctx *DbContext, migrationsDir, packageName string) *MigrationManager {
	return migrations.NewMigrationManager(ctx, migrationsDir, packageName)
}