package bun_test

import (
	"reflect"
	"testing"

	bunpkg "github.com/uptrace/bun"

	"github.com/arten331/spisql"
	spisqlbun "github.com/arten331/spisql/adapter/bun"
)

// inValuesType is the runtime type bun.In returns; we compare by type
// rather than value because the underlying struct is unexported.
var inValuesType = reflect.TypeOf(bunpkg.In([]any{}))

var mapping = spisqlbun.Mapping{
	"name":   "u.name",
	"id":     "u.id",
	"status": "u.status",
}

func TestFilter_Equal(t *testing.T) {
	frag, args, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"},
	}, mapping)
	if !ok {
		t.Fatal("ok=false")
	}
	if frag != "u.name = ?" {
		t.Fatalf("frag=%q", frag)
	}
	if !reflect.DeepEqual(args, []any{"vasya"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_In(t *testing.T) {
	frag, args, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpIn, Values: []string{"1", "2", "3"},
	}, mapping)
	if !ok {
		t.Fatal("ok=false")
	}
	if frag != "u.id IN (?)" {
		t.Fatalf("frag=%q", frag)
	}
	if len(args) != 1 {
		t.Fatalf("want a single bun.In wrapper, got %d args", len(args))
	}
	if reflect.TypeOf(args[0]) != inValuesType {
		t.Fatalf("expected %v, got %T", inValuesType, args[0])
	}
}

func TestFilter_NotIn(t *testing.T) {
	frag, _, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpNotIn, Values: []string{"1", "2"},
	}, mapping)
	if !ok || frag != "u.id NOT IN (?)" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
}

func TestFilter_GreaterThanMulti(t *testing.T) {
	frag, args, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpGreaterThan, Values: []string{"10", "20"},
	}, mapping)
	if !ok {
		t.Fatal("ok=false")
	}
	if frag != "(u.id > ? OR u.id > ?)" {
		t.Fatalf("frag=%q", frag)
	}
	if !reflect.DeepEqual(args, []any{"10", "20"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_StartsWith(t *testing.T) {
	frag, args, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "name", Operator: spisql.OpStartsWith, Values: []string{"va"},
	}, mapping)
	if !ok {
		t.Fatal("ok=false")
	}
	if frag != "(u.name ILIKE ?)" {
		t.Fatalf("frag=%q", frag)
	}
	if !reflect.DeepEqual(args, []any{"va%"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_UnmappedColumn(t *testing.T) {
	_, _, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "secret", Operator: spisql.OpEqual, Values: []string{"x"},
	}, mapping)
	if ok {
		t.Fatal("unmapped column must yield ok=false")
	}
}

func TestSort(t *testing.T) {
	got := spisqlbun.Sort(&spisql.Sort{Key: "name", Direction: spisql.DirectionDesc}, mapping)
	if got != "u.name DESC" {
		t.Fatalf("got=%q", got)
	}
	if spisqlbun.Sort(&spisql.Sort{Key: "secret", Direction: spisql.DirectionAsc}, mapping) != "" {
		t.Fatal("unmapped sort must return empty string")
	}
}

func TestFilter_IsNull(t *testing.T) {
	frag, args, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "status", Operator: spisql.OpIsNull,
	}, mapping)
	if !ok || frag != "u.status IS NULL" || len(args) != 0 {
		t.Fatalf("frag=%q args=%v ok=%v", frag, args, ok)
	}
}

func TestFilter_IsNotNull(t *testing.T) {
	frag, args, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "status", Operator: spisql.OpIsNotNull,
	}, mapping)
	if !ok || frag != "u.status IS NOT NULL" || len(args) != 0 {
		t.Fatalf("frag=%q args=%v ok=%v", frag, args, ok)
	}
}

func TestFilter_Between(t *testing.T) {
	frag, args, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpBetween, Values: []string{"1", "10"},
	}, mapping)
	if !ok || frag != "u.id BETWEEN ? AND ?" {
		t.Fatalf("frag=%q ok=%v", frag, ok)
	}
	if !reflect.DeepEqual(args, []any{"1", "10"}) {
		t.Fatalf("args=%v", args)
	}
}

func TestFilter_BetweenWrongArity(t *testing.T) {
	_, _, ok := spisqlbun.Filter(&spisql.Filter{
		Key: "id", Operator: spisql.OpBetween, Values: []string{"only-one"},
	}, mapping)
	if ok {
		t.Fatal("between with 1 value must yield ok=false")
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
	frag, args, ok := spisqlbun.Group(gr, mapping)
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
	if _, _, ok := spisqlbun.Group(gr, mapping); ok {
		t.Fatal("group with only unmapped filters must return ok=false")
	}
}
