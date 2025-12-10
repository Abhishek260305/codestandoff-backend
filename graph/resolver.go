package graph

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"codestandoff/backend/graph/model"
	"codestandoff/backend/internal/auth"
	"codestandoff/backend/internal/database"

	"github.com/google/uuid"
)

type Resolver struct {
	db *sql.DB
}

func NewResolver(db *sql.DB) *Resolver {
	return &Resolver{db: db}
}

// User resolvers
func (r *Resolver) Users(ctx context.Context) ([]*model.User, error) {
	dbUsers, err := database.GetAllUsers(r.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	users := make([]*model.User, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = dbUserToModel(u)
	}

	return users, nil
}

func (r *Resolver) User(ctx context.Context, id string) (*model.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	dbUser, err := database.GetUserByID(r.db, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return dbUserToModel(dbUser), nil
}

func (r *Resolver) Me(ctx context.Context) (*model.User, error) {
	// Get token from cookie
	cookie, err := getCookieFromContext(ctx)
	if err != nil || cookie == nil {
		return nil, errors.New("not authenticated")
	}

	// Validate JWT token
	claims, err := auth.ValidateJWT(cookie.Value)
	if err != nil {
		return nil, errors.New("invalid or expired token")
	}

	// Get user ID from claims
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	// Get user from database
	dbUser, err := database.GetUserByID(r.db, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return dbUserToModel(dbUser), nil
}

// Auth mutations
func (r *Resolver) Signup(ctx context.Context, email string, password string, firstName *string, lastName *string) (*model.AuthPayload, error) {
	// Check if user already exists
	_, err := database.GetUserByEmail(r.db, email)
	if err == nil {
		return nil, errors.New("user with this email already exists")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// Create user
	firstNameStr := ""
	lastNameStr := ""
	if firstName != nil {
		firstNameStr = *firstName
	}
	if lastName != nil {
		lastNameStr = *lastName
	}

	dbUser, err := database.CreateUser(r.db, email, password, firstNameStr, lastNameStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate JWT token
	jwtToken, err := auth.GenerateJWT(dbUser.ID, dbUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	// Create session with JWT token
	expiresAt := auth.GetTokenExpiry()
	session, err := database.CreateSession(r.db, dbUser.ID, jwtToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set JWT token in HTTP cookie
	// The extension will also set it from the response, but we try here first
	SetAuthCookie(ctx, jwtToken, expiresAt)

	return &model.AuthPayload{
		User:      dbUserToModel(dbUser),
		Token:     jwtToken,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (r *Resolver) Login(ctx context.Context, email string, password string) (*model.AuthPayload, error) {
	// Get user by email
	dbUser, err := database.GetUserByEmail(r.db, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if !database.VerifyPassword(dbUser.PasswordHash, password) {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	jwtToken, err := auth.GenerateJWT(dbUser.ID, dbUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	// Create session with JWT token
	expiresAt := auth.GetTokenExpiry()
	session, err := database.CreateSession(r.db, dbUser.ID, jwtToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set JWT token in HTTP cookie
	// The extension will also set it from the response, but we try here first
	SetAuthCookie(ctx, jwtToken, expiresAt)

	return &model.AuthPayload{
		User:      dbUserToModel(dbUser),
		Token:     jwtToken,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (r *Resolver) Logout(ctx context.Context) (bool, error) {
	// Get token from cookie
	cookie, err := getCookieFromContext(ctx)
	if err == nil && cookie != nil {
		// Delete session from database
		database.DeleteSession(r.db, cookie.Value)
	}

	// Clear the auth cookie
	ClearAuthCookie(ctx)

	return true, nil
}

// Helper function to convert database user to GraphQL model
func dbUserToModel(u *database.User) *model.User {
	user := &model.User{
		ID:            u.ID.String(),
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     u.UpdatedAt.Format(time.RFC3339),
	}

	if u.FirstName.Valid {
		user.FirstName = &u.FirstName.String
	}
	if u.LastName.Valid {
		user.LastName = &u.LastName.String
	}

	return user
}

// Placeholder implementations for other resolvers
func (r *Resolver) Problems(ctx context.Context) ([]*model.Problem, error) {
	// Placeholder implementation
	return []*model.Problem{}, nil
}

func (r *Resolver) Problem(ctx context.Context, id string) (*model.Problem, error) {
	// Placeholder implementation
	return nil, nil
}

func (r *Resolver) Matches(ctx context.Context) ([]*model.Match, error) {
	// Placeholder implementation
	return []*model.Match{}, nil
}

func (r *Resolver) Match(ctx context.Context, id string) (*model.Match, error) {
	// Placeholder implementation
	return nil, nil
}

func (r *Resolver) CreateProblem(ctx context.Context, title string, description string, difficulty string) (*model.Problem, error) {
	// Placeholder implementation
	return &model.Problem{
		ID:          "new-problem-id",
		Title:       title,
		Description: description,
		Difficulty:  difficulty,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

func (r *Resolver) CreateMatch(ctx context.Context, problemID string) (*model.Match, error) {
	// Placeholder implementation
	return &model.Match{
		ID:        "new-match-id",
		Status:    "waiting",
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}
