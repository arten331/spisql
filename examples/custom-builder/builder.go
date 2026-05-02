// builder.go — a hand-written spisql adapter for plain database/sql.
//
// This is the minimum surface a custom adapter has to cover: walk a
// *spisql.Query, look each field up in your Mapping, and switch on
// f.Operator. Drop on unknown mapping or unknown operator — that's the
// contract every adapter (built-in or yours) follows.
//
// You'd normally put this in its own package next to your data layer.
package main

import (
	"fmt"
	"strings"

	"github.com/arten331/spisql"
)

// Mapping rephrases public-API field names as DB column expressions.
type Mapping map[spisql.Column]string

// Result is what a "Build"-style adapter hands back: enough fragments to
// concatenate into a real query.
type Result struct {
	Where   string
	Args    []any
	OrderBy string
	Limit   uint64
	Offset  uint64
}

// Build turns a *spisql.Query into a Result for database/sql consumers.
// Filters whose Key isn't in the mapping, or whose Operator isn't
// handled, are silently dropped — same as the bundled adapters.
func Build(q *spisql.Query, m Mapping) Result {
	if q == nil {
		return Result{}
	}
	var (
		clauses []string
		args    []any
	)
	for _, f := range q.Filters {
		col, ok := m[f.Key]
		if !ok {
			continue
		}
		clause, fArgs, ok := buildFilter(col, f)
		if !ok {
			continue
		}
		clauses = append(clauses, clause)
		args = append(args, fArgs...)
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

func buildFilter(col string, f *spisql.Filter) (string, []any, bool) {
	switch f.Operator {
	case spisql.OpEqual:
		if len(f.Values) != 1 {
			return "", nil, false
		}
		return col + " = ?", []any{f.Values[0]}, true

	case spisql.OpNotEqual:
		if len(f.Values) != 1 {
			return "", nil, false
		}
		return col + " <> ?", []any{f.Values[0]}, true

	case spisql.OpIn, spisql.OpNotIn:
		if len(f.Values) == 0 {
			return "", nil, false
		}
		op := "IN"
		if f.Operator == spisql.OpNotIn {
			op = "NOT IN"
		}
		ph := strings.Repeat("?, ", len(f.Values))
		ph = ph[:len(ph)-2]
		args := make([]any, len(f.Values))
		for i, v := range f.Values {
			args[i] = v
		}
		return fmt.Sprintf("%s %s (%s)", col, op, ph), args, true

	case spisql.OpGreaterThan, spisql.OpGreaterThanEqual,
		spisql.OpGreaterThanEqualISODate,
		spisql.OpLessThan, spisql.OpLessThanEqual:
		if len(f.Values) != 1 {
			return "", nil, false
		}
		sym := map[spisql.Operator]string{
			spisql.OpGreaterThan:             ">",
			spisql.OpGreaterThanEqual:        ">=",
			spisql.OpGreaterThanEqualISODate: ">=",
			spisql.OpLessThan:                "<",
			spisql.OpLessThanEqual:           "<=",
		}[f.Operator]
		return fmt.Sprintf("%s %s ?", col, sym), []any{f.Values[0]}, true

	case spisql.OpIsNull:
		return col + " IS NULL", nil, true
	case spisql.OpIsNotNull:
		return col + " IS NOT NULL", nil, true

	case spisql.OpBetween:
		if len(f.Values) != 2 {
			return "", nil, false
		}
		return col + " BETWEEN ? AND ?", []any{f.Values[0], f.Values[1]}, true
	case spisql.OpNotBetween:
		if len(f.Values) != 2 {
			return "", nil, false
		}
		return col + " NOT BETWEEN ? AND ?", []any{f.Values[0], f.Values[1]}, true

	case spisql.OpSubstringOf, spisql.OpStartsWith, spisql.OpEndsWith:
		if len(f.Values) != 1 {
			return "", nil, false
		}
		v := f.Values[0]
		switch f.Operator {
		case spisql.OpSubstringOf:
			v = "%" + v + "%"
		case spisql.OpStartsWith:
			v = v + "%"
		case spisql.OpEndsWith:
			v = "%" + v
		}
		return col + " LIKE ?", []any{v}, true
	}
	return "", nil, false
}

// dollar converts ? placeholders to $1, $2, ... for PostgreSQL.
func dollar(s string) string {
	var b strings.Builder
	n := 0
	for _, r := range s {
		if r == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
