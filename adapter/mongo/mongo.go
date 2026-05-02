package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/arten331/spisql"
)

type Mapping map[spisql.Column]string

type Result struct {
	Filter bson.D
	Sort   bson.D
	Limit  int64
	Offset int64
}

func (r Result) FindOptions() *options.FindOptions {
	o := options.Find()
	if len(r.Sort) > 0 {
		o.SetSort(r.Sort)
	}
	if r.Limit > 0 {
		o.SetLimit(r.Limit)
	}
	if r.Offset > 0 {
		o.SetSkip(r.Offset)
	}
	return o
}

func (r Result) Pipeline() bson.A {
	pipe := bson.A{}
	if len(r.Sort) > 0 {
		pipe = append(pipe, bson.D{{Key: "$sort", Value: r.Sort}})
	}
	if r.Offset > 0 {
		pipe = append(pipe, bson.D{{Key: "$skip", Value: r.Offset}})
	}
	if r.Limit > 0 {
		pipe = append(pipe, bson.D{{Key: "$limit", Value: r.Limit}})
	}
	return pipe
}

func Build(q *spisql.Query, m Mapping) Result {
	if q == nil {
		return Result{Filter: bson.D{}, Sort: bson.D{}}
	}
	res := Result{
		Filter: Filters(q.Filters, m),
		Sort:   Sorts(q.Sorts, m),
		Limit:  int64(q.Pagination.Limit),
		Offset: int64(q.Pagination.Offset),
	}
	for _, gr := range q.Groups {
		if e := Group(gr, m); e.Key != "" {
			res.Filter = append(res.Filter, e)
		}
	}
	if res.Filter == nil {
		res.Filter = bson.D{}
	}
	if res.Sort == nil {
		res.Sort = bson.D{}
	}
	return res
}

func Group(gr *spisql.Group, m Mapping) bson.E {
	if gr == nil || len(gr.Filters) == 0 {
		return bson.E{}
	}
	parts := bson.A{}
	for _, f := range gr.Filters {
		e := Filter(f, m)
		if e.Key == "" {
			continue
		}
		parts = append(parts, bson.D{e})
	}
	if len(parts) == 0 {
		return bson.E{}
	}
	key := "$or"
	if gr.Op == spisql.GroupAnd {
		key = "$and"
	}
	return bson.E{Key: key, Value: parts}
}

func Filters(fs spisql.Filters, m Mapping) bson.D {
	out := bson.D{}
	for _, f := range fs {
		if e := Filter(f, m); e.Key != "" {
			out = append(out, e)
		}
	}
	return out
}

func Filter(f *spisql.Filter, m Mapping) bson.E {
	col, ok := m[f.Key]
	if !ok {
		return bson.E{}
	}
	switch f.Operator {
	case spisql.OpEqual:
		if len(f.Values) == 0 {
			return bson.E{}
		}
		v := f.Values[0]
		if v == "" {
			return bson.E{Key: col, Value: bson.D{{Key: "$eq", Value: nil}}}
		}
		return bson.E{Key: col, Value: bson.D{{Key: "$eq", Value: v}}}
	case spisql.OpIn:
		return bson.E{Key: col, Value: bson.D{{Key: "$in", Value: anySlice(f.Values)}}}
	case spisql.OpNotIn:
		return bson.E{Key: col, Value: bson.D{{Key: "$nin", Value: anySlice(f.Values)}}}
	case spisql.OpNotEqual:
		return bson.E{Key: col, Value: bson.D{{Key: "$ne", Value: firstOrNil(f.Values)}}}
	case spisql.OpGreaterThan:
		return bson.E{Key: col, Value: bson.D{{Key: "$gt", Value: firstOrNil(f.Values)}}}
	case spisql.OpGreaterThanEqual:
		return bson.E{Key: col, Value: bson.D{{Key: "$gte", Value: firstOrNil(f.Values)}}}
	case spisql.OpGreaterThanEqualISODate:
		if len(f.Values) == 0 {
			return bson.E{}
		}
		t, err := time.Parse(time.DateOnly, f.Values[0])
		if err != nil {
			return bson.E{}
		}
		return bson.E{Key: col, Value: bson.D{{Key: "$gte", Value: t}}}
	case spisql.OpLessThan:
		return bson.E{Key: col, Value: bson.D{{Key: "$lt", Value: firstOrNil(f.Values)}}}
	case spisql.OpLessThanEqual:
		return bson.E{Key: col, Value: bson.D{{Key: "$lte", Value: firstOrNil(f.Values)}}}
	case spisql.OpSubstringOf:
		return regex(col, firstString(f.Values))
	case spisql.OpStartsWith:
		return regex(col, fmt.Sprintf("^%s", firstString(f.Values)))
	case spisql.OpEndsWith:
		return regex(col, fmt.Sprintf("%s$", firstString(f.Values)))
	case spisql.OpIsNull:
		return bson.E{Key: col, Value: bson.D{{Key: "$eq", Value: nil}}}
	case spisql.OpIsNotNull:
		return bson.E{Key: col, Value: bson.D{{Key: "$ne", Value: nil}}}
	case spisql.OpBetween:
		if len(f.Values) != 2 {
			return bson.E{}
		}
		return bson.E{Key: col, Value: bson.D{
			{Key: "$gte", Value: f.Values[0]},
			{Key: "$lte", Value: f.Values[1]},
		}}
	case spisql.OpNotBetween:
		if len(f.Values) != 2 {
			return bson.E{}
		}
		return bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: col, Value: bson.D{{Key: "$lt", Value: f.Values[0]}}}},
			bson.D{{Key: col, Value: bson.D{{Key: "$gt", Value: f.Values[1]}}}},
		}}
	}
	return bson.E{}
}

func Sorts(ss spisql.Sorts, m Mapping) bson.D {
	out := bson.D{}
	for _, s := range ss {
		if e := Sort(s, m); e.Key != "" {
			out = append(out, e)
		}
	}
	return out
}

func Sort(s *spisql.Sort, m Mapping) bson.E {
	col, ok := m[s.Key]
	if !ok {
		return bson.E{}
	}
	dir := 1
	if s.Direction == spisql.DirectionDesc {
		dir = -1
	}
	return bson.E{Key: col, Value: dir}
}

func regex(col, pattern string) bson.E {
	return bson.E{Key: col, Value: bson.M{"$regex": primitive.Regex{Pattern: pattern, Options: "i"}}}
}

func anySlice(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func firstOrNil(values []string) any {
	if len(values) == 0 {
		return nil
	}
	return values[0]
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
