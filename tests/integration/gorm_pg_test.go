//go:build integration

package integration

import (
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/arten331/spisql"
	spisqlgorm "github.com/arten331/spisql/adapter/gorm"
)

var gormMapping = spisqlgorm.Mapping{
	"name":       "name",
	"email":      "email",
	"status":     "status",
	"id":         "id",
	"created_at": "created_at",
}

func TestGorm_PG(t *testing.T) {
	db := pg(t)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	cases := []struct {
		name string
		q    *spisql.Query
		want int64
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
		{"or-group", &spisql.Query{
			Groups: spisql.Groups{
				{Op: spisql.GroupOr, Filters: spisql.Filters{
					{Key: "status", Operator: spisql.OpEqual, Values: []string{"deleted"}},
					{Key: "status", Operator: spisql.OpEqual, Values: []string{"banned"}},
				}},
			},
		}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := gdb.Table("users")
			tx = spisqlgorm.Apply(tx, tc.q, gormMapping)
			var n int64
			if err := tx.Count(&n).Error; err != nil {
				t.Fatalf("count failed: %v", err)
			}
			if n != tc.want {
				t.Errorf("want %d, got %d", tc.want, n)
			}
		})
	}
}
