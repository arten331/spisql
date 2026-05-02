package gorm

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/arten331/spisql"
)

type Mapping map[spisql.Column]string

func Apply(db *gorm.DB, q *spisql.Query, m Mapping) *gorm.DB {
	if q == nil {
		return db
	}
	for _, f := range q.Filters {
		if frag, args, ok := Filter(f, m); ok {
			db = db.Where(frag, args...)
		}
	}
	for _, gr := range q.Groups {
		if frag, args, ok := Group(gr, m); ok {
			db = db.Where(frag, args...)
		}
	}
	for _, s := range q.Sorts {
		if frag := Sort(s, m); frag != "" {
			db = db.Order(frag)
		}
	}
	if q.Pagination.Limit > 0 {
		db = db.Limit(int(q.Pagination.Limit))
	}
	if q.Pagination.Offset > 0 {
		db = db.Offset(int(q.Pagination.Offset))
	}
	return db
}

func Group(gr *spisql.Group, m Mapping) (string, []any, bool) {
	if gr == nil || len(gr.Filters) == 0 {
		return "", nil, false
	}
	parts := make([]string, 0, len(gr.Filters))
	args := make([]any, 0)
	for _, f := range gr.Filters {
		frag, fArgs, ok := Filter(f, m)
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
	if gr.Op == spisql.GroupOr {
		sep = " OR "
	}
	return "(" + strings.Join(parts, sep) + ")", args, true
}

func Filter(f *spisql.Filter, m Mapping) (string, []any, bool) {
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
		return ilikeExpr(col, f.Values, "%%%s%%")
	case spisql.OpStartsWith:
		return ilikeExpr(col, f.Values, "%s%%")
	case spisql.OpEndsWith:
		return ilikeExpr(col, f.Values, "%%%s")
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

func Sort(s *spisql.Sort, m Mapping) string {
	col, ok := m[s.Key]
	if !ok {
		return ""
	}
	return col + " " + s.Direction.String()
}

func inExpr(col string, values []string, negate bool) (string, []any, bool) {
	if len(values) == 0 {
		return "", nil, false
	}
	op := "IN"
	if negate {
		op = "NOT IN"
	}
	return col + " " + op + " ?", []any{values}, true
}

func cmpExpr(col, op string, values []string) (string, []any, bool) {
	if len(values) == 0 {
		return "", nil, false
	}
	if len(values) == 1 {
		return col + " " + op + " ?", []any{values[0]}, true
	}
	frag := ""
	args := make([]any, 0, len(values))
	for i, v := range values {
		if i > 0 {
			frag += " OR "
		}
		frag += col + " " + op + " ?"
		args = append(args, v)
	}
	return "(" + frag + ")", args, true
}

func ilikeExpr(col string, values []string, pattern string) (string, []any, bool) {
	if len(values) == 0 {
		return "", nil, false
	}
	frag := ""
	args := make([]any, 0, len(values))
	for i, v := range values {
		if i > 0 {
			frag += " OR "
		}
		frag += col + " ILIKE ?"
		args = append(args, fmt.Sprintf(pattern, v))
	}
	return "(" + frag + ")", args, true
}
