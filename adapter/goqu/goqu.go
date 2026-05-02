package goqu

import (
	"fmt"

	g "github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"

	"github.com/arten331/spisql"
)

type Mapping map[spisql.Column]string

func Apply(ds *g.SelectDataset, q *spisql.Query, m Mapping) *g.SelectDataset {
	if q == nil {
		return ds
	}
	for _, e := range Filters(q.Filters, m) {
		ds = ds.Where(e)
	}
	for _, gr := range q.Groups {
		if e := Group(gr, m); e != nil {
			ds = ds.Where(e)
		}
	}
	if orders := Sorts(q.Sorts, m); len(orders) > 0 {
		ds = ds.Order(orders...)
	}
	if q.Pagination.Limit > 0 {
		ds = ds.Limit(uint(q.Pagination.Limit))
	}
	if q.Pagination.Offset > 0 {
		ds = ds.Offset(uint(q.Pagination.Offset))
	}
	return ds
}

func Group(gr *spisql.Group, m Mapping) exp.Expression {
	if gr == nil || len(gr.Filters) == 0 {
		return nil
	}
	parts := Filters(gr.Filters, m)
	if len(parts) == 0 {
		return nil
	}
	if gr.Op == spisql.GroupOr {
		return g.Or(parts...)
	}
	return g.And(parts...)
}

func Filters(fs spisql.Filters, m Mapping) []exp.Expression {
	out := make([]exp.Expression, 0, len(fs))
	for _, f := range fs {
		if e := Filter(f, m); e != nil {
			out = append(out, e)
		}
	}
	return out
}

func Filter(f *spisql.Filter, m Mapping) exp.Expression {
	col, ok := m[f.Key]
	if !ok {
		return nil
	}
	c := g.I(col)
	switch f.Operator {
	case spisql.OpEqual:
		if len(f.Values) == 1 {
			return c.Eq(f.Values[0])
		}
		return c.In(toAny(f.Values)...)
	case spisql.OpNotEqual:
		if len(f.Values) == 1 {
			return c.Neq(f.Values[0])
		}
		return c.NotIn(toAny(f.Values)...)
	case spisql.OpIn:
		return c.In(toAny(f.Values)...)
	case spisql.OpNotIn:
		return c.NotIn(toAny(f.Values)...)
	case spisql.OpGreaterThan:
		return orChain(f.Values, func(v string) exp.Expression { return c.Gt(v) })
	case spisql.OpGreaterThanEqual, spisql.OpGreaterThanEqualISODate:
		return orChain(f.Values, func(v string) exp.Expression { return c.Gte(v) })
	case spisql.OpLessThan:
		return orChain(f.Values, func(v string) exp.Expression { return c.Lt(v) })
	case spisql.OpLessThanEqual:
		return orChain(f.Values, func(v string) exp.Expression { return c.Lte(v) })
	case spisql.OpSubstringOf:
		return orChain(f.Values, func(v string) exp.Expression { return c.ILike(fmt.Sprintf("%%%s%%", v)) })
	case spisql.OpStartsWith:
		return orChain(f.Values, func(v string) exp.Expression { return c.ILike(fmt.Sprintf("%s%%", v)) })
	case spisql.OpEndsWith:
		return orChain(f.Values, func(v string) exp.Expression { return c.ILike(fmt.Sprintf("%%%s", v)) })
	case spisql.OpIsNull:
		return c.IsNull()
	case spisql.OpIsNotNull:
		return c.IsNotNull()
	case spisql.OpBetween:
		if len(f.Values) != 2 {
			return nil
		}
		return c.Between(g.Range(f.Values[0], f.Values[1]))
	case spisql.OpNotBetween:
		if len(f.Values) != 2 {
			return nil
		}
		return c.NotBetween(g.Range(f.Values[0], f.Values[1]))
	}
	return nil
}

func Sorts(ss spisql.Sorts, m Mapping) []exp.OrderedExpression {
	out := make([]exp.OrderedExpression, 0, len(ss))
	for _, s := range ss {
		if oe := Sort(s, m); oe != nil {
			out = append(out, oe)
		}
	}
	return out
}

func Sort(s *spisql.Sort, m Mapping) exp.OrderedExpression {
	col, ok := m[s.Key]
	if !ok {
		return nil
	}
	if s.Direction == spisql.DirectionDesc {
		return g.I(col).Desc()
	}
	return g.I(col).Asc()
}

func orChain(values []string, build func(string) exp.Expression) exp.Expression {
	if len(values) == 0 {
		return nil
	}
	if len(values) == 1 {
		return build(values[0])
	}
	out := make([]exp.Expression, 0, len(values))
	for _, v := range values {
		out = append(out, build(v))
	}
	return g.Or(out...)
}

func toAny(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}
