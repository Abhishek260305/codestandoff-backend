package controllers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"codestandoff/backend/graph/model"
	query "codestandoff/backend/graph/query/reports"
	"codestandoff/backend/internal/auth"
	"codestandoff/backend/internal/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PCDGraphQLController is the main controller interface for GraphQL operations
type PCDGraphQLController interface {
	// Training
	GetQuestions(ctx context.Context, input model.GetQuestionsRequest) (*model.GetQuestionsResponse, error)

	// Auth
	Me(ctx context.Context) (*model.User, error)
	Users(ctx context.Context) ([]*model.User, error)
	User(ctx context.Context, id string) (*model.User, error)
	Signup(ctx context.Context, email, password string, firstName, lastName *string) (*model.AuthPayload, error)
	Login(ctx context.Context, email, password string) (*model.AuthPayload, error)
	Logout(ctx context.Context) (bool, error)

	// Problems & Matches (placeholders)
	Problems(ctx context.Context) ([]*model.Problem, error)
	Problem(ctx context.Context, id string) (*model.Problem, error)
	Matches(ctx context.Context) ([]*model.Match, error)
	Match(ctx context.Context, id string) (*model.Match, error)
	CreateProblem(ctx context.Context, title, description, difficulty string) (*model.Problem, error)
	CreateMatch(ctx context.Context, problemID string) (*model.Match, error)
}

// PCDGraphQLControllerDeps contains dependencies for the controller
type PCDGraphQLControllerDeps struct {
	DB *sql.DB
}

type pcdGraphQLControllerImpl struct {
	deps PCDGraphQLControllerDeps
}

// NewPCDGraphQLController creates a new PCDGraphQLController
func NewPCDGraphQLController(deps PCDGraphQLControllerDeps) PCDGraphQLController {
	return &pcdGraphQLControllerImpl{
		deps: deps,
	}
}

// Context keys for request/response
type ContextKey string

const (
	ResponseWriterKey ContextKey = "responseWriter"
	RequestKey        ContextKey = "request"
)

// GetResponseWriter gets the ResponseWriter from context
func GetResponseWriter(ctx context.Context) http.ResponseWriter {
	if w, ok := ctx.Value(ResponseWriterKey).(http.ResponseWriter); ok {
		return w
	}
	return nil
}

// GetRequest gets the Request from context
func GetRequest(ctx context.Context) *http.Request {
	if r, ok := ctx.Value(RequestKey).(*http.Request); ok {
		return r
	}
	return nil
}

// GetQuestions fetches questions with pagination and filtering
func (c *pcdGraphQLControllerImpl) GetQuestions(ctx context.Context, input model.GetQuestionsRequest) (*model.GetQuestionsResponse, error) {
	// Set defaults
	offset := 0
	if input.Offset != nil {
		offset = *input.Offset
	}
	limit := 25
	if input.Limit != nil {
		limit = *input.Limit
	}
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Convert topics
	var topics []string
	if input.Topics != nil {
		topics = input.Topics
	}

	// Get query string and arguments from query builder
	queryStr, args := query.GetQuestionsQueryWithArgs(offset, limit+1, input.Search, input.Difficulty, topics, input.SortBy, input.SortOrder) // +1 to check hasMore

	log.Printf("[GetQuestions] Executing query: %s", queryStr)
	log.Printf("[GetQuestions] Query args: %v", args)

	// Execute query
	rows, err := c.deps.DB.Query(queryStr, args...)
	if err != nil {
		log.Printf("[GetQuestions] Query execution error: %v", err)
		return nil, fmt.Errorf("failed to query questions: %w", err)
	}
	defer rows.Close()

	// Scan results
	var questions []*model.Question
	for rows.Next() {
		var q database.Question
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
			log.Printf("[GetQuestions] Row scan error: %v", err)
			return nil, fmt.Errorf("failed to scan question: %w", err)
		}

		question := &model.Question{
			ID:            fmt.Sprintf("%d", q.ID),
			Title:         q.Title,
			Slug:          q.Slug,
			Description:   q.Description,
			Difficulty:    q.Difficulty,
			Topics:        q.Topics,
			TestCaseCount: q.TestCaseCount,
		}
		questions = append(questions, question)
	}

	if err = rows.Err(); err != nil {
		log.Printf("[GetQuestions] Rows iteration error: %v", err)
		return nil, fmt.Errorf("error iterating questions: %w", err)
	}

	// Check if there are more results
	hasMore := len(questions) > limit
	if hasMore {
		questions = questions[:limit] // Remove the extra item
	}

	log.Printf("[GetQuestions] Returning %d questions, hasMore=%v", len(questions), hasMore)

	return &model.GetQuestionsResponse{
		Questions:  questions,
		TotalCount: len(questions),
		HasMore:    hasMore,
	}, nil
}

// Me returns the current authenticated user
func (c *pcdGraphQLControllerImpl) Me(ctx context.Context) (*model.User, error) {
	r := GetRequest(ctx)
	if r == nil {
		return nil, errors.New("not authenticated")
	}

	cookie, err := r.Cookie("auth_token")
	if err != nil {
		return nil, errors.New("not authenticated")
	}

	claims, err := auth.ValidateJWT(cookie.Value)
	if err != nil {
		return nil, errors.New("invalid or expired token")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	dbUser, err := database.GetUserByID(c.deps.DB, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return dbUserToModel(dbUser), nil
}

// Users returns all users
func (c *pcdGraphQLControllerImpl) Users(ctx context.Context) ([]*model.User, error) {
	dbUsers, err := database.GetAllUsers(c.deps.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	users := make([]*model.User, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = dbUserToModel(u)
	}

	return users, nil
}

// User returns a user by ID
func (c *pcdGraphQLControllerImpl) User(ctx context.Context, id string) (*model.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	dbUser, err := database.GetUserByID(c.deps.DB, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return dbUserToModel(dbUser), nil
}

// Signup creates a new user
func (c *pcdGraphQLControllerImpl) Signup(ctx context.Context, email, password string, firstName, lastName *string) (*model.AuthPayload, error) {
	// Check if user already exists
	_, err := database.GetUserByEmail(c.deps.DB, email)
	if err == nil {
		return nil, errors.New("user with this email already exists")
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	firstNameStr := ""
	lastNameStr := ""
	if firstName != nil {
		firstNameStr = *firstName
	}
	if lastName != nil {
		lastNameStr = *lastName
	}

	dbUser, err := database.CreateUser(c.deps.DB, email, password, firstNameStr, lastNameStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	jwtToken, err := auth.GenerateJWT(dbUser.ID, dbUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	expiresAt := auth.GetTokenExpiry()
	session, err := database.CreateSession(c.deps.DB, dbUser.ID, jwtToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set cookie
	c.setAuthCookie(ctx, jwtToken, expiresAt)

	return &model.AuthPayload{
		User:      dbUserToModel(dbUser),
		Token:     jwtToken,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// Login authenticates a user
func (c *pcdGraphQLControllerImpl) Login(ctx context.Context, email, password string) (*model.AuthPayload, error) {
	dbUser, err := database.GetUserByEmail(c.deps.DB, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if !database.VerifyPassword(dbUser.PasswordHash, password) {
		return nil, errors.New("invalid email or password")
	}

	jwtToken, err := auth.GenerateJWT(dbUser.ID, dbUser.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	expiresAt := auth.GetTokenExpiry()
	session, err := database.CreateSession(c.deps.DB, dbUser.ID, jwtToken, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	c.setAuthCookie(ctx, jwtToken, expiresAt)

	return &model.AuthPayload{
		User:      dbUserToModel(dbUser),
		Token:     jwtToken,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// Logout logs out the current user
func (c *pcdGraphQLControllerImpl) Logout(ctx context.Context) (bool, error) {
	r := GetRequest(ctx)
	if r != nil {
		cookie, err := r.Cookie("auth_token")
		if err == nil && cookie != nil {
			database.DeleteSession(c.deps.DB, cookie.Value)
		}
	}

	c.clearAuthCookie(ctx)
	return true, nil
}

// Problems returns all problems (placeholder)
func (c *pcdGraphQLControllerImpl) Problems(ctx context.Context) ([]*model.Problem, error) {
	return []*model.Problem{}, nil
}

// Problem returns a problem by ID (placeholder)
func (c *pcdGraphQLControllerImpl) Problem(ctx context.Context, id string) (*model.Problem, error) {
	return nil, nil
}

// Matches returns all matches (placeholder)
func (c *pcdGraphQLControllerImpl) Matches(ctx context.Context) ([]*model.Match, error) {
	return []*model.Match{}, nil
}

// Match returns a match by ID (placeholder)
func (c *pcdGraphQLControllerImpl) Match(ctx context.Context, id string) (*model.Match, error) {
	return nil, nil
}

// CreateProblem creates a new problem (placeholder)
func (c *pcdGraphQLControllerImpl) CreateProblem(ctx context.Context, title, description, difficulty string) (*model.Problem, error) {
	return &model.Problem{
		ID:          "new-problem-id",
		Title:       title,
		Description: description,
		Difficulty:  difficulty,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}, nil
}

// CreateMatch creates a new match (placeholder)
func (c *pcdGraphQLControllerImpl) CreateMatch(ctx context.Context, problemID string) (*model.Match, error) {
	return &model.Match{
		ID:        "new-match-id",
		Status:    "waiting",
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// Helper functions
func (c *pcdGraphQLControllerImpl) setAuthCookie(ctx context.Context, token string, expiresAt time.Time) {
	w := GetResponseWriter(ctx)
	if w != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    token,
			Path:     "/",
			Expires:  expiresAt,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func (c *pcdGraphQLControllerImpl) clearAuthCookie(ctx context.Context) {
	w := GetResponseWriter(ctx)
	if w != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "auth_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

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
