package database

import (
	"database/sql"
	"fmt"
	"github.com/kaplan-michael/slack-kudos/pkg/config"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB is a global database connection that handlers can use
var DB *sql.DB

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// Migrations is a list of all database migrations
var Migrations = []Migration{
	{
		Version:     1,
		Description: "Initial schema",
		SQL: `
		CREATE TABLE IF NOT EXISTS kudos (
			user_id TEXT NOT NULL PRIMARY KEY,
			count INTEGER NOT NULL DEFAULT 0
		);
		`,
	},
	{
		Version:     2,
		Description: "Add workspaces table",
		SQL: `
		CREATE TABLE IF NOT EXISTS workspaces (
			team_id TEXT NOT NULL PRIMARY KEY,
			team_name TEXT NOT NULL,
			access_token TEXT NOT NULL,
			bot_user_id TEXT NOT NULL,
			scopes TEXT NOT NULL,
			expires_at TIMESTAMP,
			refresh_token TEXT,
			last_updated TIMESTAMP NOT NULL
		);
		`,
	},
	{
		Version:     3,
		Description: "Add workspace_kudos table",
		SQL: `
		CREATE TABLE IF NOT EXISTS workspace_kudos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			count INTEGER NOT NULL DEFAULT 0,
			UNIQUE(team_id, user_id)
		);
		`,
	},
	{
		Version:     4,
		Description: "Add foreign key constraint to workspace_kudos",
		SQL: `
		PRAGMA foreign_keys = OFF;
		
		CREATE TABLE workspace_kudos_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			count INTEGER NOT NULL DEFAULT 0,
			UNIQUE(team_id, user_id),
			FOREIGN KEY(team_id) REFERENCES workspaces(team_id)
		);
		
		INSERT INTO workspace_kudos_new(id, team_id, user_id, count)
		SELECT id, team_id, user_id, count FROM workspace_kudos;
		
		DROP TABLE workspace_kudos;
		ALTER TABLE workspace_kudos_new RENAME TO workspace_kudos;
		
		PRAGMA foreign_keys = ON;
		`,
	},
}

// InitDB initializes the SQLite database.
func InitDB() error {
	dbName := config.AppConfig.SQLiteFilename

	// Create database file if it doesn't exist
	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		file, err := os.Create(dbName)
		if err != nil {
			return fmt.Errorf("failed to create database file: %w", err)
		}
		file.Close()
	}

	var err error
	DB, err = sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}

	// Enable foreign keys
	_, err = DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		log.Printf("Failed to enable foreign keys: %q\n", err)
		return err
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return DB.Ping()
}

// runMigrations runs all pending database migrations
func runMigrations() error {
	// Create migrations table if it doesn't exist
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS migrations (
		version INTEGER PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL
	);
	`
	_, err := DB.Exec(sqlStmt)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current migration version
	var currentVersion int
	err = DB.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	// Run pending migrations
	for _, migration := range Migrations {
		if migration.Version <= currentVersion {
			// Migration already applied
			continue
		}

		log.Printf("Running migration %d: %s", migration.Version, migration.Description)

		// Begin transaction
		tx, err := DB.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Run migration SQL
		_, err = tx.Exec(migration.SQL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
		}

		// Insert migration record
		_, err = tx.Exec(
			"INSERT INTO migrations (version, description, applied_at) VALUES (?, ?, ?)",
			migration.Version,
			migration.Description,
			time.Now(),
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert migration record: %w", err)
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.Printf("Migration %d applied successfully", migration.Version)
	}

	return nil
}
