package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// DB is the global database connection pool.
var DB *sql.DB

// Connect initialises the connection pool using the given DSN.
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	// Reasonable pool defaults for an MVP.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	log.Println("✅  Database connected")
	DB = db
	return db, nil
}
