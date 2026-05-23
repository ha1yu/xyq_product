package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(dbPath string) error {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := createTables(); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	return nil
}

func createTables() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			sort_order INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			category_id INTEGER NOT NULL,
			remark TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (category_id) REFERENCES categories(id)
		)`,
		`CREATE TABLE IF NOT EXISTS prices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			price_date DATE NOT NULL,
			price REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
			UNIQUE(product_id, price_date)
		)`,
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		)`,
	}

	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func SeedAdmin() error {
	var count int
	DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&count)
	if count > 0 {
		return nil
	}

	_, err := DB.Exec(
		"INSERT INTO admins (username, password) VALUES (?, ?)",
		"admin",
		"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
	)
	return err
}
