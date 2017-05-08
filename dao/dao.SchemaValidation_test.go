package dao_test

import "testing"

func TestExpectedCountOfDatabaseObjects(t *testing.T) {

	// Add additional tests to this slice of struct
	var cases = []struct {
		name     string
		sql      string
		expected int
	}{
		{
			name:     "tables",
			sql:      `select count(*) from information_schema.tables where table_schema = database();`,
			expected: 43,
		},
		{
			name:     "triggers",
			sql:      `select count(*) from information_schema.triggers where trigger_schema = database();`,
			expected: 67,
		},
	}

	for _, c := range cases {
		row := d.MetadataDB.QueryRow(c.sql)
		var actual int
		row.Scan(&actual)

		if actual != c.expected {
			t.Errorf("Expected count %v in schema for %v, but got %v", c.expected, c.name, actual)
		}
	}

}
