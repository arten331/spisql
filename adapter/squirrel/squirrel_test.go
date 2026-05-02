package squirrel_test

import (
	"reflect"
	"testing"

	sq "github.com/Masterminds/squirrel"

	"github.com/arten331/spisql"
	spisqlsq "github.com/arten331/spisql/adapter/squirrel"
)

var mapping = spisqlsq.Mapping{
	"name":   "u.name",
	"id":     "u.id",
	"status": "u.status",
}

func TestApply_Equal(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u WHERE u.name = $1" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"vasya"}) {
		t.Fatalf("args = %v", args)
	}
}

func TestApply_In(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpIn, Values: []string{"1", "2", "3"}},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u WHERE u.id IN ($1,$2,$3)" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"1", "2", "3"}) {
		t.Fatalf("args = %v", args)
	}
}

func TestApply_OrderLimitOffset(t *testing.T) {
	q := &spisql.Query{
		Sorts: spisql.Sorts{
			{Key: "name", Direction: spisql.DirectionAsc},
			{Key: "id", Direction: spisql.DirectionDesc},
		},
		Pagination: spisql.Pagination{Limit: 25, Offset: 50},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, _, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT * FROM users u ORDER BY u.name ASC, u.id DESC LIMIT 25 OFFSET 50"
	if sql != want {
		t.Fatalf("\n got: %s\nwant: %s", sql, want)
	}
}

func TestApply_GreaterThanOrChain(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpGreaterThan, Values: []string{"100", "200"}},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u WHERE (u.id > $1 OR u.id > $2)" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"100", "200"}) {
		t.Fatalf("args = %v", args)
	}
}

func TestApply_StartsWith(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpStartsWith, Values: []string{"va"}},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u WHERE u.name ILIKE $1" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"va%"}) {
		t.Fatalf("args = %v", args)
	}
}

func TestApply_UnmappedDropped(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "secret", Operator: spisql.OpEqual, Values: []string{"x"}},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, _, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u" {
		t.Fatalf("expected no WHERE, got: %s", sql)
	}
}

func TestApply_NilQuery(t *testing.T) {
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, _, err := spisqlsq.Apply(b, nil, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u" {
		t.Fatalf("nil query mutated builder: %s", sql)
	}
}

func TestApply_IsNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpIsNull},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u WHERE u.status IS NULL" {
		t.Fatalf("sql = %q", sql)
	}
	if len(args) != 0 {
		t.Fatalf("args = %v", args)
	}
}

func TestApply_IsNotNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpIsNotNull},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, _, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u WHERE u.status IS NOT NULL" {
		t.Fatalf("sql = %q", sql)
	}
}

func TestApply_Between(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"1", "10"}},
		},
	}
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SELECT * FROM users u WHERE u.id BETWEEN $1 AND $2" {
		t.Fatalf("sql = %q", sql)
	}
	if !reflect.DeepEqual(args, []any{"1", "10"}) {
		t.Fatalf("args = %v", args)
	}
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
	b := sq.Select("*").From("users u").PlaceholderFormat(sq.Dollar)
	sql, args, err := spisqlsq.Apply(b, q, mapping).ToSql()
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT * FROM users u WHERE u.status = $1 AND (u.name = $2 OR u.name = $3)"
	if sql != want {
		t.Fatalf("\n got: %s\nwant: %s", sql, want)
	}
	if !reflect.DeepEqual(args, []any{"active", "vasya", "petya"}) {
		t.Fatalf("args = %v", args)
	}
}
