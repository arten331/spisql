package mongo_test

import (
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/arten331/spisql"
	spisqlmongo "github.com/arten331/spisql/adapter/mongo"
)

var mapping = spisqlmongo.Mapping{
	"name":      "name",
	"id":        "_id",
	"createdAt": "created_at",
}

func TestBuild_Equal(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"vasya"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	want := bson.D{{Key: "name", Value: bson.D{{Key: "$eq", Value: "vasya"}}}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}

func TestBuild_In(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpIn, Values: []string{"1", "2", "3"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	want := bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: []any{"1", "2", "3"}}}}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}

func TestBuild_GteDate(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "createdAt", Operator: spisql.OpGreaterThanEqualISODate, Values: []string{"2026-01-15"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	expectedTime, _ := time.Parse(time.DateOnly, "2026-01-15")
	want := bson.D{{Key: "created_at", Value: bson.D{{Key: "$gte", Value: expectedTime}}}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}

func TestBuild_GteDate_Invalid(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "createdAt", Operator: spisql.OpGreaterThanEqualISODate, Values: []string{"not-a-date"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	if len(got.Filter) != 0 {
		t.Fatalf("expected invalid date to be dropped, got %+v", got.Filter)
	}
}

func TestBuild_SortAndPagination(t *testing.T) {
	q := &spisql.Query{
		Sorts: spisql.Sorts{
			{Key: "name", Direction: spisql.DirectionAsc},
			{Key: "createdAt", Direction: spisql.DirectionDesc},
		},
		Pagination: spisql.Pagination{Limit: 25, Offset: 50},
	}
	got := spisqlmongo.Build(q, mapping)
	wantSort := bson.D{{Key: "name", Value: 1}, {Key: "created_at", Value: -1}}
	if !reflect.DeepEqual(got.Sort, wantSort) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Sort, wantSort)
	}
	if got.Limit != 25 || got.Offset != 50 {
		t.Fatalf("limit/offset = %d/%d", got.Limit, got.Offset)
	}
}

func TestBuild_FindOptions(t *testing.T) {
	q := &spisql.Query{
		Sorts:      spisql.Sorts{{Key: "name", Direction: spisql.DirectionAsc}},
		Pagination: spisql.Pagination{Limit: 10, Offset: 5},
	}
	opts := spisqlmongo.Build(q, mapping).FindOptions()
	if opts == nil {
		t.Fatal("expected non-nil FindOptions")
	}
	if opts.Limit == nil || *opts.Limit != 10 {
		t.Fatalf("limit = %v", opts.Limit)
	}
	if opts.Skip == nil || *opts.Skip != 5 {
		t.Fatalf("skip = %v", opts.Skip)
	}
}

func TestBuild_Pipeline(t *testing.T) {
	q := &spisql.Query{
		Sorts:      spisql.Sorts{{Key: "name", Direction: spisql.DirectionAsc}},
		Pagination: spisql.Pagination{Limit: 10, Offset: 5},
	}
	pipe := spisqlmongo.Build(q, mapping).Pipeline()
	if len(pipe) != 3 {
		t.Fatalf("expected 3 stages (sort/skip/limit), got %d: %+v", len(pipe), pipe)
	}
}

func TestBuild_UnmappedDropped(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "secret", Operator: spisql.OpEqual, Values: []string{"x"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	if len(got.Filter) != 0 {
		t.Fatalf("unmapped filter should drop, got %+v", got.Filter)
	}
}

func TestBuild_IsNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpIsNull},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	want := bson.D{{Key: "name", Value: bson.D{{Key: "$eq", Value: nil}}}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}

func TestBuild_IsNotNull(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpIsNotNull},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	want := bson.D{{Key: "name", Value: bson.D{{Key: "$ne", Value: nil}}}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}

func TestBuild_Between(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"1", "10"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	want := bson.D{{Key: "_id", Value: bson.D{
		{Key: "$gte", Value: "1"},
		{Key: "$lte", Value: "10"},
	}}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}

func TestBuild_NotBetween(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpNotBetween, Values: []string{"1", "10"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	want := bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "_id", Value: bson.D{{Key: "$lt", Value: "1"}}}},
		bson.D{{Key: "_id", Value: bson.D{{Key: "$gt", Value: "10"}}}},
	}}}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}

func TestBuild_BetweenWrongArity_Dropped(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "id", Operator: spisql.OpBetween, Values: []string{"only-one"}},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	if len(got.Filter) != 0 {
		t.Fatalf("between with 1 value must be dropped, got %+v", got.Filter)
	}
}

func TestBuild_OrGroup(t *testing.T) {
	q := &spisql.Query{
		Filters: spisql.Filters{
			{Key: "name", Operator: spisql.OpEqual, Values: []string{"x"}},
		},
		Groups: spisql.Groups{
			{
				Op: spisql.GroupOr,
				Filters: spisql.Filters{
					{Key: "id", Operator: spisql.OpEqual, Values: []string{"1"}},
					{Key: "id", Operator: spisql.OpEqual, Values: []string{"2"}},
				},
			},
		},
	}
	got := spisqlmongo.Build(q, mapping)
	want := bson.D{
		{Key: "name", Value: bson.D{{Key: "$eq", Value: "x"}}},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "_id", Value: bson.D{{Key: "$eq", Value: "1"}}}},
			bson.D{{Key: "_id", Value: bson.D{{Key: "$eq", Value: "2"}}}},
		}},
	}
	if !reflect.DeepEqual(got.Filter, want) {
		t.Fatalf("\n got: %+v\nwant: %+v", got.Filter, want)
	}
}
