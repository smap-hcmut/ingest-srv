package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"ingest-srv/config"

	_ "github.com/lib/pq" // PostgreSQL driver
)

const (
	// defaultConnectTimeout is the maximum time to wait for initial connection
	defaultConnectTimeout = 5 * time.Second
	// defaultMaxIdleConns is the maximum number of idle connections in the pool
	defaultMaxIdleConns = 25
	// defaultMaxOpenConns is the maximum number of open connections to the database
	defaultMaxOpenConns = 200
	// defaultConnMaxLifetime is the maximum amount of time a connection may be reused
	defaultConnMaxLifetime = 30 * time.Minute
	// defaultConnMaxIdleTime is the maximum amount of time a connection may be idle
	defaultConnMaxIdleTime = 5 * time.Minute
)

var (
	instance *sql.DB
	once     sync.Once
	mu       sync.RWMutex
	initErr  error // Stores the last initialization error to allow retry
)

// Connect initializes and connects to PostgreSQL database using singleton pattern.
// If connection fails, it can be retried by calling Connect() again.
// Returns the existing connection instance if already connected.
func Connect(ctx context.Context, cfg config.PostgresConfig) (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	// Return existing instance if already connected
	if instance != nil {
		return instance, nil
	}

	// Reset sync.Once if previous initialization failed to allow retry
	if initErr != nil {
		once = sync.Once{}
		initErr = nil
	}

	var err error
	once.Do(func() {
		// Create context with timeout for connection attempt
		connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
		defer cancel()

		// Build connection string with configurable SSL mode
		// Supported modes: disable, require, verify-ca, verify-full
		sslMode := cfg.SSLMode
		if sslMode == "" {
			sslMode = "disable" // Default to disable for local development
		}
		searchPath := cfg.Schema
		if searchPath == "" {
			searchPath = "public"
		}
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslMode, searchPath)

		// Open database connection (does not actually connect yet)
		db, dbErr := sql.Open("postgres", dsn)
		if dbErr != nil {
			err = fmt.Errorf("failed to open PostgreSQL connection: %w", dbErr)
			initErr = err
			return
		}

		// Configure connection pool settings
		db.SetMaxIdleConns(defaultMaxIdleConns)
		db.SetMaxOpenConns(defaultMaxOpenConns)
		db.SetConnMaxLifetime(defaultConnMaxLifetime)
		db.SetConnMaxIdleTime(defaultConnMaxIdleTime)

		// Verify connection by pinging the database
		if pingErr := db.PingContext(connectCtx); pingErr != nil {
			// Close connection to prevent resource leak
			_ = db.Close()
			err = fmt.Errorf("failed to ping PostgreSQL: %w", pingErr)
			initErr = err
			return
		}

		instance = db
	})

	return instance, err
}

// GetClient returns the singleton PostgreSQL client instance.
// Panics if the client has not been initialized by calling Connect() first.
func GetClient() *sql.DB {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("PostgreSQL client not initialized. Call Connect() first")
	}
	return instance
}

// Disconnect closes the PostgreSQL connection and resets the singleton instance.
// This allows a new connection to be established by calling Connect() again.
func Disconnect(ctx context.Context, db *sql.DB) error {
	mu.Lock()
	defer mu.Unlock()

	if db != nil {
		if err := db.Close(); err != nil {
			return fmt.Errorf("failed to close PostgreSQL connection: %w", err)
		}

		instance = nil
		initErr = nil
		once = sync.Once{} // Reset to allow reconnection
	}
	return nil
}

// HealthCheck performs a health check on the PostgreSQL connection by pinging the database.
// Returns an error if the connection is not initialized or the ping fails.
func HealthCheck(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("PostgreSQL client not initialized")
	}

	if err := instance.PingContext(ctx); err != nil {
		return fmt.Errorf("PostgreSQL health check failed: %w", err)
	}

	return nil
}

// IsConnected checks if the PostgreSQL client instance exists.
// Note: This only checks if the instance is initialized, not if the connection is actually alive.
// Use HealthCheck() to verify the connection is working.
func IsConnected() bool {
	mu.RLock()
	defer mu.RUnlock()

	return instance != nil
}

// Reconnect closes the existing PostgreSQL connection and establishes a new one.
// This is useful when you need to reconnect after a connection loss or configuration change.
func Reconnect(ctx context.Context, cfg config.PostgresConfig) error {
	mu.Lock()
	defer mu.Unlock()

	// Close existing connection if present
	if instance != nil {
		_ = instance.Close()
		instance = nil
	}

	// Reset sync.Once and error state to allow new connection
	once = sync.Once{}
	initErr = nil

	// Build connection string with configurable SSL mode
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslMode)

	// Open new database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to create new PostgreSQL connection: %w", err)
	}

	// Configure connection pool settings
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)
	db.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	// Verify connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	if pingErr := db.PingContext(connectCtx); pingErr != nil {
		// Close connection to prevent resource leak
		_ = db.Close()
		return fmt.Errorf("failed to connect to PostgreSQL: %w", pingErr)
	}

	instance = db

	return nil
}
