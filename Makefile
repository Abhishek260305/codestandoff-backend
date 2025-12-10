all: mod generate

mod:
	go mod tidy

generate:
	go run github.com/99designs/gqlgen generate

graphql:
	go run github.com/99designs/gqlgen generate

build:
	go build ./main.go

run:
	go run ./main.go

test:
	go test ./... -coverprofile coverage.out
	go tool cover -func coverage.out

.PHONY: all mod generate graphql build run test

