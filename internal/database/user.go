package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                      uuid.UUID
	Email                   string
	PasswordHash            string
	FirstName               sql.NullString
	LastName                sql.NullString
	EmailVerified           bool
	GoogleID                sql.NullString
	GithubID                sql.NullString
	Rating                  sql.NullInt64
	PeakRating              sql.NullInt64
	CurrentRank             sql.NullString
	RankTier                sql.NullString
	RankSubdivision         sql.NullInt64
	GlobalRank              sql.NullInt64
	TotalMatches            sql.NullInt64
	Wins                    sql.NullInt64
	Losses                  sql.NullInt64
	LastRatingUpdate        sql.NullTime
	LastActivity            sql.NullTime
	DemotionProtectionUntil sql.NullTime
	RatingDecayAppliedAt    sql.NullTime
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// CreateUser creates a new user in the database
func CreateUser(db *sql.DB, email, password, firstName, lastName string) (*User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	id := uuid.New()
	now := time.Now()

	var firstNameNull, lastNameNull sql.NullString
	if firstName != "" {
		firstNameNull = sql.NullString{String: firstName, Valid: true}
	}
	if lastName != "" {
		lastNameNull = sql.NullString{String: lastName, Valid: true}
	}

	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, email_verified, google_id, github_id, rating, peak_rating, current_rank, rank_tier, rank_subdivision, total_matches, wins, losses, last_activity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, email, password_hash, first_name, last_name, email_verified, google_id, github_id, rating, peak_rating, current_rank, rank_tier, rank_subdivision, global_rank, total_matches, wins, losses, last_rating_update, last_activity, demotion_protection_until, rating_decay_applied_at, created_at, updated_at
	`

	user := &User{}
	err = db.QueryRow(
		query,
		id,
		email,
		string(hashedPassword),
		firstNameNull,
		lastNameNull,
		false,
		sql.NullString{}, // No Google ID for regular signup
		sql.NullString{}, // No GitHub ID for regular signup
		0,                // Start at Bronze 1 (0 rating)
		0,                // Peak rating starts at 0
		"Bronze 1",       // Starting rank
		"Bronze",         // Starting tier
		1,                // Starting subdivision
		0,                // Total matches
		0,                // Wins
		0,                // Losses
		now,              // Last activity
		now,
		now,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.GoogleID,
		&user.GithubID,
		&user.Rating,
		&user.PeakRating,
		&user.CurrentRank,
		&user.RankTier,
		&user.RankSubdivision,
		&user.GlobalRank,
		&user.TotalMatches,
		&user.Wins,
		&user.Losses,
		&user.LastRatingUpdate,
		&user.LastActivity,
		&user.DemotionProtectionUntil,
		&user.RatingDecayAppliedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, google_id, github_id, rating, peak_rating, current_rank, rank_tier, rank_subdivision, global_rank, total_matches, wins, losses, last_rating_update, last_activity, demotion_protection_until, rating_decay_applied_at, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &User{}
	err := db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.GoogleID,
		&user.GithubID,
		&user.Rating,
		&user.PeakRating,
		&user.CurrentRank,
		&user.RankTier,
		&user.RankSubdivision,
		&user.GlobalRank,
		&user.TotalMatches,
		&user.Wins,
		&user.Losses,
		&user.LastRatingUpdate,
		&user.LastActivity,
		&user.DemotionProtectionUntil,
		&user.RatingDecayAppliedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func GetUserByID(db *sql.DB, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, google_id, github_id, rating, peak_rating, current_rank, rank_tier, rank_subdivision, global_rank, total_matches, wins, losses, last_rating_update, last_activity, demotion_protection_until, rating_decay_applied_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &User{}
	err := db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.GoogleID,
		&user.GithubID,
		&user.Rating,
		&user.PeakRating,
		&user.CurrentRank,
		&user.RankTier,
		&user.RankSubdivision,
		&user.GlobalRank,
		&user.TotalMatches,
		&user.Wins,
		&user.Losses,
		&user.LastRatingUpdate,
		&user.LastActivity,
		&user.DemotionProtectionUntil,
		&user.RatingDecayAppliedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// VerifyPassword checks if the provided password matches the user's password hash
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// GetAllUsers retrieves all users
func GetAllUsers(db *sql.DB) ([]*User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, google_id, github_id, rating, peak_rating, current_rank, rank_tier, rank_subdivision, global_rank, total_matches, wins, losses, last_rating_update, last_activity, demotion_protection_until, rating_decay_applied_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.FirstName,
			&user.LastName,
			&user.EmailVerified,
			&user.GoogleID,
			&user.GithubID,
			&user.Rating,
			&user.PeakRating,
			&user.CurrentRank,
			&user.RankTier,
			&user.RankSubdivision,
			&user.GlobalRank,
			&user.TotalMatches,
			&user.Wins,
			&user.Losses,
			&user.LastRatingUpdate,
			&user.LastActivity,
			&user.DemotionProtectionUntil,
			&user.RatingDecayAppliedAt,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// GetUserByGoogleID retrieves a user by Google ID
func GetUserByGoogleID(db *sql.DB, googleID string) (*User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, google_id, github_id, rating, peak_rating, current_rank, rank_tier, rank_subdivision, global_rank, total_matches, wins, losses, last_rating_update, last_activity, demotion_protection_until, rating_decay_applied_at, created_at, updated_at
		FROM users
		WHERE google_id = $1
	`

	user := &User{}
	err := db.QueryRow(query, googleID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.GoogleID,
		&user.GithubID,
		&user.Rating,
		&user.PeakRating,
		&user.CurrentRank,
		&user.RankTier,
		&user.RankSubdivision,
		&user.GlobalRank,
		&user.TotalMatches,
		&user.Wins,
		&user.Losses,
		&user.LastRatingUpdate,
		&user.LastActivity,
		&user.DemotionProtectionUntil,
		&user.RatingDecayAppliedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUserGoogleID updates the Google ID for a user
func UpdateUserGoogleID(db *sql.DB, userID uuid.UUID, googleID string) error {
	query := `
		UPDATE users
		SET google_id = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := db.Exec(query, googleID, time.Now(), userID)
	return err
}

// GetUserByGithubID retrieves a user by GitHub ID
func GetUserByGithubID(db *sql.DB, githubID string) (*User, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, google_id, github_id, rating, peak_rating, current_rank, rank_tier, rank_subdivision, global_rank, total_matches, wins, losses, last_rating_update, last_activity, demotion_protection_until, rating_decay_applied_at, created_at, updated_at
		FROM users
		WHERE github_id = $1
	`

	user := &User{}
	err := db.QueryRow(query, githubID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.EmailVerified,
		&user.GoogleID,
		&user.GithubID,
		&user.Rating,
		&user.PeakRating,
		&user.CurrentRank,
		&user.RankTier,
		&user.RankSubdivision,
		&user.GlobalRank,
		&user.TotalMatches,
		&user.Wins,
		&user.Losses,
		&user.LastRatingUpdate,
		&user.LastActivity,
		&user.DemotionProtectionUntil,
		&user.RatingDecayAppliedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUserGithubID updates the GitHub ID for a user
func UpdateUserGithubID(db *sql.DB, userID uuid.UUID, githubID string) error {
	query := `
		UPDATE users
		SET github_id = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := db.Exec(query, githubID, time.Now(), userID)
	return err
}
