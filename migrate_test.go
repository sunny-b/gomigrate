package gomigrate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddMigration(t *testing.T) {
	assert.Equal(t, 0, migrations.Len())

	tests := []struct {
		name    string
		in      *Migration
		version uint64
	}{
		{
			name: "first",
			in: &Migration{
				Name:            "first",
				SimpleMigration: "CREATE TABLE test (id INT);",
				SimpleRollback:  "DROP TABLE test",
			},
			version: 1,
		},
		{
			name: "second",
			in: &Migration{
				Name:            "second",
				SimpleMigration: "ALTER TABLE test ADD COLUMN test INT;",
				SimpleRollback:  "ALTER TABLE test DROP COLUMN test;",
			},
			version: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddMigration(tt.in)
			assert.Equal(t, tt.version, uint64(migrations.Len()))
			assert.Equal(t, tt.version, tt.in.version)
		})
	}
}
