package squirrel

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"github.com/arten331/spisql"
)

type Mapping map[spisql.Column]string

func Apply(b sq.SelectBuilder, q *spisql.Query, m Mapping) sq.SelectBuilder {
	if q == nil {
		return b
	}
	for _, expr := range Filters(q.Filters, m) {
		b = b.Where(expr)
	}
	for _, g := range q.Groups {
		if expr := Group(g, m); expr != nil {
			b = b.Where(expr)
		}
	}
	if orders := Sorts(q.Sorts, m); len(orders) > 0 {
		b = b.OrderBy(orders...)
	}
	if q.Pagination.Limit > 0 {
		b = b.Limit(q.Pagination.Limit)
	}
	if q.Pagination.Offset > 0 {
		b = b.Offset(q.Pagination.Offset)
	}
	return b
}

func Group(g *spisql.Group, m Mapping) sq.Sqlizer {
	if g == nil || len(g.Filters) == 0 {
		return nil
	}
	parts := Filters(g.Filters, m)
	if len(parts) == 0 {
		return nil
	}
	if g.Op == spisql.GroupOr {
		out := make(sq.Or, 0, len(parts))
		for _, p := range parts {
			out = append(out, p)
		}
		return out
	}
	out := make(sq.And, 0, len(parts))
	for _, p := range parts {
		out = append(out, p)
	}
	return out
}

func Filters(fs spisql.Filters, m Mapping) []sq.Sqlizer {
	out := make([]sq.Sqlizer, 0, len(fs))
	for _, f := range fs {
		if expr := Filter(f, m); expr != nil {
			out = append(out, expr)
		}
	}
	return out
}

func Filter(f *spisql.Filter, m Mapping) sq.Sqlizer {
	col, ok := m[f.Key]
	if !ok {
		return nil
	}
	switch f.Operator {
	case spisql.OpEqual:
		if len(f.Values) == 1 {
			return sq.Eq{col: f.Values[0]}
		}
		return sq.Eq{col: f.Values}
	case spisql.OpNotEqual:
		if len(f.Values) == 1 {
			return sq.NotEq{col: f.Values[0]}
		}
		return sq.NotEq{col: f.Values}
	case spisql.OpIn:
		return sq.Eq{col: f.Values}
	case spisql.OpNotIn:
		return sq.NotEq{col: f.Values}
	case spisql.OpGreaterThan:
		return orChain(f.Values, func(v string) sq.Sqlizer { return sq.Gt{col: v} })
	case spisql.OpGreaterThanEqual, spisql.OpGreaterThanEqualISODate:
		return orChain(f.Values, func(v string) sq.Sqlizer { return sq.GtOrEq{col: v} })
	case spisql.OpLessThan:
		return orChain(f.Values, func(v string) sq.Sqlizer { return sq.Lt{col: v} })
	case spisql.OpLessThanEqual:
		return orChain(f.Values, func(v string) sq.Sqlizer { return sq.LtOrEq{col: v} })
	case spisql.OpSubstringOf:
		return orChain(f.Values, func(v string) sq.Sqlizer { return sq.ILike{col: fmt.Sprintf("%%%s%%", v)} })
	case spisql.OpStartsWith:
		return orChain(f.Values, func(v string) sq.Sqlizer { return sq.ILike{col: fmt.Sprintf("%s%%", v)} })
	case spisql.OpEndsWith:
		return orChain(f.Values, func(v string) sq.Sqlizer { return sq.ILike{col: fmt.Sprintf("%%%s", v)} })
	case spisql.OpIsNull:
		return sq.Eq{col: nil}
	case spisql.OpIsNotNull:
		return sq.NotEq{col: nil}
	case spisql.OpBetween:
		if len(f.Values) != 2 {
			return nil
		}
		return sq.Expr(col+" BETWEEN ? AND ?", f.Values[0], f.Values[1])
	case spisql.OpNotBetween:
		if len(f.Values) != 2 {
			return nil
		}
		return sq.Expr(col+" NOT BETWEEN ? AND ?", f.Values[0], f.Values[1])
	}
	return nil
}

func Sorts(ss spisql.Sorts, m Mapping) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if str := Sort(s, m); str != "" {
			out = append(out, str)
		}
	}
	return out
}

func Sort(s *spisql.Sort, m Mapping) string {
	col, ok := m[s.Key]
	if !ok {
		return ""
	}
	return col + " " + s.Direction.String()
}

func orChain(values []string, build func(v string) sq.Sqlizer) sq.Sqlizer {
	if len(values) == 0 {
		return nil
	}
	if len(values) == 1 {
		return build(values[0])
	}
	out := make(sq.Or, 0, len(values))
	for _, v := range values {
		out = append(out, build(v))
	}
	return out
}
