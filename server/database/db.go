package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

var defaultCategories = []struct {
	Name      string
	SortOrder int
}{
	{"消耗品", 1}, {"炼妖石", 2}, {"图册", 3}, {"如意丹", 4},
	{"宝石", 5}, {"装备石", 6}, {"珍珠", 7}, {"书铁", 8},
	{"元宵", 9}, {"附魔宝珠", 10}, {"兽诀", 11}, {"阵法", 12}, {"其他", 13},
}

func InitDB(dbPath string, defaults map[string]string) error {
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

	seedDefaults()
	for k, v := range defaults {
		DB.Exec("INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)", k, v)
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
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	for _, s := range stmts {
		if _, err := DB.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func seedDefaults() {
	var adminCount int
	DB.QueryRow("SELECT COUNT(*) FROM admins").Scan(&adminCount)
	if adminCount == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if err == nil {
			DB.Exec("INSERT INTO admins (username, password) VALUES (?, ?)", "admin", string(hash))
			log.Println("Created default admin: admin/123456")
		}
	}

	for _, cat := range defaultCategories {
		DB.Exec("INSERT OR IGNORE INTO categories (name, sort_order) VALUES (?, ?)", cat.Name, cat.SortOrder)
	}
}
