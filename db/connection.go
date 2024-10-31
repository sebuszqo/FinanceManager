package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

// DBService represents a service that interacts with a database.
type DBService struct {
	DB *sql.DB
}

// NewDBService initializes a new database service by loading environment variables and establishing a connection to the database.
func NewDBService() (*DBService, error) {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, continuing with system environment variables")
	}

	// Get the connection string from environment variables
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		return nil, fmt.Errorf("missing DB_CONNECTION_STRING in environment variables")
	}

	// Open the database connection
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("could not open db connection: %v", err)
	}

	// Set database connection settings
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Ping the database to check connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("could not connect to the database: %v", err)
	}

	return &DBService{DB: db}, nil
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *DBService) Health() map[string]string {
	stats := make(map[string]string)

	err := s.DB.Ping()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"
	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
func (s *DBService) Close() error {
	log.Println("Closing database connection")
	return s.DB.Close()
}
