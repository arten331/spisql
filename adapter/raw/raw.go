package raw

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arten331/spisql"
)

type Mapping map[spisql.Column]string

type Result struct {
	Where   string
	Args    []any
	OrderBy string
	Limit   uint64
	Offset  uint64
}

func Build(q *spisql.Query, m Mapping) Result {
	if q == nil {
		return Result{}
	}
	var (
		clauses []string
		args    []any
	)
	for _, f := range q.Filters {
		clause, fArgs, ok := buildFilter(f, m)
		if !ok {
			continue
		}
		clauses = append(clauses, "("+clause+")")
		args = append(args, fArgs...)
	}
	for _, g := range q.Groups {
		clause, gArgs, ok := buildGroup(g, m)
		if !ok {
			continue
		}
		clauses = append(clauses, clause)
		args = append(args, gArgs...)
	}

	var orders []string
	for _, s := range q.Sorts {
		col, ok := m[s.Key]
		if !ok {
			continue
		}
		orders = append(orders, col+" "+s.Direction.String())
	}

	return Result{
		Where:   strings.Join(clauses, " AND "),
		Args:    args,
		OrderBy: strings.Join(orders, ", "),
		Limit:   q.Pagination.Limit,
		Offset:  q.Pagination.Offset,
	}
}

func buildGroup(g *spisql.Group, m Mapping) (string, []any, bool) {
	if g == nil || len(g.Filters) == 0 {
		return "", nil, false
	}
	parts := make([]string, 0, len(g.Filters))
	args := make([]any, 0)
	for _, f := range g.Filters {
		frag, fArgs, ok := buildFilter(f, m)
		if !ok {
			continue
		}
		parts = append(parts, "("+frag+")")
		args = append(args, fArgs...)
	}
	if len(parts) == 0 {
		return "", nil, false
	}
	sep := " AND "
	if g.Op == spisql.GroupOr {
		sep = " OR "
	}
	return "(" + strings.Join(parts, sep) + ")", args, true
}

func buildFilter(f *spisql.Filter, m Mapping) (string, []any, bool) {
	col, ok := m[f.Key]
	if !ok {
		return "", nil, false
	}
	switch f.Operator {
	case spisql.OpEqual:
		if len(f.Values) == 1 {
			return col + " = ?", []any{f.Values[0]}, true
		}
		return inExpr(col, f.Values, false)
	case spisql.OpNotEqual:
		if len(f.Values) == 1 {
			return col + " <> ?", []any{f.Values[0]}, true
		}
		return inExpr(col, f.Values, true)
	case spisql.OpIn:
		return inExpr(col, f.Values, false)
	case spisql.OpNotIn:
		return inExpr(col, f.Values, true)
	case spisql.OpGreaterThan:
		return cmpExpr(col, ">", f.Values)
	case spisql.OpGreaterThanEqual, spisql.OpGreaterThanEqualISODate:
		return cmpExpr(col, ">=", f.Values)
	case spisql.OpLessThan:
		return cmpExpr(col, "<", f.Values)
	case spisql.OpLessThanEqual:
		return cmpExpr(col, "<=", f.Values)
	case spisql.OpSubstringOf:
		return likeExpr(col, f.Values, "%%%s%%")
	case spisql.OpStartsWith:
		return likeExpr(col, f.Values, "%s%%")
	case spisql.OpEndsWith:
		return likeExpr(col, f.Values, "%%%s")
	case spisql.OpIsNull:
		return col + " IS NULL", nil, true
	case spisql.OpIsNotNull:
		return col + " IS NOT NULL", nil, true
	case spisql.OpBetween:
		return betweenExpr(col, f.Values, false)
	case spisql.OpNotBetween:
		return betweenExpr(col, f.Values, true)
	}
	return "", nil, false
}

func betweenExpr(col string, values []string, negate bool) (string, []any, bool) {
	if len(values) != 2 {
		return "", nil, false
	}
	op := "BETWEEN"
	if negate {
		op = "NOT BETWEEN"
	}
	return col + " " + op + " ? AND ?", []any{values[0], values[1]}, true
}

func inExpr(col string, values []string, negate bool) (string, []any, bool) {
	if len(values) == 0 {
		return "", nil, false
	}
	op := "IN"
	if negate {
		op = "NOT IN"
	}
	args := make([]any, 0, len(values))
	placeholders := make([]string, 0, len(values))
	for _, v := range values {
		placeholders = append(placeholders, "?")
		args = append(args, v)
	}
	return col + " " + op + " (" + strings.Join(placeholders, ", ") + ")", args, true
}

func cmpExpr(col, op string, values []string) (string, []any, bool) {
	if len(values) == 0 {
		return "", nil, false
	}
	if len(values) == 1 {
		return col + " " + op + " ?", []any{values[0]}, true
	}
	parts := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	for _, v := range values {
		parts = append(parts, col+" "+op+" ?")
		args = append(args, v)
	}
	return strings.Join(parts, " OR "), args, true
}

func likeExpr(col string, values []string, pattern string) (string, []any, bool) {
	if len(values) == 0 {
		return "", nil, false
	}
	parts := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	for _, v := range values {
		parts = append(parts, col+" LIKE ?")
		args = append(args, fmt.Sprintf(pattern, v))
	}
	return strings.Join(parts, " OR "), args, true
}

func Dollar(sql string) string {
	var b strings.Builder
	b.Grow(len(sql) + 8)
	n := 0
	for _, r := range sql {
		if r == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
