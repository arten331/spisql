package nethttp

import (
	"net/http"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
)

func Parse(r *http.Request, opts parser.Options) *spisql.Query {
	return parser.Parse(r.URL.Query(), opts)
}
