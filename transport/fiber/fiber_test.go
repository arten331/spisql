package fiber_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
	spisqlfiber "github.com/arten331/spisql/transport/fiber"
)

func TestParse_FullQuery(t *testing.T) {
	app := fiber.New()
	var got *spisql.Query
	app.Get("/users", func(c *fiber.Ctx) error {
		got = spisqlfiber.Parse(c, parser.Options{})
		return nil
	})

	req := httptest.NewRequest(
		"GET",
		"/users?name=vasya&id[in]=1&id[in]=2&sort=-created_at&limit=20&offset=40",
		nil,
	)
	if _, err := app.Test(req, -1); err != nil {
		t.Fatal(err)
	}
	if len(got.Filters) != 2 {
		t.Fatalf("want 2 filters, got %d", len(got.Filters))
	}
	if len(got.Sorts) != 1 || got.Sorts[0].Direction != spisql.DirectionDesc {
		t.Fatalf("sorts=%+v", got.Sorts)
	}
	if got.Pagination.Limit != 20 || got.Pagination.Offset != 40 {
		t.Fatalf("pagination=%+v", got.Pagination)
	}
}
