package goqu_test

import (
	"reflect"
	"testing"

	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"

	"github.com/arten331/spisql"
	spisqlgoqu "github.com/arten331/spisql/adapter/goqu"
)

var mapping = spisqlgoqu.Mapping{
	"name":   "u.name",
	"id":     "u.id",
	"status": "u.status",
}

func newDS() *g.SelectDataset {
	return g.Dialect("postgres").From("users")
}

func assertSQL(t *testing.T, gotSQL, wantSQL string, gotArgs, wantArgs []any) {
	t.Helper()
	if gotSQL != wantSQL {
		t.Fatalf("\n got: %s\nwant: %s", gotSQL, wantSQL)
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("\n got args: %v\nwant args: %v", gotArgs, wantArgs)
	}
}

func TestApply_Equal(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
		},
	}
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT * FROM "users" WHERE ("u"."name" = $1)`, args, []any{"vasya"})
}

func TestApply_In(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpIn, Values: []string{"1", "2", "3"}},
		},
	}
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT * FROM "users" WHERE ("u"."id" IN ($1, $2, $3))`, args, []any{"1", "2", "3"})
}

func TestApply_GreaterThanOrChain(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpGreaterThan, Values: []string{"100", "200"}},
		},
	}
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql,
		`SELECT * FROM "users" WHERE (("u"."id" > $1) OR ("u"."id" > $2))`,
		args, []any{"100", "200"})
}

func TestApply_OrderLimitOffset(t *testing.T) {
	q := &spisql.Query{
		Sorts: spisql.Sorts{
			{Key: "name", Direction: spisql.DirectionAsc},
			{Key: "id", Direction: spisql.DirectionDesc},
		},
		Pagination: spisql.Pagination{Limit: 25, Offset: 50},
	}
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	want := `SELECT * FROM "users" ORDER BY "u"."name" ASC, "u"."id" DESC LIMIT 25 OFFSET 50`
	if sql != want {
		t.Fatalf("\n got: %s\nwant: %s", sql, want)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %v", args)
	}
}

func TestApply_StartsWith(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpStartsWith, Values: []string{"va"}},
		},
	}
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql, `SELECT * FROM "users" WHERE ("u"."name" ILIKE $1)`, args, []any{"va%"})
}

func TestApply_UnmappedDropped(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "secret", Operator: spisql.OpEqual, Values: []string{"x"}},
		},
	}
	sql, _, err := spisqlgoqu.Apply(newDS(), q, mapping).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	if sql != `SELECT * FROM "users"` {
		t.Fatalf("expected unmodified select, got: %s", sql)
	}
}

func TestApply_NilQuery(t *testing.T) {
	sql, _, err := spisqlgoqu.Apply(newDS(), nil, mapping).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	if sql != `SELECT * FROM "users"` {
		t.Fatalf("nil query mutated builder: %s", sql)
	}
}

func TestApply_IsNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpIsNull},
		},
	}
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	if sql != `SELECT * FROM "users" WHERE ("u"."status" IS NULL)` {
		t.Fatalf("sql=%q", sql)
	}
	if len(args) != 0 {
		t.Fatalf("args=%v", args)
	}
}

func TestApply_IsNotNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpIsNotNull},
		},
	}
	sql, _, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	if sql != `SELECT * FROM "users" WHERE ("u"."status" IS NOT NULL)` {
		t.Fatalf("sql=%q", sql)
	}
}

func TestApply_Between(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"1", "10"}},
		},
	}
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	assertSQL(t, sql,
		`SELECT * FROM "users" WHERE ("u"."id" BETWEEN $1 AND $2)`,
		args, []any{"1", "10"})
}

func TestApply_OrGroup(t *testing.T) {
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
	sql, args, err := spisqlgoqu.Apply(newDS(), q, mapping).Prepared(true).ToSQL()
	if err != nil {
		t.Fatal(err)
	}
	want := `SELECT * FROM "users" WHERE (("u"."status" = $1) AND (("u"."name" = $2) OR ("u"."name" = $3)))`
	if sql != want {
		t.Fatalf("\n got: %s\nwant: %s", sql, want)
	}
	if !reflect.DeepEqual(args, []any{"active", "vasya", "petya"}) {
		t.Fatalf("args=%v", args)
	}
}
