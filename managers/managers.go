package managers

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sunny-b/gomigrate/managers/mysql"
)

var (
	ErrUnsupportedDriver = errors.New("unsupported driver")
)

type MigrationManager interface {
	CreateMigrationsTable(ctx context.Context, db *sql.DB) error
	FetchCurrentMigrationVersion(ctx context.Context, db *sql.DB) (int64, error)
	IncrementMigrationVersion(ctx context.Context, db *sql.DB) error
	DecrementMigrationVersion(ctx context.Context, db *sql.DB) error
	CreateDatabase(ctx context.Context, db *sql.DB, name string) error
	DropDatabase(ctx context.Context, db *sql.DB, name string) error
}

func GetManagerFromDriver(driver string) (MigrationManager, error) {
	switch driver {
	case "mysql":
		return &mysql.Manager{}, nil
	default:
		return nil, ErrUnsupportedDriver
	}
}
