//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/arten331/spisql"
	spisqlmongo "github.com/arten331/spisql/adapter/mongo"
)

var mongoMapping = spisqlmongo.Mapping{
	"name":      "name",
	"email":     "email",
	"status":    "status",
	"createdAt": "created_at",
}

func TestMongo(t *testing.T) {
	coll := mongoColl(t)
	cases := []struct {
		name string
		q    *spisql.Query
		want int64
	}{
		{"eq", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
		}}, 1},
		{"in", &spisql.Query{Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpIn, Values: []string{"active", "banned"}},
		}}, 3},
		{"isnull", &spisql.Query{Filters: spisql.Filters{
			{Key: "email", Operator: spisql.OpIsNull},
		}}, 1},
		{"isnotnull", &spisql.Query{Filters: spisql.Filters{
			{Key: "email", Operator: spisql.OpIsNotNull},
		}}, 3},
		{"gtedate", &spisql.Query{Filters: spisql.Filters{
			{Key: "createdAt", Operator: spisql.OpGreaterThanEqualISODate, Values: []string{"2024-06-01"}},
		}}, 3}, // petya, masha, kostya
		{"substringof", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpSubstringOf, Values: []string{"as"}},
		}}, 2}, // vasya, masha
		{"startswith", &spisql.Query{Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpStartsWith, Values: []string{"v"}},
		}}, 1},
		{"or-group", &spisql.Query{
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
			res := spisqlmongo.Build(tc.q, mongoMapping)
			n, err := coll.CountDocuments(context.Background(), res.Filter)
			if err != nil {
				t.Fatalf("count failed: %v (filter=%+v)", err, res.Filter)
			}
			if n != tc.want {
				t.Errorf("want %d, got %d (filter=%+v)", tc.want, n, res.Filter)
			}
		})
	}
}
