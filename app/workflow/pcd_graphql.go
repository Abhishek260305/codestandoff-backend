package workflow

import (
	"context"

	"codestandoff/backend/app/controllers"
	"codestandoff/backend/graph/model"
)

// PCDGraphQLServiceServer is the main workflow interface for GraphQL resolvers
// Named to match plg-crm-dashboard architecture
type PCDGraphQLServiceServer interface {
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

// PCDGraphQLServiceDeps contains dependencies for the workflow
type PCDGraphQLServiceDeps struct {
	Controller controllers.PCDGraphQLController
}

type pcdGraphQLServiceImpl struct {
	deps PCDGraphQLServiceDeps
}

// NewPCDGraphQLService creates a new PCDGraphQLServiceServer
func NewPCDGraphQLService(deps PCDGraphQLServiceDeps) PCDGraphQLServiceServer {
	return &pcdGraphQLServiceImpl{
		deps: deps,
	}
}

// GetQuestions fetches questions through the controller
func (impl *pcdGraphQLServiceImpl) GetQuestions(ctx context.Context, input model.GetQuestionsRequest) (*model.GetQuestionsResponse, error) {
	return impl.deps.Controller.GetQuestions(ctx, input)
}

// Me returns the current authenticated user
func (impl *pcdGraphQLServiceImpl) Me(ctx context.Context) (*model.User, error) {
	return impl.deps.Controller.Me(ctx)
}

// Users returns all users
func (impl *pcdGraphQLServiceImpl) Users(ctx context.Context) ([]*model.User, error) {
	return impl.deps.Controller.Users(ctx)
}

// User returns a user by ID
func (impl *pcdGraphQLServiceImpl) User(ctx context.Context, id string) (*model.User, error) {
	return impl.deps.Controller.User(ctx, id)
}

// Signup creates a new user
func (impl *pcdGraphQLServiceImpl) Signup(ctx context.Context, email, password string, firstName, lastName *string) (*model.AuthPayload, error) {
	return impl.deps.Controller.Signup(ctx, email, password, firstName, lastName)
}

// Login authenticates a user
func (impl *pcdGraphQLServiceImpl) Login(ctx context.Context, email, password string) (*model.AuthPayload, error) {
	return impl.deps.Controller.Login(ctx, email, password)
}

// Logout logs out the current user
func (impl *pcdGraphQLServiceImpl) Logout(ctx context.Context) (bool, error) {
	return impl.deps.Controller.Logout(ctx)
}

// Problems returns all problems
func (impl *pcdGraphQLServiceImpl) Problems(ctx context.Context) ([]*model.Problem, error) {
	return impl.deps.Controller.Problems(ctx)
}

// Problem returns a problem by ID
func (impl *pcdGraphQLServiceImpl) Problem(ctx context.Context, id string) (*model.Problem, error) {
	return impl.deps.Controller.Problem(ctx, id)
}

// Matches returns all matches
func (impl *pcdGraphQLServiceImpl) Matches(ctx context.Context) ([]*model.Match, error) {
	return impl.deps.Controller.Matches(ctx)
}

// Match returns a match by ID
func (impl *pcdGraphQLServiceImpl) Match(ctx context.Context, id string) (*model.Match, error) {
	return impl.deps.Controller.Match(ctx, id)
}

// CreateProblem creates a new problem
func (impl *pcdGraphQLServiceImpl) CreateProblem(ctx context.Context, title, description, difficulty string) (*model.Problem, error) {
	return impl.deps.Controller.CreateProblem(ctx, title, description, difficulty)
}

// CreateMatch creates a new match
func (impl *pcdGraphQLServiceImpl) CreateMatch(ctx context.Context, problemID string) (*model.Match, error) {
	return impl.deps.Controller.CreateMatch(ctx, problemID)
}

