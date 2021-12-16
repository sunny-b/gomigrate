package gomigrate

import (
	"container/list"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/sunny-b/gomigrate/managers"
	"github.com/xo/dburl"
)

type Migrator struct {
	mgr        managers.MigrationManager
	db         *sql.DB
	config     *Config
	migrations *list.List
}

type Config struct {
	Driver   string
	Username string
	Password string
	Host     string
	Port     string
	Database string
	Options  map[string]string
}

func (c *Config) URL() string {
	u := fmt.Sprintf("%s://%s:%s@%s:%s/%s", c.Driver, c.Username, c.Password, c.Host, c.Port, c.Database)
	if len(c.Options) != 0 {
		u += "?" + c.optionsToString()
	}

	return u
}

func (c *Config) optionsToString() string {
	var options string
	for k, v := range c.Options {
		options += fmt.Sprintf("%s=%s&", k, v)
	}

	return strings.TrimRight(options, "&")
}

type Migration struct {
	Name       string
	version    int64
	Deprecated bool
	Up         string
	UpFn       func(context.Context, *sql.DB) error
	UpTest     func(context.Context, *sql.DB, *testing.T)
	Down       string
	DownFn     func(context.Context, *sql.DB) error
	DownTest   func(context.Context, *sql.DB, *testing.T)
}

func NewMigrator(c *Config) *Migrator {
	mgr, err := managers.GetManagerFromDriver(c.Driver)
	if err != nil {
		panic(err)
	}

	db, err := dburl.Open(c.URL())
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	if err := mgr.CreateMigrationsTable(context.Background(), db); err != nil {
		panic(err)
	}

	return &Migrator{
		mgr:        mgr,
		db:         db,
		config:     c,
		migrations: list.New().Init(),
	}
}

func (m *Migrator) Len() int64 {
	return int64(m.migrations.Len())
}

func (m *Migrator) AddMigration(migration *Migration) {
	migration.version = int64(m.migrations.Len()) + 1

	m.migrations.PushBack(migration)
}

func (m *Migrator) AddMigrations(steps ...*Migration) []*Migration {
	for _, migration := range steps {
		m.AddMigration(migration)
	}

	return steps
}

func (m *Migrator) Run(ctx context.Context) (int64, error) {
	return m.StepTo(ctx, int64(m.migrations.Len()))
}

func (m *Migrator) Reset(ctx context.Context) (int64, error) {
	return m.RollbackTo(ctx, 0)
}

func (m *Migrator) StepTo(ctx context.Context, targetVersion int64) (int64, error) {
	if err := m.lazyInit(ctx, m.db); err != nil {
		return -1, err
	}

	lastVersionRan, err := m.mgr.FetchCurrentMigrationVersion(ctx, m.db)
	if err != nil {
		return -1, err
	}

	return m.stepFromTo(ctx, lastVersionRan, targetVersion)
}

func (m *Migrator) Step(ctx context.Context, db *sql.DB) (int64, error) {
	if err := m.lazyInit(ctx, db); err != nil {
		return -1, err
	}

	lastVersionRan, err := m.mgr.FetchCurrentMigrationVersion(ctx, db)
	if err != nil {
		return -1, err
	}

	return m.stepFromTo(ctx, lastVersionRan, lastVersionRan+1)
}

func (m *Migrator) stepFromTo(ctx context.Context, currentVersion, targetVersion int64) (int64, error) {
	for mig := m.migrations.Front(); mig != nil; mig = mig.Next() {
		if migration, ok := mig.Value.(*Migration); ok {
			if migration.version <= currentVersion || migration.Deprecated {
				continue
			}

			if migration.version > targetVersion {
				return currentVersion, nil
			}

			err := m.step(ctx, m.db, migration)
			if err != nil {
				return currentVersion, err
			}

			currentVersion = migration.version
		}
	}

	return currentVersion, nil
}

func (m *Migrator) step(ctx context.Context, db *sql.DB, migration *Migration) error {
	var err error
	if migration.Up != "" {
		_, err = db.ExecContext(ctx, migration.Up)
	} else {
		err = migration.UpFn(ctx, db)
	}
	if err != nil {
		return err
	}

	return m.mgr.IncrementMigrationVersion(ctx, db)
}

func (m *Migrator) RollbackTo(ctx context.Context, targetVersion int64) (int64, error) {
	if err := m.lazyInit(ctx, m.db); err != nil {
		return -1, err
	}

	lastVersionRan, err := m.mgr.FetchCurrentMigrationVersion(ctx, m.db)
	if err != nil {
		return -1, err
	}

	return m.rollbackFromTo(ctx, lastVersionRan, targetVersion)
}

func (m *Migrator) Rollback(ctx context.Context) (int64, error) {
	if err := m.lazyInit(ctx, m.db); err != nil {
		return -1, err
	}

	lastVersionRan, err := m.mgr.FetchCurrentMigrationVersion(ctx, m.db)
	if err != nil {
		return -1, err
	}

	return m.rollbackFromTo(ctx, lastVersionRan, lastVersionRan-1)
}

func (m *Migrator) rollbackFromTo(ctx context.Context, currentVersion, targetVersion int64) (int64, error) {
	for mig := m.migrations.Back(); mig != nil; mig = mig.Prev() {
		if migration, ok := mig.Value.(*Migration); ok {
			if migration.version > currentVersion || migration.Deprecated {
				continue
			}

			if migration.version == targetVersion {
				return currentVersion, nil
			}

			err := m.rollback(ctx, m.db, migration)
			if err != nil {
				return currentVersion, err
			}

			currentVersion = migration.version
		}
	}

	return currentVersion, nil
}

func (m *Migrator) rollback(ctx context.Context, db *sql.DB, migration *Migration) error {
	var err error
	if migration.Down != "" {
		_, err = db.ExecContext(ctx, migration.Down)
	} else {
		err = migration.DownFn(ctx, db)
	}
	if err != nil {
		return err
	}

	return m.mgr.DecrementMigrationVersion(ctx, db)
}

func (m *Migrator) lazyInit(ctx context.Context, db *sql.DB) error {
	return m.mgr.CreateMigrationsTable(ctx, db)
	// if db == nil {
	// 	return ErrNoDatabase
	// }

	// ok, err := verifyMigrationsTable(ctx, db)
	// if err != nil {
	// 	return err
	// }

	// if ok {
	// 	return nil
	// }

	// _, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS migrations (
	// 	version INT
	// );`)
	// if err != nil {
	// 	return err
	// }

	// var version int64
	// err = db.QueryRowContext(ctx, `SELECT version FROM migrations;`).Scan(&version)
	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		_, err = db.ExecContext(ctx, `INSERT INTO migrations VALUES (0);`)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		return nil
	// 	}

	// 	return err
	// }

	// return nil
}

func (m *Migrator) CurrentVersion(ctx context.Context) (int64, error) {
	if err := m.lazyInit(ctx, m.db); err != nil {
		return -1, err
	}

	return m.mgr.FetchCurrentMigrationVersion(ctx, m.db)
}

func (m *Migrator) RunUpTests(t *testing.T) {
	var prevMigrations []*Migration

	for mig := m.migrations.Front(); mig != nil; mig = mig.Next() {
		migration := mig.Value.(*Migration)
		if migration.UpTest == nil {
			prevMigrations = append(prevMigrations, migration)
			continue
		}

		t.Run(migration.Name, m.withTestDB(func(t *testing.T, db *sql.DB) {
			ctx := context.Background()

			prevMigrations = append(prevMigrations, migration)

			// run previous migrations to setup for current test
			for _, prevMigration := range prevMigrations {
				err := m.step(ctx, db, prevMigration)
				require.NoError(t, err)
			}

			migration.UpTest(ctx, db, t)
		}))
	}
}

func (m *Migrator) RunDownTests(t *testing.T) {
	var prevMigrations []*Migration

	for mig := m.migrations.Front(); mig != nil; mig = mig.Next() {
		migration := mig.Value.(*Migration)
		if migration.DownTest == nil {
			prevMigrations = append(prevMigrations, migration)
			continue
		}

		t.Run(migration.Name, m.withTestDB(func(t *testing.T, db *sql.DB) {
			ctx := context.Background()

			prevMigrations = append(prevMigrations, migration)

			// run previous migrations to setup for current test
			for _, prevMigration := range prevMigrations {
				err := m.step(ctx, db, prevMigration)
				require.NoError(t, err)
			}

			err := m.rollback(ctx, db, migration)
			require.NoError(t, err)

			migration.DownTest(ctx, db, t)
		}))
	}
}

func (m *Migrator) RunAllTests(t *testing.T) {
	t.Helper()
	err := m.lazyInit(context.Background(), m.db)
	require.NoError(t, err)

	var prevMigrations []*Migration

	for mig := m.migrations.Front(); mig != nil; mig = mig.Next() {
		migration := mig.Value.(*Migration)
		if migration.UpTest == nil && migration.DownTest == nil {
			prevMigrations = append(prevMigrations, migration)
			continue
		}

		t.Run(migration.Name, m.withTestDB(func(t *testing.T, db *sql.DB) {
			ctx := context.Background()

			prevMigrations = append(prevMigrations, migration)

			// run previous migrations to setup for current test
			for _, prevMigration := range prevMigrations {
				err := m.step(ctx, db, prevMigration)
				require.NoError(t, err)
			}

			if migration.UpTest != nil {
				migration.UpTest(ctx, db, t)
			}

			if migration.DownTest != nil {
				err := m.rollback(ctx, db, migration)
				require.NoError(t, err)

				migration.DownTest(ctx, db, t)
			}
		}))
	}
}

func (m *Migrator) withTestDB(f func(t *testing.T, db *sql.DB)) func(t *testing.T) {
	return func(t *testing.T) {
		testID := make([]byte, 2)
		_, err := rand.Read(testID)
		require.NoError(t, err)

		dbName := fmt.Sprintf("migration_test_%x", hex.EncodeToString(testID))
		// _, err = m.db.Exec(fmt.Sprintf("CREATE DATABASE %s;", dbName))
		err = m.mgr.CreateDatabase(context.Background(), m.db, dbName)
		require.NoError(t, err)
		// defer m.db.Exec(fmt.Sprintf("DROP DATABASE %s;", dbName))
		defer m.mgr.DropDatabase(context.Background(), m.db, dbName)

		testConfig := *m.config
		testConfig.Database = dbName

		testDB, err := dburl.Open(testConfig.URL())
		require.NoError(t, err)
		defer testDB.Close()

		m.lazyInit(context.Background(), testDB)
		f(t, testDB)
	}
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
