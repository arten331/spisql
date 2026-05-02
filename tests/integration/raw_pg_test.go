//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/adapter/raw"
)

var rawMapping = raw.Mapping{
	"name":       "name",
	"email":      "email",
	"status":     "status",
	"id":         "id",
	"created_at": "created_at",
}

func TestRaw_PG(t *testing.T) {
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
		{"startswith", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpStartsWith, Values: []string{"v"}},
		}}, 1},
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
			res := raw.Build(tc.q, rawMapping)
			sqlStr := "SELECT count(*) FROM users"
			if res.Where != "" {
				sqlStr += " WHERE " + res.Where
			}
			sqlStr = raw.Dollar(sqlStr)
			var n int
			if err := db.QueryRowContext(context.Background(), sqlStr, res.Args...).Scan(&n); err != nil {
				t.Fatalf("query=%q args=%v: %v", sqlStr, res.Args, err)
			}
			if n != tc.want {
				t.Errorf("want %d, got %d (sql=%s args=%v)", tc.want, n, sqlStr, res.Args)
			}
		})
	}
}
