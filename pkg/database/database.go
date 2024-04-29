package database

import (
	"database/sql"
	"github.com/kaplan-michael/slack-kudos/pkg/config"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// DB is a global database connection that handlers can use
var DB *sql.DB

// InitDB initializes the SQLite database.
func InitDB() error {
	dbName := config.AppConfig.SQLiteFilename

	if _, err := os.Stat(dbName); os.IsNotExist(err) {
		if err := createDB(dbName); err != nil {
			return err
		}
	}

	var err error
	DB, err = sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}

	return DB.Ping()
}

// createDB creates the database and tables if they do not exist
func createDB(dbName string) error {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	sqlStmt := `
	CREATE TABLE kudos (
		user_id TEXT NOT NULL PRIMARY KEY,
		count INTEGER NOT NULL DEFAULT 0
	);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return err
	}
	return nil
}
