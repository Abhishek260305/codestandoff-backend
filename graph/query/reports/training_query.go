package query

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// GetQuestionsQueryWithArgs returns the query string and arguments separately
func GetQuestionsQueryWithArgs(offset, limit int, search *string, difficulty *string, topics []string, sortBy *string, sortOrder *string) (string, []interface{}) {
	// Base query
	query := "SELECT id, title, slug, description, difficulty, topics, test_case_count FROM public.questions"

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Add search filter if provided (searches by title or id)
	if search != nil && *search != "" {
		searchPattern := "%" + *search + "%"
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR CAST(id AS TEXT) LIKE $%d)", argIndex, argIndex+1))
		args = append(args, searchPattern, *search+"%")
		argIndex += 2
	}

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
	orderBy := "id"
	orderDir := "ASC"

	if sortBy != nil && *sortBy != "" {
		// Validate sortBy - only allow "id" or "difficulty"
		if *sortBy == "id" || *sortBy == "difficulty" {
			orderBy = *sortBy
		}
	}

	if sortOrder != nil && *sortOrder != "" {
		// Validate sortOrder - only allow "ASC" or "DESC"
		if *sortOrder == "ASC" || *sortOrder == "DESC" {
			orderDir = *sortOrder
		}
	}

	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	return query, args
}
