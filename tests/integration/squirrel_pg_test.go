//go:build integration

package integration

import (
	"context"
	"testing"

	sq "github.com/Masterminds/squirrel"

	"github.com/arten331/spisql"
	spisqlsq "github.com/arten331/spisql/adapter/squirrel"
)

var sqMapping = spisqlsq.Mapping{
	"name":       "name",
	"email":      "email",
	"status":     "status",
	"id":         "id",
	"created_at": "created_at",
}

func TestSquirrel_PG(t *testing.T) {
	db := pg(t)
	cases := []struct {
		name string
		q    *spisql.Query
		want int
	}{
		{"eq", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
		}}, 1},
		{"in", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpIn, Values: []string{"vasya", "petya"}},
		}}, 2},
		{"isnull", &spisql.Query{Filters: spisql.Filters{
			{Key: "email", Operator: spisql.OpIsNull},
		}}, 1},
		{"isnotnull", &spisql.Query{Filters: spisql.Filters{
			{Key: "email", Operator: spisql.OpIsNotNull},
		}}, 3},
		{"between", &spisql.Query{Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"2", "3"}},
		}}, 2},
		{"substringof", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpSubstringOf, Values: []string{"as"}},
		}}, 2}, // vasya, masha
		{"or-group", &spisql.Query{
			Filters: spisql.Filters{
				{Key: "status", Operator: spisql.OpEqual, Values: []string{"active"}},
			},
			Groups: spisql.Groups{
				{Op: spisql.GroupOr, Filters: spisql.Filters{
					{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
					{Key: "name", Operator: spisql.OpEqual, Values: []string{"petya"}},
				}},
			},
		}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := sq.Select("count(*)").From("users").PlaceholderFormat(sq.Dollar)
			s, args, err := spisqlsq.Apply(b, tc.q, sqMapping).ToSql()
			if err != nil {
				t.Fatal(err)
			}
			var n int
			if err := db.QueryRowContext(context.Background(), s, args...).Scan(&n); err != nil {
				t.Fatalf("query=%q args=%v: %v", s, args, err)
			}
			if n != tc.want {
				t.Errorf("want %d, got %d (sql=%s args=%v)", tc.want, n, s, args)
			}
		})
	}
}
