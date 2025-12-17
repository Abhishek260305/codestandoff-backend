package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"codestandoff/backend/app/controllers"
	"codestandoff/backend/app/workflow"
	"codestandoff/backend/graph"
	"codestandoff/backend/internal/auth"
	"codestandoff/backend/internal/database"
	"codestandoff/backend/internal/oauth"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/rs/cors"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Initialize database connection
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Database connection established")

	// Initialize OAuth providers
	auth.InitGoogleOAuth()
	auth.InitGitHubOAuth()

	// Initialize OAuth handler
	oauthHandler := oauth.NewHandler(db)

	// Initialize controller
	controller := controllers.NewPCDGraphQLController(controllers.PCDGraphQLControllerDeps{
		DB: db,
	})

	// Initialize workflow
	wf := workflow.NewPCDGraphQLService(workflow.PCDGraphQLServiceDeps{
		Controller: controller,
	})

	// Initialize GraphQL resolver
	resolver := &graph.Resolver{
		Workflow: wf,
	}

	// Create GraphQL server
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000", // host-ui
			"http://localhost:3001", // dashboard-ui
			"http://localhost:3002", // training-ui
			"http://localhost:3003", // 1v1-ui
			"http://localhost:3004", // playground-ui
			"http://localhost:3005", // signup-builder-ui
			"http://localhost:3006", // marketing-ui
		},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	// Middleware to inject request/response into context
	graphqlHandler := c.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, controllers.ResponseWriterKey, w)
		ctx = context.WithValue(ctx, controllers.RequestKey, r)
		r = r.WithContext(ctx)
		srv.ServeHTTP(w, r)
	}))

	playgroundHandler := c.Handler(playground.Handler("GraphQL playground", "/query"))

	// OAuth routes
	googleAuthHandler := c.Handler(http.HandlerFunc(oauthHandler.GoogleAuth))
	googleCallbackHandler := c.Handler(http.HandlerFunc(oauthHandler.GoogleCallback))
	githubAuthHandler := c.Handler(http.HandlerFunc(oauthHandler.GitHubAuth))
	githubCallbackHandler := c.Handler(http.HandlerFunc(oauthHandler.GitHubCallback))

	// Register routes
	http.Handle("/auth/google", googleAuthHandler)
	http.Handle("/auth/google/callback", googleCallbackHandler)
	http.Handle("/auth/github", githubAuthHandler)
	http.Handle("/auth/github/callback", githubCallbackHandler)
	http.Handle("/query", graphqlHandler)
	http.Handle("/", playgroundHandler)

	log.Printf("Connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
