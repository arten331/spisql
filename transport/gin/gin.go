package gin

import (
	"github.com/gin-gonic/gin"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
)

func Parse(c *gin.Context, opts parser.Options) *spisql.Query {
	return parser.Parse(c.Request.URL.Query(), opts)
}
