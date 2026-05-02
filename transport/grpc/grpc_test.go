package grpc_test

import (
	"reflect"
	"testing"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
	spisqlgrpc "github.com/arten331/spisql/transport/grpc"
)

func TestParse_FullQuery(t *testing.T) {
	q := spisqlgrpc.Parse(
		"name=vasya&id[in]=1&id[in]=2&id[in]=3",
		"-created_at",
		20, 40,
		parser.Options{},
	)
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

func TestParse_DefaultsApplied(t *testing.T) {
	q := spisqlgrpc.Parse("", "", 0, 0, parser.Options{DefaultLimit: 25})
	if q.Pagination.Limit != 25 {
		t.Fatalf("default limit not applied: %d", q.Pagination.Limit)
	}
}

func TestParse_LimitClamped(t *testing.T) {
	q := spisqlgrpc.Parse("", "", 1000, 0, parser.Options{MaxLimit: 50})
	if q.Pagination.Limit != 50 {
		t.Fatalf("limit clamp: %d", q.Pagination.Limit)
	}
}

func TestParse_TypedFieldsOverrideEmbeddedInFilter(t *testing.T) {
	q := spisqlgrpc.Parse(
		"sort=name&limit=99",
		"-created_at",
		25, 0,
		parser.Options{},
	)
	if q.Pagination.Limit != 25 {
		t.Fatalf("typed limit should override embedded: %d", q.Pagination.Limit)
	}
	if len(q.Sorts) != 1 || q.Sorts[0].Key != "created_at" {
		t.Fatalf("typed sort should override: %+v", q.Sorts)
	}
}

func TestParse_FilterOnly(t *testing.T) {
	q := spisqlgrpc.Parse("status=active", "", 0, 0, parser.Options{})
	if len(q.Filters) != 1 || q.Filters[0].Key != "status" {
		t.Fatalf("filter not parsed: %+v", q.Filters)
	}
}

func TestParse_BracketEncodedInFilter(t *testing.T) {
	q := spisqlgrpc.Parse("id[in]=1&id[in]=2", "", 0, 0, parser.Options{})
	if len(q.Filters) != 1 || q.Filters[0].Operator != spisql.OpIn {
		t.Fatalf("filter=%+v", q.Filters)
	}
	if !reflect.DeepEqual(q.Filters[0].Values, []string{"1", "2"}) {
		t.Fatalf("values=%v", q.Filters[0].Values)
	}
}
