package fiber

import (
	"net/url"

	"github.com/gofiber/fiber/v2"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
)

func Parse(c *fiber.Ctx, opts parser.Options) *spisql.Query {
	v := url.Values{}
	c.Context().QueryArgs().VisitAll(func(key, value []byte) {
		v.Add(string(key), string(value))
	})
	return parser.Parse(v, opts)
}
