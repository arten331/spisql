package parser_test

import (
	"net/url"
	"reflect"
	"testing"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
)

func mustParse(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query %q: %v", raw, err)
	}
	return v
}

func TestParse_BareEqualityIsEq(t *testing.T) {
	q := parser.Parse(mustParse(t, "status=active&email=foo@x.com"), parser.Options{})
	got := map[string]string{}
	for _, f := range q.Filters {
		if f.Operator != spisql.OpEqual || len(f.Values) != 1 {
			t.Fatalf("unexpected filter: %+v", f)
		}
		got[string(f.Key)] = f.Values[0]
	}
	want := map[string]string{"status": "active", "email": "foo@x.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestParse_BracketedComparison(t *testing.T) {
	q := parser.Parse(mustParse(t, "created_at[gte]=2024-01-01"), parser.Options{})
	if len(q.Filters) != 1 {
		t.Fatalf("want 1 filter, got %d", len(q.Filters))
	}
	want := &spisql.Filter{
		Key:      "created_at",
		Operator: spisql.OpGreaterThanEqual,
		Values:   []string{"2024-01-01"},
	}
	if !reflect.DeepEqual(q.Filters[0], want) {
		t.Fatalf("got %+v want %+v", q.Filters[0], want)
	}
}

func TestParse_CommaInValueSurvives(t *testing.T) {
	q := parser.Parse(mustParse(t, "city[eq]=Paris%2C%20France"), parser.Options{})
	if len(q.Filters) != 1 || q.Filters[0].Values[0] != "Paris, France" {
		t.Fatalf("comma not preserved: %+v", q.Filters)
	}
}

func TestParse_RepeatedKeyMultivalue(t *testing.T) {
	q := parser.Parse(mustParse(t, "id[in]=1&id[in]=2&id[in]=3"), parser.Options{})
	if len(q.Filters) != 1 || q.Filters[0].Operator != spisql.OpIn {
		t.Fatalf("filter=%+v", q.Filters)
	}
	if !reflect.DeepEqual(q.Filters[0].Values, []string{"1", "2", "3"}) {
		t.Fatalf("values=%v", q.Filters[0].Values)
	}
}

func TestParse_BetweenPreservesOrder(t *testing.T) {
	q := parser.Parse(
		mustParse(t, "created[between]=2024-01-01&created[between]=2024-12-31"),
		parser.Options{},
	)
	if len(q.Filters) != 1 || q.Filters[0].Operator != spisql.OpBetween {
		t.Fatalf("filter=%+v", q.Filters)
	}
	if !reflect.DeepEqual(q.Filters[0].Values, []string{"2024-01-01", "2024-12-31"}) {
		t.Fatalf("between order: %v", q.Filters[0].Values)
	}
}

func TestParse_NullaryOpHasNoValues(t *testing.T) {
	q := parser.Parse(mustParse(t, "email[isnull]="), parser.Options{})
	if len(q.Filters) != 1 || q.Filters[0].Operator != spisql.OpIsNull {
		t.Fatalf("filter=%+v", q.Filters)
	}
	if len(q.Filters[0].Values) != 0 {
		t.Fatalf("nullary should drop values, got %v", q.Filters[0].Values)
	}
}

func TestParse_SortDescAndAsc(t *testing.T) {
	q := parser.Parse(mustParse(t, "sort=-created_at,name"), parser.Options{})
	if len(q.Sorts) != 2 {
		t.Fatalf("want 2 sorts, got %d", len(q.Sorts))
	}
	if q.Sorts[0].Key != "created_at" || q.Sorts[0].Direction != spisql.DirectionDesc {
		t.Fatalf("sort[0]=%+v", q.Sorts[0])
	}
	if q.Sorts[1].Key != "name" || q.Sorts[1].Direction != spisql.DirectionAsc {
		t.Fatalf("sort[1]=%+v", q.Sorts[1])
	}
}

func TestParse_SortRepeatedKey(t *testing.T) {
	q := parser.Parse(mustParse(t, "sort=-created_at&sort=name"), parser.Options{})
	if len(q.Sorts) != 2 {
		t.Fatalf("want 2 sorts via repeat, got %d", len(q.Sorts))
	}
}

func TestParse_Pagination(t *testing.T) {
	q := parser.Parse(mustParse(t, "limit=25&offset=50"), parser.Options{})
	if q.Pagination.Limit != 25 || q.Pagination.Offset != 50 {
		t.Fatalf("pagination=%+v", q.Pagination)
	}
}

func TestParse_DefaultLimit(t *testing.T) {
	q := parser.Parse(mustParse(t, ""), parser.Options{})
	if q.Pagination.Limit != 10 {
		t.Fatalf("default limit should be 10, got %d", q.Pagination.Limit)
	}
}

func TestParse_LimitCappedByMax(t *testing.T) {
	q := parser.Parse(mustParse(t, "limit=999"), parser.Options{MaxLimit: 100})
	if q.Pagination.Limit != 100 {
		t.Fatalf("limit should be capped at 100, got %d", q.Pagination.Limit)
	}
}

func TestParse_FilterWhitelistDrops(t *testing.T) {
	q := parser.Parse(
		mustParse(t, "name=v&secret=x&secret[eq]=y"),
		parser.Options{Filter: parser.FilterOptions{"name": {}}},
	)
	if len(q.Filters) != 1 || q.Filters[0].Key != "name" {
		t.Fatalf("whitelist not applied: %+v", q.Filters)
	}
}

func TestParse_SortWhitelistDrops(t *testing.T) {
	q := parser.Parse(
		mustParse(t, "sort=-created_at,secret"),
		parser.Options{Sort: parser.SortOptions{"created_at": {}}},
	)
	if len(q.Sorts) != 1 || q.Sorts[0].Key != "created_at" {
		t.Fatalf("sort whitelist not applied: %+v", q.Sorts)
	}
}

func TestParse_ReservedKeysNotFilters(t *testing.T) {
	q := parser.Parse(mustParse(t, "limit=25&offset=50&sort=name"), parser.Options{})
	if len(q.Filters) != 0 {
		t.Fatalf("reserved keys leaked into filters: %+v", q.Filters)
	}
}

func TestParse_MalformedBracketKeyIgnored(t *testing.T) {
	q := parser.Parse(
		mustParse(t, "name[]=v&[eq]=v&col[op][extra]=v&good=ok"),
		parser.Options{},
	)
	if len(q.Filters) != 1 || q.Filters[0].Key != "good" {
		t.Fatalf("malformed not ignored: %+v", q.Filters)
	}
}

func TestParse_ColumnNamedLikeReservedKeyNeedsBrackets(t *testing.T) {
	q := parser.Parse(
		mustParse(t, "limit[eq]=42"),
		parser.Options{Filter: parser.FilterOptions{"limit": {}}},
	)
	if len(q.Filters) != 1 || q.Filters[0].Key != "limit" {
		t.Fatalf("bracket-form filter on 'limit' didn't pass: %+v", q.Filters)
	}
	if q.Filters[0].Values[0] != "42" {
		t.Fatalf("value: %v", q.Filters[0].Values)
	}
}
