// Example: gin-postgres demonstrates a production-ready list endpoint
// with pagination, filtering, sorting, and proper response structures.
//
// Features:
// - JSONResponseList with Total, Limit, Offset metadata
// - Full count query for pagination
// - Real database execution (pgx)
// - Comprehensive error handling
// - Swagger-compatible struct tags
//
// Usage:
//  docker-compose up -d postgres
//  go run ./main.go
//  curl 'http://localhost:8080/users?name=john&created_at[gte]=2024-01-01&sort=-created_at&limit=10&offset=0'
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	spisqlsq "github.com/arten331/spisql/adapter/squirrel"
	"github.com/arten331/spisql/parser"
	spisqlgin "github.com/arten331/spisql/transport/gin"
)

// User represents a user record
type User struct {
	ID        int64     `json:"id" example:"123"`
	Name      string    `json:"name" example:"John Doe"`
	Email     string    `json:"email" example:"john@example.com"`
	CreatedAt time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// JSONResponseList is the standard paginated response structure
// with metadata for the client to build pagination UI
type JSONResponseList[T any] struct {
	Total  uint64 `json:"total" example:"100" default:"0" binding:"min=0" minimum:"0"`
	Limit  uint64 `json:"limit" example:"10" default:"10" binding:"min=0" minimum:"0"`
	Offset uint64 `json:"offset" example:"20" default:"0" binding:"min=0" minimum:"0"`
	List   []T    `json:"list,omitempty" swaggertype:"object"`
}

// AppContext holds shared dependencies
type AppContext struct {
	db *pgxpool.Pool
}

func main() {
	// Parse database URL from environment or use default
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:postgres@localhost:5432/demo?sslmode=disable"
	}

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("✓ Connected to database")

	// Initialize database schema
	if err := initDB(ctx, db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("✓ Database schema initialized")

	appCtx := &AppContext{db: db}

	// Setup Gin router
	r := gin.Default()

	// List users endpoint
	r.GET("/users", appCtx.listUsers)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("✓ Server starting on :8080")
	log.Println("  GET /users?name=john&sort=-created_at&limit=10&offset=0")
	log.Println("  GET /health")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// listUsers handles the list users endpoint
//
//	GET /users
//	Query params:
//	  - name: filter by name (substring match)
//	  - email: filter by email (equality)
//	  - created_at[gte]: filter by creation date
//	  - sort: sort order (e.g., -created_at,name)
//	  - limit: page size (default 10, max 100)
//	  - offset: pagination offset
func (a *AppContext) listUsers(c *gin.Context) {
	// Parse query with validation
	q := spisqlgin.Parse(c, parser.Options{
		Filter: parser.FilterOptions{
			"name":       {},
			"email":      {},
			"created_at": {},
		},
		Sort: parser.SortOptions{
			"created_at": {},
			"name":       {},
		},
		DefaultLimit: 10,
		MaxLimit:     100,
	})

	// Field mapping from API to database columns
	mapping := spisqlsq.Mapping{
		"name":       "u.name",
		"email":      "u.email",
		"created_at": "u.created_at",
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get total count (before applying limit/offset)
	total, err := a.countUsers(ctx, q, mapping)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to count records",
		})
		return
	}

	// Build and execute the main query
	builder := sq.Select("u.id", "u.name", "u.email", "u.created_at").
		From("users u").
		PlaceholderFormat(sq.Dollar)

	// Apply filters, sorts, and pagination
	sql, args, err := spisqlsq.Apply(builder, q, mapping).ToSql()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to build query",
		})
		return
	}

	// Execute query
	rows, err := a.db.Query(ctx, sql, args...)
	if err != nil {
		log.Printf("Query error: %v\nSQL: %s\nArgs: %v", err, sql, args)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch records",
		})
		return
	}
	defer rows.Close()

	// Scan results
	users := make([]User, 0, q.Pagination.Limit)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			log.Printf("Scan error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to scan record",
			})
			return
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Rows error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "database error",
		})
		return
	}

	// Return paginated response
	response := JSONResponseList[User]{
		Total:  total,
		Limit:  q.Pagination.Limit,
		Offset: q.Pagination.Offset,
		List:   users,
	}

	c.JSON(http.StatusOK, response)
}

// countUsers returns the total count of users matching the filters
func (a *AppContext) countUsers(ctx context.Context, q *parser.Query, mapping spisqlsq.Mapping) (uint64, error) {
	// Build count query using the same filters but without limit/offset
	builder := sq.Select("COUNT(*)").
		From("users u").
		PlaceholderFormat(sq.Dollar)

	// Note: We apply filters but create a copy of q without pagination
	qCopy := *q
	qCopy.Pagination = parser.Pagination{} // Clear pagination for count

	sql, args, err := spisqlsq.Apply(builder, &qCopy, mapping).ToSql()
	if err != nil {
		return 0, err
	}

	var count uint64
	if err := a.db.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// initDB creates the schema and seeds sample data
func initDB(ctx context.Context, db *pgxpool.Pool) error {
	// Create table
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Exec(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Check if data already exists
	var count int64
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return nil // Already seeded
	}

	// Seed sample data
	now := time.Now()
	sampleUsers := []struct {
		name  string
		email string
		date  time.Time
	}{
		{"Alice Johnson", "alice@example.com", now.AddDate(0, -3, 0)},
		{"Bob Smith", "bob@example.com", now.AddDate(0, -2, -15)},
		{"Charlie Brown", "charlie@example.com", now.AddDate(0, -2, 0)},
		{"Diana Prince", "diana@example.com", now.AddDate(0, -1, -30)},
		{"Eve Wilson", "eve@example.com", now.AddDate(0, -1, 0)},
		{"Frank Miller", "frank@example.com", now.AddDate(0, 0, -20)},
		{"Grace Lee", "grace@example.com", now.AddDate(0, 0, -10)},
		{"Henry Davis", "henry@example.com", now.AddDate(0, 0, -5)},
		{"Iris Chen", "iris@example.com", now.AddDate(0, 0, -2)},
		{"Jack Thompson", "jack@example.com", now},
	}

	for _, u := range sampleUsers {
		_, err := db.Exec(ctx,
			"INSERT INTO users (name, email, created_at) VALUES ($1, $2, $3)",
			u.name, u.email, u.date,
		)
		if err != nil {
			return fmt.Errorf("failed to seed data: %w", err)
		}
	}

	return nil
}

// Helper: Print query as JSON for debugging
func printQuery(q *parser.Query) {
	b, _ := json.MarshalIndent(q, "", "  ")
	fmt.Println(string(b))
}
