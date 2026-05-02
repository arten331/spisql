package echo

import (
	"github.com/labstack/echo/v4"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
)

func Parse(c echo.Context, opts parser.Options) *spisql.Query {
	return parser.Parse(c.Request().URL.Query(), opts)
}
