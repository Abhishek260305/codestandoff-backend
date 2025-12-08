# CodeStandoff Backend

GraphQL backend server for CodeStandoff 2.0 built with Go.

## Overview

This is the main backend application providing:
- GraphQL API
- User management
- Problem management
- Match management
- PostgreSQL database integration

## Tech Stack

- Go 1.21+
- GraphQL (gqlgen)
- PostgreSQL
- lib/pq (PostgreSQL driver)

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 12+

### Installation

1. **Set up PostgreSQL database**
   ```bash
   createdb codestandoff
   ```

2. **Configure environment variables**
   Create a `.env` file:
   ```
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=postgres
   DB_PASSWORD=postgres
   DB_NAME=codestandoff
   PORT=8080
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Generate GraphQL code** (if schema changes)
   ```bash
   go run github.com/99designs/gqlgen generate
   ```

### Running the Server

```bash
go run main.go
```

The GraphQL playground will be available at http://localhost:8080/

### API Endpoints

- **GraphQL Playground**: http://localhost:8080/
- **GraphQL API**: http://localhost:8080/query

## GraphQL Schema

### Types
- `User`: User information
- `Problem`: Coding problems
- `Match`: 1v1 matches

### Queries
- `users`: Get all users
- `user(id)`: Get user by ID
- `problems`: Get all problems
- `problem(id)`: Get problem by ID
- `matches`: Get all matches
- `match(id)`: Get match by ID

### Mutations
- `createUser(email, username)`: Create a new user
- `createProblem(title, description, difficulty)`: Create a new problem
- `createMatch(problemId)`: Create a new match

## Development

### Project Structure

```
backend/
├── graph/
│   ├── schema.graphqls    # GraphQL schema definition
│   ├── resolver.go        # GraphQL resolvers
│   ├── generated/         # Generated GraphQL code
│   └── model/             # Generated models
├── internal/
│   └── database/         # Database connection and utilities
└── main.go               # Application entry point
```

## Repository

https://github.com/Abhishek260305/codestandoff-backend

