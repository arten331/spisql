// http-postgres demonstrates using spisql to translate list-endpoint
// queries into PostgreSQL via the squirrel adapter.
package main

import (
	"fmt"
	"log"
	"net/http"

	sq "github.com/Masterminds/squirrel"

	spisqlsq "github.com/arten331/spisql/adapter/squirrel"
	"github.com/arten331/spisql/parser"
	spisqlhttp "github.com/arten331/spisql/transport/nethttp"
)

var mapping = spisqlsq.Mapping{
	"id":         "u.id",
	"name":       "u.name",
	"email":      "u.email",
	"created_at": "u.created_at",
}

var opts = parser.Options{
	Filter:       parser.FilterOptions{"id": {}, "name": {}, "email": {}, "created_at": {}},
	Sort:         parser.SortOptions{"name": {}, "created_at": {}},
	DefaultLimit: 25,
	MaxLimit:     100,
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", listUsers)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// listUsers parses the HTTP query parameters using spisql, builds a SQL query,
// and responds with the generated SQL and arguments.
func listUsers(w http.ResponseWriter, r *http.Request) {
	q := spisqlhttp.Parse(r, opts)

	b := sq.Select("u.id", "u.name", "u.email", "u.created_at").
		From("users u").
		PlaceholderFormat(sq.Dollar)

	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "SQL : %s\nargs: %v\n", sql, args)
}
