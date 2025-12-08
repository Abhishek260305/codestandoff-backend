package main

import (
	"log"
	"net/http"
	"os"

	"codestandoff/backend/graph"
	"codestandoff/backend/graph/generated"
	"codestandoff/backend/internal/database"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
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
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Continuing without database connection...")
	} else {
		defer db.Close()
		log.Println("Database connection established")
	}

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

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("Connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

