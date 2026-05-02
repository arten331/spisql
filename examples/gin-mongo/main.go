// Example: gin-mongo demonstrates a production-ready list endpoint
// with MongoDB using proper pagination response structures.
//
// Features:
// - JSONResponseList with Total, Limit, Offset metadata
// - Full count query for pagination
// - Real MongoDB execution via mongo-driver
// - Comprehensive error handling
// - Query filtering and sorting
//
// Usage:
//  docker-compose up -d mongo
//  go run ./main.go
//  curl 'http://localhost:8080/products?name=laptop&sort=-created_at&limit=10&offset=0'
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	sposql "github.com/arten331/spisql"
	sqlmongo "github.com/arten331/spisql/adapter/mongo"
	"github.com/arten331/spisql/parser"
	sqlgin "github.com/arten331/spisql/transport/gin"
)

// Product represents a product document in MongoDB
type Product struct {
	ID        string    `json:"id" bson:"_id"`
	Name      string    `json:"name" bson:"name"`
	Category  string    `json:"category" bson:"category"`
	Price     float64   `json:"price" bson:"price"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

// JSONResponseList is the standard paginated response structure
type JSONResponseList[T any] struct {
	Total  uint64 `json:"total" example:"100" default:"0" binding:"min=0" minimum:"0"`
	Limit  uint64 `json:"limit" example:"10" default:"10" binding:"min=0" minimum:"0"`
	Offset uint64 `json:"offset" example:"20" default:"0" binding:"min=0" minimum:"0"`
	List   []T    `json:"list,omitempty"`
}

type AppContext struct {
	db *mongo.Database
}

func main() {
	// MongoDB connection
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}
	log.Println("✓ Connected to MongoDB")

	db := client.Database("demo")

	// Initialize database
	if err := initDB(ctx, db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("✓ Database schema initialized")

	appCtx := &AppContext{db: db}

	// Setup Gin router
	r := gin.Default()

	// List products endpoint
	r.GET("/products", appCtx.listProducts)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("✓ Server starting on :8080")
	log.Println("  GET /products?name=laptop&sort=-created_at&limit=10&offset=0")
	log.Println("  GET /health")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// listProducts handles the list products endpoint
//
//	GET /products
//	Query params:
//	  - name: filter by name (substring match)
//	  - category: filter by category (equality)
//	  - price[gte]: filter by minimum price
//	  - sort: sort order (e.g., -created_at,name)
//	  - limit: page size (default 10, max 100)
//	  - offset: pagination offset
func (a *AppContext) listProducts(c *gin.Context) {
	// Parse query with validation
	q := sqlgin.Parse(c, parser.Options{
		Filter: parser.FilterOptions{
			"name":     {},
			"category": {},
			"price":    {},
		},
		Sort: parser.SortOptions{
			"created_at": {},
			"name":       {},
			"price":      {},
		},
		DefaultLimit: 10,
		MaxLimit:     100,
	})

	// Field mapping from API to MongoDB fields
	mapping := sqlmongo.Mapping{
		"name":     "name",
		"category": "category",
		"price":    "price",
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	coll := a.db.Collection("products")

	// Get total count (before applying limit/offset)
	total, err := a.countProducts(ctx, coll, q, mapping)
	if err != nil {
		log.Printf("Count error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to count records",
		})
		return
	}

	// Build filter using adapter
	filter, err := sqlmongo.BuildFilter(q.Filters, mapping)
	if err != nil {
		log.Printf("Filter build error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to build filter",
		})
		return
	}

	// Build find options (sort, limit, offset)
	findOpts := options.Find()

	// Apply sorts
	if len(q.Sorts) > 0 {
		sortDoc := bson.D{}
		for _, sort := range q.Sorts {
			dir := int32(1)
			if sort.Desc {
				dir = -1
			}
			// Map field name from API to DB
			fieldName := mapping[sort.Key.Name]
			if fieldName == "" {
				fieldName = sort.Key.Name
			}
			sortDoc = append(sortDoc, bson.E{Key: fieldName, Value: dir})
		}
		findOpts.SetSort(sortDoc)
	}

	// Apply pagination
	findOpts.SetSkip(int64(q.Pagination.Offset))
	findOpts.SetLimit(int64(q.Pagination.Limit))

	// Execute query
	cursor, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		log.Printf("Query error: %v, Filter: %v", err, filter)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch records",
		})
		return
	}
	defer cursor.Close(ctx)

	// Decode results
	var products []Product
	if err := cursor.All(ctx, &products); err != nil {
		log.Printf("Decode error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decode records",
		})
		return
	}

	if products == nil {
		products = []Product{}
	}

	// Return paginated response
	response := JSONResponseList[Product]{
		Total:  total,
		Limit:  q.Pagination.Limit,
		Offset: q.Pagination.Offset,
		List:   products,
	}

	c.JSON(http.StatusOK, response)
}

// countProducts returns the total count of products matching the filters
func (a *AppContext) countProducts(ctx context.Context, coll *mongo.Collection, q *parser.Query, mapping sqlmongo.Mapping) (uint64, error) {
	// Build filter (same as data query, but without limit/offset)
	filter, err := sqlmongo.BuildFilter(q.Filters, mapping)
	if err != nil {
		return 0, err
	}

	count, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}
	return uint64(count), nil
}

// initDB creates collections and seeds sample data
func initDB(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection("products")

	// Check if data already exists
	count, err := coll.EstimatedDocumentCount(ctx)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Already seeded
	}

	// Seed sample data
	now := time.Now()
	docs := []interface{}{
		Product{
			ID:        "1",
			Name:      "MacBook Pro",
			Category:  "laptops",
			Price:     2499.99,
			CreatedAt: now.AddDate(0, -3, 0),
		},
		Product{
			ID:        "2",
			Name:      "Dell XPS 13",
			Category:  "laptops",
			Price:     1199.99,
			CreatedAt: now.AddDate(0, -2, 0),
		},
		Product{
			ID:        "3",
			Name:      "iPad Pro",
			Category:  "tablets",
			Price:     1099.99,
			CreatedAt: now.AddDate(0, -1, 0),
		},
		Product{
			ID:        "4",
			Name:      "AirPods Pro",
			Category:  "audio",
			Price:     249.99,
			CreatedAt: now.AddDate(0, 0, -10),
		},
		Product{
			ID:        "5",
			Name:      "Samsung Galaxy S24",
			Category:  "phones",
			Price:     999.99,
			CreatedAt: now.AddDate(0, 0, -5),
		},
	}

	_, err = coll.InsertMany(ctx, docs)
	return err
}

// BuildFilter is a helper to build MongoDB filter from filippters.
// In real code, use the adapter's BuildFilter function directly.
func BuildFilter(filters sposql.Filters, mapping sqlmongo.Mapping) (bson.M, error) {
	return sqlmongo.BuildFilter(filters, mapping)
}
