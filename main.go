package main

import (
	"log"
	"net/http"
	"os"

	"codestandoff/backend/graph"
	"codestandoff/backend/graph/generated"
	"codestandoff/backend/internal/auth"
	"codestandoff/backend/internal/database"
	"codestandoff/backend/internal/oauth"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/rs/cors"
)

const defaultPort = "8080"

// cookieResponseWriter wraps the ResponseWriter to capture cookie setting
type cookieResponseWriter struct {
	http.ResponseWriter
	cookies []*http.Cookie
}

func (w *cookieResponseWriter) SetCookie(cookie *http.Cookie) {
	w.cookies = append(w.cookies, cookie)
	http.SetCookie(w.ResponseWriter, cookie)
}

// responseWriterHandler wraps the GraphQL handler to inject ResponseWriter into context
type responseWriterHandler struct {
	handler http.Handler
}

func (h *responseWriterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Wrap ResponseWriter to capture cookie setting
	cookieWriter := &cookieResponseWriter{
		ResponseWriter: w,
		cookies:        make([]*http.Cookie, 0),
	}

	ctx := r.Context()
	ctx = graph.WithResponseWriter(ctx, cookieWriter)
	ctx = graph.WithRequest(ctx, r)
	r = r.WithContext(ctx)
	h.handler.ServeHTTP(cookieWriter, r)
}

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

	// Initialize GraphQL resolver
	resolver := graph.NewResolver(db)

	// Create GraphQL server
	srv := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{
				Resolvers: resolver,
			},
		),
	)

	// Add extension to provide ResponseWriter access
	srv.Use(graph.ResponseWriterExtension{})

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

	// Create a handler that wraps both CORS and injects ResponseWriter
	// The order matters: CORS first, then our wrapper
	corsHandler := c.Handler(srv)
	wrappedHandler := &responseWriterHandler{
		handler: corsHandler,
	}

	// GraphQL handler with CORS and ResponseWriter injection
	graphqlHandler := wrappedHandler
	playgroundHandler := c.Handler(playground.Handler("GraphQL playground", "/query"))

	// OAuth routes (with CORS) - register these BEFORE the root handler
	googleAuthHandler := c.Handler(http.HandlerFunc(oauthHandler.GoogleAuth))
	googleCallbackHandler := c.Handler(http.HandlerFunc(oauthHandler.GoogleCallback))
	githubAuthHandler := c.Handler(http.HandlerFunc(oauthHandler.GitHubAuth))
	githubCallbackHandler := c.Handler(http.HandlerFunc(oauthHandler.GitHubCallback))

	// Register OAuth routes first (more specific routes should be registered before catch-all)
	http.Handle("/auth/google", googleAuthHandler)
	http.Handle("/auth/google/callback", googleCallbackHandler)
	http.Handle("/auth/github", githubAuthHandler)
	http.Handle("/auth/github/callback", githubCallbackHandler)

	// Register GraphQL routes
	http.Handle("/query", graphqlHandler)

	// Register root handler last (catch-all)
	http.Handle("/", playgroundHandler)

	log.Printf("Connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
