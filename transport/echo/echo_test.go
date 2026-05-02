package echo_test

import (
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
	spisqlecho "github.com/arten331/spisql/transport/echo"
)

func TestParse_FullQuery(t *testing.T) {
	e := echo.New()
	r := httptest.NewRequest(
		"GET",
		"/users?name=vasya&id[in]=1&id[in]=2&sort=-created_at&limit=20&offset=40",
		nil,
	)
	w := httptest.NewRecorder()
	c := e.NewContext(r, w)

	q := spisqlecho.Parse(c, parser.Options{})
	if len(q.Filters) != 2 {
		t.Fatalf("want 2 filters, got %d", len(q.Filters))
	}
	if len(q.Sorts) != 1 || q.Sorts[0].Direction != spisql.DirectionDesc {
		t.Fatalf("sorts=%+v", q.Sorts)
	}
	if q.Pagination.Limit != 20 || q.Pagination.Offset != 40 {
		t.Fatalf("pagination=%+v", q.Pagination)
	}
}
