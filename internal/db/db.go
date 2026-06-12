package db

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	DB.SetMaxOpenConns(1)
	if err := migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := seedSettings(); err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}
	return nil
}

func Close() {
	if DB != nil {
		_ = DB.Close()
	}
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT '',
		price REAL,
		currency TEXT NOT NULL DEFAULT 'USD',
		priority INTEGER NOT NULL DEFAULT 0,
		category TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'pending',
		desired_date TEXT,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		purchased_at TEXT,
		price_confirmed INTEGER NOT NULL DEFAULT 0,
		image_url TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS paydays (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		frequency TEXT NOT NULL DEFAULT 'monthly',
		day_of_month INTEGER,
		day_of_week INTEGER,
		interval_val INTEGER DEFAULT 1,
		label TEXT NOT NULL DEFAULT 'Payday',
		next_date TEXT,
		active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS budget_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		amount REAL NOT NULL,
		label TEXT NOT NULL DEFAULT '',
		effective_date TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS purchase_plan (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER NOT NULL REFERENCES items(id),
		scheduled_date TEXT NOT NULL,
		payday_id INTEGER REFERENCES paydays(id),
		budget_cycle TEXT NOT NULL,
		rank INTEGER NOT NULL DEFAULT 0,
		amount_allocated REAL,
		status TEXT NOT NULL DEFAULT 'planned',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		notes TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS item_savings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER NOT NULL REFERENCES items(id),
		accumulated REAL NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS notification_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER REFERENCES items(id),
		type TEXT NOT NULL,
		channel TEXT NOT NULL,
		subject TEXT NOT NULL DEFAULT '',
		body TEXT NOT NULL DEFAULT '',
		sent_at TEXT NOT NULL DEFAULT (datetime('now')),
		delivered INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS price_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER NOT NULL REFERENCES items(id),
		price REAL,
		currency TEXT NOT NULL DEFAULT 'USD',
		scraped_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS purchase_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER NOT NULL REFERENCES items(id),
		planned_price REAL,
		actual_price REAL NOT NULL,
		currency TEXT NOT NULL DEFAULT 'USD',
		purchased_at TEXT NOT NULL DEFAULT (datetime('now')),
		notes TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);
	`
	if _, err := DB.Exec(schema); err != nil {
		return fmt.Errorf("exec schema: %w", err)
	}

	migrations := []string{
		`ALTER TABLE items ADD COLUMN image_url TEXT NOT NULL DEFAULT ''`,
	}
	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil && !strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	log.Println("database migrated successfully")
	return nil
}

func seedSettings() error {
	for k, v := range defaultSettings {
		_, err := DB.Exec(
			"INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)",
			k, v,
		)
		if err != nil {
			return fmt.Errorf("seed setting %s: %w", k, err)
		}
	}
	return nil
}
