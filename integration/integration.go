//go:build smoke
// +build smoke

package integration

import (
	"context"
	"database/sql"
	"testing"

	mysqlerrors "github.com/go-mysql/errors"
	"github.com/stretchr/testify/require"
	"github.com/sunny-b/gomigrate"
	"github.com/sunny-b/gomigrate/managers/mysql"
)

var (
	migrator *gomigrate.Migrator
)

func init() {
	migrator = gomigrate.NewMigrator(&gomigrate.Config{
		Driver:   "mysql",
		Username: "root",
		Password: "root",
		Database: "testdb",
		Host:     "localhost",
		Port:     "33306",
	})

	migrator.AddMigrations(
		&gomigrate.Migration{
			Name: "First",
			Up:   `CREATE TABLE IF NOT EXISTS testing (id INT);`,
			UpTest: func(ctx context.Context, db *sql.DB, t *testing.T) {
				exists, err := tableExists(ctx, db, "testing")
				require.NoError(t, err)
				require.True(t, exists)
			},
			Down: `DROP TABLE IF EXISTS testing;`,
			DownTest: func(ctx context.Context, db *sql.DB, t *testing.T) {
				exists, err := tableExists(ctx, db, "testing")
				require.NoError(t, err)
				require.False(t, exists)
			},
		},
		&gomigrate.Migration{
			Name: "Second",
			Down: `DROP TABLE IF EXISTS another_test;`,
			DownTest: func(ctx context.Context, db *sql.DB, t *testing.T) {
				exists, err := tableExists(ctx, db, "another_test")
				require.NoError(t, err)
				require.False(t, exists)
			},
			UpFn: func(ctx context.Context, db *sql.DB) error {
				_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS another_test (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(10))`)
				if err != nil {
					return err
				}

				_, err = db.ExecContext(ctx, `INSERT INTO another_test (name) VALUES ('sunny')`)
				if err != nil {
					return err
				}

				return nil
			},
			UpTest: func(ctx context.Context, db *sql.DB, t *testing.T) {
				exists, err := tableExists(ctx, db, "another_test")
				require.NoError(t, err)
				require.True(t, exists)
			},
		},
		&gomigrate.Migration{
			Name: "Third",
			Up:   `CREATE TABLE test2 (id INT);`,
			UpTest: func(ctx context.Context, db *sql.DB, t *testing.T) {
				exists, err := tableExists(ctx, db, "test2")
				require.NoError(t, err)
				require.True(t, exists)
			},
			Down: `DROP TABLE test2;`,
			DownTest: func(ctx context.Context, db *sql.DB, t *testing.T) {
				exists, err := tableExists(ctx, db, "test2")
				require.NoError(t, err)
				require.False(t, exists)
			},
		},
	)
}

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT 1 FROM `+table+` LIMIT 0;`,
	)
	if err == nil {
		rows.Close()
		return true, nil
	}

	if mysqlerrors.MySQLErrorCode(err) == mysql.ERR_NO_SUCH_TABLE {
		return false, nil
	}

	return false, err
}
