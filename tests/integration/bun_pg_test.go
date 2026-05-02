//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/arten331/spisql"
	spisqlbun "github.com/arten331/spisql/adapter/bun"
)

var bunMapping = spisqlbun.Mapping{
	"name":       "name",
	"email":      "email",
	"status":     "status",
	"id":         "id",
	"created_at": "created_at",
}

func TestBun_PG(t *testing.T) {
	db := pg(t)
	bunDB := bun.NewDB(db, pgdialect.New())
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
		{"nin", &spisql.Query{Filters: spisql.Filters{
			{Key: "status", Operator: spisql.OpNotIn, Values: []string{"deleted", "banned"}},
		}}, 2},
		{"isnull", &spisql.Query{Filters: spisql.Filters{
			{Key: "email", Operator: spisql.OpIsNull},
		}}, 1},
		{"between", &spisql.Query{Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"2", "3"}},
		}}, 2},
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
			sq := bunDB.NewSelect().Table("users").ColumnExpr("count(*)")
			sq = spisqlbun.Apply(sq, tc.q, bunMapping)
			var n int
			if err := sq.Scan(context.Background(), &n); err != nil {
				rawSQL, _ := sq.AppendQuery(bunDB.Formatter(), nil)
				t.Fatalf("scan failed: %v (sql=%s)", err, rawSQL)
			}
			if n != tc.want {
				t.Errorf("want %d, got %d", tc.want, n)
			}
		})
	}
}
