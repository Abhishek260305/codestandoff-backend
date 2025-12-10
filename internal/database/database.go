package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "thunderbird")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "codestandoff")

	// Build connection string - handle empty password
	var psqlInfo string
	if password == "" {
		psqlInfo = fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			host, port, user, dbname,
		)
	} else {
		psqlInfo = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname,
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
