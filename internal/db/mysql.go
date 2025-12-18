package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/armchr/codeapi/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

// MySQLConnection manages the MySQL database connection
type MySQLConnection struct {
	db     *sql.DB
	config config.MySQLConfig
	logger *zap.Logger
}

// NewMySQLConnection creates a new MySQL connection pool
func NewMySQLConnection(cfg config.MySQLConfig, logger *zap.Logger) (*MySQLConnection, error) {
	// Build DSN (Data Source Name) without database name first
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
	)

	// Add connection parameters
	dsn += "?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"

	logger.Info("Connecting to MySQL",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("username", cfg.Username))

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	conn := &MySQLConnection{
		db:     db,
		config: cfg,
		logger: logger,
	}

	logger.Info("MySQL connection established successfully")
	return conn, nil
}

// EnsureDatabase creates the database if it doesn't exist and reconnects to use it
func (m *MySQLConnection) EnsureDatabase(dbName string) error {
	m.logger.Info("Ensuring database exists", zap.String("database", dbName))

	// Create database if not exists
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName)
	if _, err := m.db.Exec(query); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Close current connection
	m.db.Close()

	// Reconnect with database selected
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		m.config.Username,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		dbName,
	)
	dsn += "?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to reconnect to database %s: %w", dbName, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database %s: %w", dbName, err)
	}

	m.db = db

	m.logger.Info("Database ready", zap.String("database", dbName))
	return nil
}

// GetDB returns the underlying sql.DB connection
func (m *MySQLConnection) GetDB() *sql.DB {
	return m.db
}

// Ping checks if the database connection is alive
func (m *MySQLConnection) Ping() error {
	return m.db.Ping()
}

// Close closes the database connection
func (m *MySQLConnection) Close() error {
	if m.db != nil {
		m.logger.Info("Closing MySQL connection")
		return m.db.Close()
	}
	return nil
}

// Stats returns database statistics
func (m *MySQLConnection) Stats() sql.DBStats {
	return m.db.Stats()
}
