package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sunny-b/gomigrate"
)

func main() {
	var (
		rollback = os.Getenv("ROLLBACK")
	)

	db, err := sql.Open("mysql", "root@tcp(127.0.0.1:33306)/sunny")
	if err != nil {
		fmt.Println("didn't connect")
		return
	}
	fmt.Println("connected!")

	gomigrate.AddMigrations(
		&gomigrate.Migration{
			Name:            "First",
			SimpleMigration: `CREATE TABLE IF NOT EXISTS testing (id INT);`,
			SimpleRollback:  `DROP TABLE IF EXISTS testing;`,
		},
		&gomigrate.Migration{
			Name:           "Second",
			SimpleRollback: `DROP TABLE IF EXISTS another_test;`,
			MigrationFn: func(ctx context.Context, db *sql.DB) error {
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
		},
		&gomigrate.Migration{
			Name:            "Third",
			SimpleMigration: `CREATE TABLE test2 (id INT);`,
			SimpleRollback:  `DROP TABLE test2;`,
		},
	)

	lastVersion, err := gomigrate.Forward(context.TODO(), db, 2)
	if err != nil {
		fmt.Println("failed to run migrations")
		fmt.Println(err.Error())
		return
	}

	fmt.Println("stepped forward!")
	fmt.Println("last version:", lastVersion)

	lastVersion, err = gomigrate.Run(context.TODO(), db)
	if err != nil {
		fmt.Println("failed to run migrations")
		fmt.Println(err.Error())
		return
	}

	fmt.Println("migrated!")
	fmt.Println("last version:", lastVersion)

	if rollback != "" {
		lastVersion, err := gomigrate.Back(context.TODO(), db, 1)
		if err != nil {
			fmt.Println("failed to reset migrations")
			fmt.Println(err.Error())
			return
		}

		fmt.Println("stepped back!")
		fmt.Println("last version:", lastVersion)

		lastVersion, err = gomigrate.Reset(context.TODO(), db)
		if err != nil {
			fmt.Println("failed to reset migrations")
			fmt.Println(err.Error())
			fmt.Println("last version:", lastVersion)
			return
		}

		fmt.Println("reset!")
	}
}
