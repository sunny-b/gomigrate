package gomigrate

import (
	"container/list"
	"context"
	"database/sql"
	"errors"
	"testing"
)

var migrations list.List

type Migration struct {
	Name            string
	version         uint64
	Deprecated      bool
	SimpleMigration string
	MigrationFn     func(context.Context, *sql.DB) error
	MigrationTest   func(context.Context, *sql.DB, *testing.T)
	SimpleRollback  string
	RollbackFn      func(context.Context, *sql.DB) error
	RollbackTest    func(context.Context, *sql.DB, *testing.T)
}

func AddMigration(m *Migration) {
	m.version = uint64(migrations.Len()) + 1

	migrations.PushBack(m)
}

func AddMigrations(steps ...*Migration) {
	for _, m := range steps {
		AddMigration(m)
	}
}

func Run(ctx context.Context, db *sql.DB) (uint64, error) {
	if err := lazyInit(ctx, db); err != nil {
		return 0, err
	}

	currentVersion, err := fetchCurrentMigrationVersion(ctx, db)
	if err != nil {
		return 0, err
	}

	lastVersionRan := currentVersion

	for m := migrations.Front(); m != nil; m = m.Next() {
		if migration, ok := m.Value.(*Migration); ok {
			if migration.version <= currentVersion || migration.Deprecated {
				continue
			}

			lastVersionRan = migration.version

			var err error
			if migration.SimpleMigration != "" {
				_, err = db.ExecContext(ctx, migration.SimpleMigration)
			} else {
				err = migration.MigrationFn(ctx, db)
			}
			if err != nil {
				return lastVersionRan, err
			}

			_, err = db.ExecContext(ctx, `UPDATE migrations SET version = version+1;`)
			if err != nil {
				return lastVersionRan, err
			}
		}
	}

	return lastVersionRan, nil
}

func Reset(ctx context.Context, db *sql.DB) (uint64, error) {
	if err := lazyInit(ctx, db); err != nil {
		return 0, err
	}

	currentVersion, err := fetchCurrentMigrationVersion(ctx, db)
	if err != nil {
		return 0, err
	}

	lastVersionRan := currentVersion

	for m := migrations.Back(); m != nil; m = m.Prev() {
		if migration, ok := m.Value.(*Migration); ok {
			if migration.version > currentVersion || migration.Deprecated {
				continue
			}

			lastVersionRan = migration.version

			var err error
			if migration.SimpleRollback != "" {
				_, err = db.ExecContext(ctx, migration.SimpleRollback)
			} else {
				err = migration.RollbackFn(ctx, db)
			}
			if err != nil {
				return lastVersionRan, err
			}

			_, err = db.ExecContext(ctx, `UPDATE migrations SET version = version-1;`)
			if err != nil {
				return lastVersionRan, err
			}
		}
	}

	return lastVersionRan, nil
}

func Forward(ctx context.Context, db *sql.DB, targetVersion uint64) (uint64, error) {
	if err := lazyInit(ctx, db); err != nil {
		return 0, err
	}

	currentVersion, err := fetchCurrentMigrationVersion(ctx, db)
	if err != nil {
		return 0, err
	}

	lastVersionRan := currentVersion

	for m := migrations.Front(); m != nil; m = m.Next() {
		if migration, ok := m.Value.(*Migration); ok {
			if migration.version <= currentVersion || migration.Deprecated {
				continue
			}

			if migration.version > targetVersion {
				return lastVersionRan, nil
			}

			lastVersionRan = migration.version

			var err error
			if migration.SimpleMigration != "" {
				_, err = db.ExecContext(ctx, migration.SimpleMigration)
			} else {
				err = migration.MigrationFn(ctx, db)
			}
			if err != nil {
				return lastVersionRan, err
			}

			_, err = db.ExecContext(ctx, `UPDATE migrations SET version = version+1;`)
			if err != nil {
				return lastVersionRan, err
			}
		}
	}

	return lastVersionRan, nil
}

func Back(ctx context.Context, db *sql.DB, targetVersion uint64) (uint64, error) {
	if err := lazyInit(ctx, db); err != nil {
		return 0, err
	}

	currentVersion, err := fetchCurrentMigrationVersion(ctx, db)
	if err != nil {
		return 0, err
	}

	lastVersionRan := currentVersion

	for m := migrations.Back(); m != nil; m = m.Prev() {
		if migration, ok := m.Value.(*Migration); ok {
			if migration.version > currentVersion || migration.Deprecated {
				continue
			}

			if migration.version == targetVersion {
				return lastVersionRan, nil
			}

			lastVersionRan = migration.version

			var err error
			if migration.SimpleRollback != "" {
				_, err = db.ExecContext(ctx, migration.SimpleRollback)
			} else {
				err = migration.RollbackFn(ctx, db)
			}
			if err != nil {
				return lastVersionRan, err
			}

			_, err = db.ExecContext(ctx, `UPDATE migrations SET version = version-1;`)
			if err != nil {
				return lastVersionRan, err
			}
		}
	}

	return lastVersionRan, nil
}

func lazyInit(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migrations (
		version INT
	);`)
	if err != nil {
		return err
	}

	var version uint64
	err = db.QueryRowContext(ctx, `SELECT version FROM migrations;`).Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = db.ExecContext(ctx, `INSERT INTO migrations VALUES (0);`)
			if err != nil {
				return err
			}
			return nil
		}

		return err
	}

	return nil
}

func fetchCurrentMigrationVersion(ctx context.Context, db *sql.DB) (uint64, error) {
	var version uint64
	err := db.QueryRowContext(ctx, `SELECT version FROM migrations;`).Scan(&version)

	return version, err
}

/*
	AddMigration
	Run (applies all migrations)
	Reset (rollback all migrations)
	Forward (apply 1 or more migrations)
	Back (rollback one or more migrations)

	lazyInit()
	getCurrentMigrationVersion()


*/
