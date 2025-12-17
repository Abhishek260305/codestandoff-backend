package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateSession creates a new session for a user
func CreateSession(db *sql.DB, userID uuid.UUID, token string, expiresAt time.Time) (*Session, error) {
	id := uuid.New()
	now := time.Now()

	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token, expires_at, created_at
	`

	session := &Session{}
	err := db.QueryRow(
		query,
		id,
		userID,
		token,
		expiresAt,
		now,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSessionByToken retrieves a session by token
func GetSessionByToken(db *sql.DB, token string) (*Session, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions
		WHERE token = $1 AND expires_at > NOW()
	`

	session := &Session{}
	err := db.QueryRow(query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession deletes a session by token
func DeleteSession(db *sql.DB, token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := db.Exec(query, token)
	return err
}

// DeleteUserSessions deletes all sessions for a user
func DeleteUserSessions(db *sql.DB, userID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE user_id = $1`
	_, err := db.Exec(query, userID)
	return err
}

// CleanExpiredSessions removes expired sessions
func CleanExpiredSessions(db *sql.DB) error {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`
	_, err := db.Exec(query)
	return err
}

