//go:build integration

package integration

import (
	"context"
	"testing"

	g "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"

	"github.com/arten331/spisql"
	spisqlgoqu "github.com/arten331/spisql/adapter/goqu"
)

var goquMapping = spisqlgoqu.Mapping{
	"name":       "name",
	"email":      "email",
	"status":     "status",
	"id":         "id",
	"created_at": "created_at",
}

func TestGoqu_PG(t *testing.T) {
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
		{"between", &spisql.Query{Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"2", "3"}},
		}}, 2},
		{"endswith", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEndsWith, Values: []string{"ya"}},
		}}, 3}, // vasya, petya, kostya
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
			ds := g.Dialect("postgres").From("users").Select(g.COUNT("*"))
			s, args, err := spisqlgoqu.Apply(ds, tc.q, goquMapping).Prepared(true).ToSQL()
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
