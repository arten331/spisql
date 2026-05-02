package gorm_test

import (
	"reflect"
	"testing"

	"github.com/arten331/spisql"
	spisqlgorm "github.com/arten331/spisql/adapter/gorm"
)

var mapping = spisqlgorm.Mapping{
	"name":   "u.name",
	"id":     "u.id",
	"status": "u.status",
}

func TestFilter_Equal(t *testing.T) {
	frag, args, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"},
	}, mapping)
	if !ok || frag != "u.name = ?" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
	if !reflect.DeepEqual(args, []any{"vasya"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_In(t *testing.T) {
	frag, args, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpIn, Values: []string{"1", "2", "3"},
	}, mapping)
	if !ok || frag != "u.id IN ?" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
	if !reflect.DeepEqual(args, []any{[]string{"1", "2", "3"}}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_NotIn(t *testing.T) {
	frag, _, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpNotIn, Values: []string{"1", "2"},
	}, mapping)
	if !ok || frag != "u.id NOT IN ?" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
}

func TestFilter_GreaterThanMulti(t *testing.T) {
	frag, args, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpGreaterThan, Values: []string{"10", "20"},
	}, mapping)
	if !ok || frag != "(u.id > ? OR u.id > ?)" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
	if !reflect.DeepEqual(args, []any{"10", "20"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_StartsWith(t *testing.T) {
	frag, args, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "name", Operator: spisql.OpStartsWith, Values: []string{"va"},
	}, mapping)
	if !ok || frag != "(u.name ILIKE ?)" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
	if !reflect.DeepEqual(args, []any{"va%"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_UnmappedColumn(t *testing.T) {
	_, _, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "secret", Operator: spisql.OpEqual, Values: []string{"x"},
	}, mapping)
	if ok {
		t.Fatal("unmapped column must yield ok=false")
	}
}

func TestSort(t *testing.T) {
	got := spisqlgorm.Sort(&spisql.Sort{Key: "name", Direction: spisql.DirectionDesc}, mapping)
	if got != "u.name DESC" {
		t.Fatalf("got=%q", got)
	}
	if spisqlgorm.Sort(&spisql.Sort{Key: "secret", Direction: spisql.DirectionAsc}, mapping) != "" {
		t.Fatal("unmapped sort must return empty string")
	}
}

func TestApply_NilQuery(t *testing.T) {
	// Apply with nil query must not panic and must return its db argument
	// untouched. We pass nil since Apply checks q == nil before touching db.
	got := spisqlgorm.Apply(nil, nil, mapping)
	if got != nil {
		t.Fatalf("nil query with nil db should return nil, got %v", got)
	}
}

func TestFilter_IsNull(t *testing.T) {
	frag, args, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "status", Operator: spisql.OpIsNull,
	}, mapping)
	if !ok || frag != "u.status IS NULL" || len(args) != 0 {
		t.Fatalf("frag=%q args=%v ok=%v", frag, args, ok)
	}
}

func TestFilter_IsNotNull(t *testing.T) {
	frag, args, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "status", Operator: spisql.OpIsNotNull,
	}, mapping)
	if !ok || frag != "u.status IS NOT NULL" || len(args) != 0 {
		t.Fatalf("frag=%q args=%v ok=%v", frag, args, ok)
	}
}

func TestFilter_Between(t *testing.T) {
	frag, args, ok := spisqlgorm.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpBetween, Values: []string{"1", "10"},
	}, mapping)
	if !ok || frag != "u.id BETWEEN ? AND ?" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
	if !reflect.DeepEqual(args, []any{"1", "10"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestGroup_Or(t *testing.T) {
	gr := &spisql.Group{
		Op: spisql.GroupOr,
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"petya"}},
		},
	}
	frag, args, ok := spisqlgorm.Group(gr, mapping)
	if !ok || frag != "((u.name = ?) OR (u.name = ?))" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
	if !reflect.DeepEqual(args, []any{"vasya", "petya"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestGroup_AllUnmappedDropped(t *testing.T) {
	gr := &spisql.Group{
		Op: spisql.GroupOr,
		Filters: spisql.Filters{
			{Key: "secret", Operator: spisql.OpEqual, Values: []string{"x"}},
		},
	}
	if _, _, ok := spisqlgorm.Group(gr, mapping); ok {
		t.Fatal("group with only unmapped filters must return ok=false")
	}
}
