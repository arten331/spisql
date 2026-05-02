// Example: gin-gorm demonstrates a production-ready list endpoint
// with PostgreSQL using GORM as the ORM and proper pagination response structures.
//
// Features:
// - JSONResponseList with Total, Limit, Offset metadata
// - Total count query for pagination
// - GORM for schema management and queries
// - Comprehensive error handling
//
// Usage:
//  docker-compose up -d postgres
//  go run ./main.go
//  curl 'http://localhost:8080/articles?status=published&sort=-created_at&limit=10'
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	sqlgorm "github.com/arten331/spisql/adapter/gorm"
	"github.com/arten331/spisql/parser"
	sqlgin "github.com/arten331/spisql/transport/gin"
)

// Article represents a blog article
type Article struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // published, draft
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JSONResponseList is the standard paginated response structure
type JSONResponseList[T any] struct {
	Total  uint64 `json:"total" example:"100" default:"0" binding:"min=0" minimum:"0"`
	Limit  uint64 `json:"limit" example:"10" default:"10" binding:"min=0" minimum:"0"`
	Offset uint64 `json:"offset" example:"20" default:"0" binding:"min=0" minimum:"0"`
	List   []T    `json:"list,omitempty"`
}

type AppContext struct {
	db *gorm.DB
}

func main() {
	// PostgreSQL connection
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:postgres@localhost:5432/demo?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("✓ Connected to database")

	// Auto-migrate schema
	if err := db.AutoMigrate(&Article{}); err != nil {
		log.Fatalf("Failed to migrate schema: %v", err)
	}
	log.Println("✓ Database schema initialized")

	// Seed sample data if needed
	seedData(db)

	appCtx := &AppContext{db: db}

	// Setup Gin router
	r := gin.Default()

	// List articles endpoint
	r.GET("/articles", appCtx.listArticles)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("✓ Server starting on :8080")
	log.Println("  GET /articles?status=published&sort=-created_at&limit=10")
	log.Println("  GET /health")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// listArticles handles the list articles endpoint
//
//	GET /articles
//	Query params:
//	  - title: filter by title (substring match, case-insensitive)
//	  - status: filter by status (published, draft)
//	  - author: filter by author name
//	  - created_at[gte]: filter by creation date
//	  - sort: sort order (e.g., -created_at,title)
//	  - limit: page size (default 10, max 100)
//	  - offset: pagination offset
func (a *AppContext) listArticles(c *gin.Context) {
	// Parse query with validation
	q := sqlgin.Parse(c, parser.Options{
		Filter: parser.FilterOptions{
			"title":      {},
			"status":     {},
			"author":     {},
			"created_at": {},
		},
		Sort: parser.SortOptions{
			"created_at": {},
			"title":      {},
			"author":     {},
		},
		DefaultLimit: 10,
		MaxLimit:     100,
	})

	// Field mapping from API to database columns
	mapping := sqlgorm.Mapping{
		"title":      "articles.title",
		"status":     "articles.status",
		"author":     "articles.author",
		"created_at": "articles.created_at",
	}

	// Get total count
	var total int64
	if err := a.db.Model(&Article{}).Scopes(filterScope(q, mapping)).Count(&total).Error; err != nil {
		log.Printf("Count error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to count records",
		})
		return
	}

	// Build and execute main query
	query := a.db.Model(&Article{})

	// Apply filters
	query = sqlgorm.Apply(query, q, mapping)

	// Fetch results
	var articles []Article
	if err := query.Error; err != nil {
		log.Printf("Query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch records",
		})
		return
	}

	if err := query.Find(&articles).Error; err != nil {
		log.Printf("Find error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch records",
		})
		return
	}

	if articles == nil {
		articles = []Article{}
	}

	// Return paginated response
	response := JSONResponseList[Article]{
		Total:  uint64(total),
		Limit:  q.Pagination.Limit,
		Offset: q.Pagination.Offset,
		List:   articles,
	}

	c.JSON(http.StatusOK, response)
}

// filterScope returns a GORM scope that applies only the filters (no pagination)
// This is used for the COUNT query
func filterScope(q *parser.Query, mapping sqlgorm.Mapping) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// Create a copy of the query without pagination for counting
		qCopy := *q
		qCopy.Pagination = parser.Pagination{}
		return sqlgorm.Apply(db, &qCopy, mapping)
	}
}

// seedData inserts sample articles if the table is empty
func seedData(db *gorm.DB) {
	var count int64
	db.Model(&Article{}).Count(&count)
	if count > 0 {
		return
	}

	now := time.Now()
	articles := []Article{
		{
			Title:     "Getting Started with Go",
			Status:    "published",
			Author:    "Alice",
			CreatedAt: now.AddDate(0, -3, 0),
		},
		{
			Title:     "Advanced Go Concurrency",
			Status:    "published",
			Author:    "Bob",
			CreatedAt: now.AddDate(0, -2, 0),
		},
		{
			Title:     "REST APIs with Gin",
			Status:    "published",
			Author:    "Charlie",
			CreatedAt: now.AddDate(0, -1, 0),
		},
		{
			Title:     "Database Design Patterns",
			Status:    "draft",
			Author:    "Diana",
			CreatedAt: now.AddDate(0, 0, -5),
		},
		{
			Title:     "Microservices Architecture",
			Status:    "published",
			Author:    "Eve",
			CreatedAt: now,
		},
	}

	for _, a := range articles {
		if err := db.Create(&a).Error; err != nil {
			log.Printf("Failed to seed data: %v", err)
		}
	}
}
