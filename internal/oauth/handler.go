package oauth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"codestandoff/backend/internal/auth"
	"codestandoff/backend/internal/database"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	googleoauth2 "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// GoogleAuth initiates Google OAuth flow
func (h *Handler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	state, err := auth.GenerateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Get redirect URI from query parameter
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "http://localhost:3000"
	}

	// Store state and redirect URI in cookies for validation
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    redirectURI,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	config := auth.GetGoogleOAuthConfig()
	if config == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GoogleCallback handles Google OAuth callback
func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Get redirect URI from cookie
	redirectURI := "http://localhost:3000" // default
	if redirectCookie, err := r.Cookie("oauth_redirect_uri"); err == nil {
		redirectURI = redirectCookie.Value
	}

	// Clear state and redirect cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not provided", http.StatusBadRequest)
		return
	}

	config := auth.GetGoogleOAuthConfig()
	if config == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	token, err := config.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	// Get user info from Google
	client := config.Client(r.Context(), token)
	service, err := googleoauth2.NewService(r.Context(), option.WithHTTPClient(client))
	if err != nil {
		log.Printf("Error creating OAuth2 service: %v", err)
		http.Error(w, "Failed to create OAuth2 service", http.StatusInternalServerError)
		return
	}

	userInfo, err := service.Userinfo.Get().Do()
	if err != nil {
		log.Printf("Error getting user info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Create or get user
	dbUser, err := h.createOrGetOAuthUser(userInfo.Email, userInfo.GivenName, userInfo.FamilyName, userInfo.Id)
	if err != nil {
		log.Printf("Error creating/getting user: %v", err)
		http.Error(w, "Failed to create/get user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	jwtToken, err := auth.GenerateJWT(dbUser.ID, dbUser.Email)
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Create session
	expiresAt := auth.GetTokenExpiry()
	_, err = database.CreateSession(h.db, dbUser.ID, jwtToken, expiresAt)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set JWT token in HTTP cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		Domain:   "",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	// Redirect to frontend (using the redirect URI from cookie)
	http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
}

// createOrGetOAuthUser creates a new user from OAuth or returns existing user
func (h *Handler) createOrGetOAuthUser(email, firstName, lastName, googleID string) (*database.User, error) {
	// Check if user exists by email
	user, err := database.GetUserByEmail(h.db, email)
	if err == nil {
		// User exists, update Google ID if not set
		if !user.GoogleID.Valid || user.GoogleID.String == "" {
			err := database.UpdateUserGoogleID(h.db, user.ID, googleID)
			if err != nil {
				return nil, fmt.Errorf("failed to update Google ID: %w", err)
			}
			user.GoogleID = sql.NullString{String: googleID, Valid: true}
		}
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// Check if user exists by Google ID
	user, err = database.GetUserByGoogleID(h.db, googleID)
	if err == nil {
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check user by Google ID: %w", err)
	}

	// Create new user (no password for OAuth users)
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

	newUser := &database.User{}
	err = h.db.QueryRow(
		query,
		id,
		email,
		"", // No password for OAuth users
		firstNameNull,
		lastNameNull,
		true, // Email verified by Google
		googleID,
		sql.NullString{}, // No GitHub ID for Google OAuth
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
		&newUser.ID,
		&newUser.Email,
		&newUser.PasswordHash,
		&newUser.FirstName,
		&newUser.LastName,
		&newUser.EmailVerified,
		&newUser.GoogleID,
		&newUser.GithubID,
		&newUser.Rating,
		&newUser.PeakRating,
		&newUser.CurrentRank,
		&newUser.RankTier,
		&newUser.RankSubdivision,
		&newUser.GlobalRank,
		&newUser.TotalMatches,
		&newUser.Wins,
		&newUser.Losses,
		&newUser.LastRatingUpdate,
		&newUser.LastActivity,
		&newUser.DemotionProtectionUntil,
		&newUser.RatingDecayAppliedAt,
		&newUser.CreatedAt,
		&newUser.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// GitHubAuth initiates GitHub OAuth flow
func (h *Handler) GitHubAuth(w http.ResponseWriter, r *http.Request) {
	state, err := auth.GenerateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Get redirect URI from query parameter
	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		redirectURI = "http://localhost:3000"
	}

	// Store state and redirect URI in cookies for validation
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    redirectURI,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	config := auth.GetGitHubOAuthConfig()
	if config == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GitHubCallback handles GitHub OAuth callback
func (h *Handler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	if state != stateCookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Get redirect URI from cookie
	redirectURI := "http://localhost:3000" // default
	if redirectCookie, err := r.Cookie("oauth_redirect_uri"); err == nil {
		redirectURI = redirectCookie.Value
	}

	// Clear state and redirect cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_redirect_uri",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not provided", http.StatusBadRequest)
		return
	}

	config := auth.GetGitHubOAuthConfig()
	if config == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	token, err := config.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("Error exchanging code for token: %v", err)
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	// Get user info from GitHub
	userInfo, err := h.getGitHubUserInfo(r.Context(), token.AccessToken)
	if err != nil {
		log.Printf("Error getting user info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Create or get user
	dbUser, err := h.createOrGetGitHubUser(userInfo.Email, userInfo.Name, userInfo.Login, userInfo.ID)
	if err != nil {
		log.Printf("Error creating/getting user: %v", err)
		http.Error(w, "Failed to create/get user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	jwtToken, err := auth.GenerateJWT(dbUser.ID, dbUser.Email)
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Create session
	expiresAt := auth.GetTokenExpiry()
	_, err = database.CreateSession(h.db, dbUser.ID, jwtToken, expiresAt)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set JWT token in HTTP cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    jwtToken,
		Path:     "/",
		Domain:   "",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	// Redirect to frontend (using the redirect URI from cookie)
	http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
}

// GitHubUserInfo represents GitHub user information
type GitHubUserInfo struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// getGitHubUserInfo fetches user information from GitHub API
func (h *Handler) getGitHubUserInfo(ctx context.Context, accessToken string) (*GitHubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var userInfo GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// If email is not public, fetch from emails endpoint
	if userInfo.Email == "" {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var emails []struct {
				Email   string `json:"email"`
				Primary bool   `json:"primary"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&emails); err == nil {
				for _, email := range emails {
					if email.Primary {
						userInfo.Email = email.Email
						break
					}
				}
				// If no primary email, use the first one
				if userInfo.Email == "" && len(emails) > 0 {
					userInfo.Email = emails[0].Email
				}
			}
		}
	}

	return &userInfo, nil
}

// createOrGetGitHubUser creates a new user from GitHub OAuth or returns existing user
func (h *Handler) createOrGetGitHubUser(email, name, login string, githubID int) (*database.User, error) {
	githubIDStr := fmt.Sprintf("%d", githubID)
	
	// Split name into first and last name
	var firstName, lastName string
	if name != "" {
		parts := splitName(name)
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = parts[1]
		}
	} else {
		firstName = login
	}

	// Check if user exists by email
	user, err := database.GetUserByEmail(h.db, email)
	if err == nil {
		// User exists, update GitHub ID if not set
		if !user.GithubID.Valid || user.GithubID.String == "" {
			err := database.UpdateUserGithubID(h.db, user.ID, githubIDStr)
			if err != nil {
				return nil, fmt.Errorf("failed to update GitHub ID: %w", err)
			}
			user.GithubID = sql.NullString{String: githubIDStr, Valid: true}
		}
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// Check if user exists by GitHub ID
	user, err = database.GetUserByGithubID(h.db, githubIDStr)
	if err == nil {
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check user by GitHub ID: %w", err)
	}

	// Create new user (no password for OAuth users)
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

	newUser := &database.User{}
	err = h.db.QueryRow(
		query,
		id,
		email,
		"", // No password for OAuth users
		firstNameNull,
		lastNameNull,
		true, // Email verified by GitHub (if public)
		sql.NullString{}, // No Google ID for GitHub OAuth
		githubIDStr,
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
		&newUser.ID,
		&newUser.Email,
		&newUser.PasswordHash,
		&newUser.FirstName,
		&newUser.LastName,
		&newUser.EmailVerified,
		&newUser.GoogleID,
		&newUser.GithubID,
		&newUser.Rating,
		&newUser.PeakRating,
		&newUser.CurrentRank,
		&newUser.RankTier,
		&newUser.RankSubdivision,
		&newUser.GlobalRank,
		&newUser.TotalMatches,
		&newUser.Wins,
		&newUser.Losses,
		&newUser.LastRatingUpdate,
		&newUser.LastActivity,
		&newUser.DemotionProtectionUntil,
		&newUser.RatingDecayAppliedAt,
		&newUser.CreatedAt,
		&newUser.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return newUser, nil
}

// splitName splits a full name into first and last name
func splitName(name string) []string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return []string{""}
	}
	if len(parts) == 1 {
		return []string{parts[0]}
	}
	// Return first name and rest as last name
	return []string{parts[0], strings.Join(parts[1:], " ")}
}

