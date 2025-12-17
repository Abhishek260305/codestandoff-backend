package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// Question represents a question in the database
type Question struct {
	ID            int
	Title         string
	Slug          string
	Description   string
	Difficulty    string
	Topics        []string
	TestCaseCount int
}

// GetQuestions retrieves questions from the database with optional filtering and pagination
func GetQuestions(db *sql.DB, offset, limit int, difficulty *string, topics []string) ([]Question, error) {
	// Build base query
	query := "SELECT id, title, slug, description, difficulty, topics, test_case_count FROM questions"

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add difficulty filter if provided
	if difficulty != nil && *difficulty != "" {
		conditions = append(conditions, fmt.Sprintf("difficulty = $%d", argIndex))
		args = append(args, *difficulty)
		argIndex++
	}

	// Add topics filter if provided
	if len(topics) > 0 {
		conditions = append(conditions, fmt.Sprintf("topics && $%d", argIndex))
		args = append(args, pq.Array(topics))
		argIndex++
	}

	// Add WHERE clause if conditions exist
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering
	query += " ORDER BY id ASC"

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer rows.Close()

	// Scan results
	var questions []Question
	for rows.Next() {
		var q Question
		err := rows.Scan(
			&q.ID,
			&q.Title,
			&q.Slug,
			&q.Description,
			&q.Difficulty,
			pq.Array(&q.Topics),
			&q.TestCaseCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}
		questions = append(questions, q)
	}

	// Check for errors from iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating questions: %w", err)
	}

	return questions, nil
}
