package grpc

import (
	"net/url"
	"strconv"

	"github.com/arten331/spisql"
	"github.com/arten331/spisql/parser"
)

func Parse(filter, sort string, limit, offset uint64, opts parser.Options) *spisql.Query {
	values, _ := url.ParseQuery(filter)
	if sort != "" {
		values.Set(parser.KeySort, sort)
	}
	if limit > 0 {
		values.Set(parser.KeyLimit, strconv.FormatUint(limit, 10))
	}
	if offset > 0 {
		values.Set(parser.KeyOffset, strconv.FormatUint(offset, 10))
	}
	return parser.Parse(values, opts)
}
