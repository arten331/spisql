// custom-builder shows how to plug spisql into a service that doesn't use
// any of the bundled adapters. The handler parses the request via the
// nethttp transport, hands the *spisql.Query to a hand-written builder
// (see builder.go), and prints the SQL that would be executed.
//
// Run:
//
//	go run .
//	curl 'http://localhost:8080/users?name[substringof]=ann&created_at[gte]=2024-01-01&sort=-created_at&limit=20'
//	curl 'http://localhost:8080/users?email[in]=a@x.com&email[in]=b@x.com'
//	curl 'http://localhost:8080/users?email[isnotnull]=&sort=name'
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/arten331/spisql/parser"
	spisqlhttp "github.com/arten331/spisql/transport/nethttp"
)

var mapping = Mapping{
	"name":       "u.name",
	"email":      "u.email",
	"created_at": "u.created_at",
}

var opts = parser.Options{
	Filter: parser.FilterOptions{
		"name":       {},
		"email":      {},
		"created_at": {},
	},
	Sort: parser.SortOptions{
		"created_at": {},
		"name":       {},
	},
	DefaultLimit: 25,
	MaxLimit:     200,
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", listUsers)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	q := spisqlhttp.Parse(r, opts)
	res := Build(q, mapping)

	query := "SELECT u.id, u.name, u.email FROM users u"
	if res.Where != "" {
		query += " WHERE " + res.Where
	}
	if res.OrderBy != "" {
		query += " ORDER BY " + res.OrderBy
	}
	if res.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", res.Limit)
	}
	if res.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", res.Offset)
	}

	// Convert ? → $N for Postgres. Skip this line for MySQL/SQLite.
	query = dollar(query)

	fmt.Fprintf(w, "SQL : %s\nargs: %v\n", query, res.Args)
}
