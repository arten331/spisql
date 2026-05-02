package raw_test

import (
	"reflect"
	"testing"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/adapter/raw"
)

var mapping = raw.Mapping{
	"name":   "u.name",
	"id":     "u.id",
	"status": "u.status",
}

func TestBuild_Equal(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.name = ?)" {
		t.Fatalf("Where = %q", got.Where)
	}
	if !reflect.DeepEqual(got.Args, []any{"vasya"}) {
		t.Fatalf("Args = %v", got.Args)
	}
}

func TestBuild_In(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpIn, Values: []string{"1", "2", "3"}},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.id IN (?, ?, ?))" {
		t.Fatalf("Where = %q", got.Where)
	}
	if !reflect.DeepEqual(got.Args, []any{"1", "2", "3"}) {
		t.Fatalf("Args = %v", got.Args)
	}
}

func TestBuild_MultiClauseAndSort(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpEqual, Values: []string{"active"}},
			{Key: "id", Operator: spisql.OpGreaterThan, Values: []string{"100"}},
		},
		Sorts: spisql.Sorts{
			{Key: "name", Direction: spisql.DirectionAsc},
			{Key: "id", Direction: spisql.DirectionDesc},
		},
		Pagination: spisql.Pagination{Limit: 25, Offset: 50},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.status = ?) AND (u.id > ?)" {
		t.Fatalf("Where = %q", got.Where)
	}
	if got.OrderBy != "u.name ASC, u.id DESC" {
		t.Fatalf("OrderBy = %q", got.OrderBy)
	}
	if got.Limit != 25 || got.Offset != 50 {
		t.Fatalf("pagination: %d/%d", got.Limit, got.Offset)
	}
}

func TestBuild_UnmappedSilentlyDropped(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"x"}},
			{Key: "secret", Operator: spisql.OpEqual, Values: []string{"y"}},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.name = ?)" {
		t.Fatalf("expected only mapped clause, got %q", got.Where)
	}
}

func TestBuild_LikeOps(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpStartsWith, Values: []string{"va"}},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.name LIKE ?)" {
		t.Fatalf("Where = %q", got.Where)
	}
	if !reflect.DeepEqual(got.Args, []any{"va%"}) {
		t.Fatalf("Args = %v", got.Args)
	}
}

func TestDollar(t *testing.T) {
	in := "a = ? AND b IN (?, ?, ?)"
	want := "a = $1 AND b IN ($2, $3, $4)"
	if got := raw.Dollar(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuild_Nil(t *testing.T) {
	got := raw.Build(nil, mapping)
	if got.Where != "" || len(got.Args) != 0 || got.OrderBy != "" {
		t.Fatalf("nil query produced non-empty result: %+v", got)
	}
}

func TestBuild_IsNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpIsNull},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.status IS NULL)" {
		t.Fatalf("Where=%q", got.Where)
	}
	if len(got.Args) != 0 {
		t.Fatalf("Args=%v", got.Args)
	}
}

func TestBuild_IsNotNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpIsNotNull},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.status IS NOT NULL)" {
		t.Fatalf("Where=%q", got.Where)
	}
}

func TestBuild_Between(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"1", "10"}},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "(u.id BETWEEN ? AND ?)" {
		t.Fatalf("Where=%q", got.Where)
	}
	if !reflect.DeepEqual(got.Args, []any{"1", "10"}) {
		t.Fatalf("Args=%v", got.Args)
	}
}

func TestBuild_BetweenWrongArity_Dropped(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"1"}},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "" {
		t.Fatalf("between with 1 value must be dropped, got Where=%q", got.Where)
	}
}

func TestBuild_OrGroup(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpEqual, Values: []string{"active"}},
		},
		Groups: spisql.Groups{
			{
				Op: spisql.GroupOr,
				Filters: spisql.Filters{
					{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
					{Key: "name", Operator: spisql.OpEqual, Values: []string{"petya"}},
				},
			},
		},
	}
	got := raw.Build(q, mapping)
	want := "(u.status = ?) AND ((u.name = ?) OR (u.name = ?))"
	if got.Where != want {
		t.Fatalf("Where=%q want=%q", got.Where, want)
	}
	if !reflect.DeepEqual(got.Args, []any{"active", "vasya", "petya"}) {
		t.Fatalf("Args=%v", got.Args)
	}
}

func TestBuild_AndGroup(t *testing.T) {
	q := &spisql.Query{
		Groups: spisql.Groups{
			{
				Op: spisql.GroupAnd,
				Filters: spisql.Filters{
					{Key: "name", Operator: spisql.OpEqual, Values: []string{"a"}},
					{Key: "id", Operator: spisql.OpGreaterThan, Values: []string{"1"}},
				},
			},
		},
	}
	got := raw.Build(q, mapping)
	if got.Where != "((u.name = ?) AND (u.id > ?))" {
		t.Fatalf("Where=%q", got.Where)
	}
}
