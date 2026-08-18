package dbproxy

import "testing"

func TestIsWrite(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want bool
	}{
		{"insert", "INSERT INTO nodes (id, name) VALUES (1, 'n1')", true},
		{"insert ignore", "INSERT IGNORE INTO nodes (id) VALUES (1)", true},
		{"replace", "REPLACE INTO nodes (id, name) VALUES (1, 'n1')", true},
		{"update", "UPDATE nodes SET name = 'n2' WHERE id = 1", true},
		{"delete", "DELETE FROM nodes WHERE id = 1", true},
		{"create table", "CREATE TABLE nodes (id INT PRIMARY KEY, name VARCHAR(64))", true},
		{"alter table", "ALTER TABLE nodes ADD COLUMN uuid VARCHAR(36)", true},
		{"create index", "CREATE INDEX nodes_name_idx ON nodes (name)", true},
		{"drop index", "DROP INDEX nodes_name_idx ON nodes", true},
		{"drop table", "DROP TABLE nodes", true},
		{"truncate", "TRUNCATE TABLE nodes", true},
		{"rename table", "RENAME TABLE nodes TO machines", true},
		{"create database", "CREATE DATABASE ironic", true},
		{"drop database", "DROP DATABASE ironic", true},

		{"select", "SELECT * FROM nodes", false},
		{"select for update", "SELECT * FROM nodes WHERE id = 1 FOR UPDATE", false},
		{"show tables", "SHOW TABLES", false},
		{"show variables", "SHOW VARIABLES LIKE 'sql_mode'", false},
		{"explain", "EXPLAIN SELECT * FROM nodes", false},
		{"use", "USE ironic", false},
		{"begin", "BEGIN", false},
		{"start transaction", "START TRANSACTION", false},
		{"commit", "COMMIT", false},
		{"rollback", "ROLLBACK", false},
		{"set names", "SET NAMES utf8mb4", false},
		{"set session var", "SET @@session.sql_mode = 'ANSI_QUOTES'", false},
		{"set autocommit", "SET autocommit = 0", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IsWrite(tc.sql)
			if err != nil {
				t.Fatalf("IsWrite(%q) returned error: %v", tc.sql, err)
			}
			if got != tc.want {
				t.Errorf("IsWrite(%q) = %v, want %v", tc.sql, got, tc.want)
			}
		})
	}
}

func TestIsWriteParseError(t *testing.T) {
	if _, err := IsWrite("THIS IS NOT SQL AT ALL ((("); err == nil {
		t.Error("IsWrite on invalid SQL: expected error, got nil")
	}
}
