// Package db provides the MySQL database client for the data service.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQL wraps a *sql.DB with connection lifecycle helpers.
type MySQL struct {
	db *sql.DB
}

// NewMySQL opens a MySQL connection pool with the given DSN.
func NewMySQL(dsn string, maxOpenConns int) (*MySQL, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns / 2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MySQL{db: db}, nil
}

// DB returns the underlying *sql.DB for use by repositories.
func (m *MySQL) DB() *sql.DB {
	return m.db
}

// Ping verifies the database connection is alive.
func (m *MySQL) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

// Close closes the database connection pool.
func (m *MySQL) Close() error {
	return m.db.Close()
}
