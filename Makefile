.PHONY: generate build run clean

# Generate GraphQL code
generate:
	go mod download all
	go run github.com/99designs/gqlgen generate

# Build the application
build:
	go build -o bin/server ./main.go

# Run the application
run:
	go run main.go

# Clean generated files
clean:
	rm -rf bin/

# Generate and build
all: generate build
