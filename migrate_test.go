package gomigrate

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddMigration(t *testing.T) {
	m := &Migrator{
		migrations: list.New().Init(),
	}
	assert.Equal(t, 0, m.migrations.Len())

	tests := []struct {
		name    string
		in      *Migration
		version int64
	}{
		{
			name: "first",
			in: &Migration{
				Name: "first",
				Up:   "CREATE TABLE test (id INT);",
				Down: "DROP TABLE test",
			},
			version: 1,
		},
		{
			name: "second",
			in: &Migration{
				Name: "second",
				Up:   "ALTER TABLE test ADD COLUMN test INT;",
				Down: "ALTER TABLE test DROP COLUMN test;",
			},
			version: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.AddMigration(tt.in)
			assert.Equal(t, tt.version, int64(m.migrations.Len()))
			assert.Equal(t, tt.version, tt.in.version)
		})
	}
}
