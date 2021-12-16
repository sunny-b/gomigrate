package mysql

import (
	"context"
	"database/sql"
	"errors"

	mysqlerrors "github.com/go-mysql/errors"
)

type Manager struct{}

func (m *Manager) CreateMigrationsTable(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrNoDatabase
	}

	exists, err := m.checkIfExists(ctx, db)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migrations (
		version INT
	) Engine=INNODB;`)
	if err != nil {
		return err
	}

	var version int64
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

func tableExists(ctx context.Context, db *sql.DB) (bool, error) {
	var exists bool
	rows, err := db.QueryContext(
		ctx,
		`SELECT 1 FROM migrations LIMIT 0;`,
	)
	if err == nil {
		rows.Close()
		return true, nil
	}

	if mysqlerrors.MySQLErrorCode(err) == ERR_NO_SUCH_TABLE {
		return false, nil
	}

	return exists, err
}

func (m *Manager) fetchCurrentMigrationVersion(ctx context.Context, db *sql.DB) (int64, error) {
	var version int64
	err := db.QueryRowContext(ctx, `SELECT version FROM migrations;`).Scan(&version)

	return version, err
}

func (m *Manager) checkIfExists(ctx context.Context, db *sql.DB) (bool, error) {
	exists, err := tableExists(ctx, db)
	if err != nil {
		return exists, err
	}

	if !exists {
		return false, nil
	}

	_, err = m.fetchCurrentMigrationVersion(ctx, db)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (m *Manager) CreateDatabase(ctx context.Context, db *sql.DB, name string) error {
	if db == nil {
		return ErrNoDatabase
	}

	_, err := db.ExecContext(ctx, `CREATE DATABASE IF NOT EXISTS `+name+`;`)
	return err
}

func (m *Manager) DropDatabase(ctx context.Context, db *sql.DB, name string) error {
	if db == nil {
		return ErrNoDatabase
	}

	_, err := db.ExecContext(ctx, `DROP DATABASE IF EXISTS `+name+`;`)
	return err
}

func (m *Manager) FetchCurrentMigrationVersion(ctx context.Context, db *sql.DB) (int64, error) {
	return m.fetchCurrentMigrationVersion(ctx, db)
}

func (m *Manager) IncrementMigrationVersion(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrNoDatabase
	}

	_, err := db.ExecContext(ctx, `UPDATE migrations SET version = version + 1;`)
	return err
}

func (m *Manager) DecrementMigrationVersion(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return ErrNoDatabase
	}

	_, err := db.ExecContext(ctx, `UPDATE migrations SET version = version - 1;`)
	return err
}
