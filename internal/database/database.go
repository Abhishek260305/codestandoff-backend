package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	// Check for DATABASE_URL first (common in cloud providers)
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		db, err := sql.Open("postgres", databaseURL)
		if err != nil {
			return nil, err
		}
		err = db.Ping()
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	// Fall back to individual environment variables
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "thunderbird")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "codestandoff")
	sslmode := getEnv("DB_SSLMODE", "disable") // Default to disable for local, require for cloud

	// Build connection string - handle empty password
	var psqlInfo string
	if password == "" {
		psqlInfo = fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=%s",
			host, port, user, dbname, sslmode,
		)
	} else {
		psqlInfo = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode,
		)
	}

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
