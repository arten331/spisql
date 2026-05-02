package nethttp_test

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
	spisqlhttp "github.com/arten331/spisql/transport/nethttp"
)

func TestParse_FullQuery(t *testing.T) {
	r := httptest.NewRequest(
		"GET",
		"/users?name=vasya&id[in]=1&id[in]=2&id[in]=3&sort=-created_at&limit=20&offset=40",
		nil,
	)
	q := spisqlhttp.Parse(r, parser.Options{})
	if len(q.Filters) != 2 {
		t.Fatalf("want 2 filters, got %d", len(q.Filters))
	}
	got := map[string][]string{}
	for _, f := range q.Filters {
		got[string(f.Key)] = f.Values
	}
	if !reflect.DeepEqual(got["name"], []string{"vasya"}) {
		t.Fatalf("name filter: %v", got["name"])
	}
	if !reflect.DeepEqual(got["id"], []string{"1", "2", "3"}) {
		t.Fatalf("id filter: %v", got["id"])
	}
	if len(q.Sorts) != 1 || q.Sorts[0].Direction != spisql.DirectionDesc {
		t.Fatalf("sorts=%+v", q.Sorts)
	}
	if q.Pagination.Limit != 20 || q.Pagination.Offset != 40 {
		t.Fatalf("pagination=%+v", q.Pagination)
	}
}

func TestParse_Empty(t *testing.T) {
	r := httptest.NewRequest("GET", "/users", nil)
	q := spisqlhttp.Parse(r, parser.Options{})
	if len(q.Filters) != 0 || len(q.Sorts) != 0 {
		t.Fatalf("want empty, got %+v", q)
	}
	if q.Pagination.Limit != 10 {
		t.Fatalf("default limit=10, got %d", q.Pagination.Limit)
	}
}

func TestParse_WhitelistApplied(t *testing.T) {
	r := httptest.NewRequest("GET", "/users?name=x&secret=y", nil)
	q := spisqlhttp.Parse(r, parser.Options{
		Filter: parser.FilterOptions{"name": {}},
	})
	if len(q.Filters) != 1 || q.Filters[0].Key != "name" {
		t.Fatalf("whitelist failed: %+v", q.Filters)
	}
}
