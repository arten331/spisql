//go:build integration

package grpcgateway

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/arten331/spisql"
	spisqlsq "github.com/arten331/spisql/adapter/squirrel"
	"github.com/arten331/spisql/parser"
	spisqlgrpc "github.com/arten331/spisql/transport/grpc"
	usersv1 "github.com/arten331/spisql/tests/grpcgateway/proto/usersv1"
)

type usersServer struct {
	usersv1.UnimplementedUsersServer
	db *sql.DB
}

var (
	allowedFilter = parser.FilterOptions{
		"name": {}, "email": {}, "status": {}, "id": {}, "created_at": {},
	}
	allowedSort = parser.SortOptions{
		"name": {}, "id": {}, "created_at": {},
	}
	sqMapping = spisqlsq.Mapping{
		"name":       "name",
		"email":      "email",
		"status":     "status",
		"id":         "id",
		"created_at": "created_at",
	}
)

func (s *usersServer) List(ctx context.Context, req *usersv1.ListUsersRequest) (*usersv1.ListUsersResponse, error) {
	q := req.GetQuery()
	sq3 := spisqlgrpc.Parse(q.GetFilter(), q.GetSort(), q.GetLimit(), q.GetOffset(), parser.Options{
		Filter:       allowedFilter,
		Sort:         allowedSort,
		DefaultLimit: 10,
		MaxLimit:     100,
	})

	items, err := s.fetch(ctx, sq3)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetch: %v", err)
	}
	total, err := s.count(ctx, sq3)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "count: %v", err)
	}
	return &usersv1.ListUsersResponse{Items: items, Total: total}, nil
}

func (s *usersServer) fetch(ctx context.Context, q *spisql.Query) ([]*usersv1.User, error) {
	b := sq.Select("id", "name", "COALESCE(email, '')", "status").
		From("users").
		PlaceholderFormat(sq.Dollar)
	stmt, args, err := spisqlsq.Apply(b, q, sqMapping).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, fmt.Errorf("query %q args=%v: %w", stmt, args, err)
	}
	defer rows.Close()
	out := make([]*usersv1.User, 0)
	for rows.Next() {
		u := &usersv1.User{}
		if err := rows.Scan(&u.Id, &u.Name, &u.Email, &u.Status); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *usersServer) count(ctx context.Context, q *spisql.Query) (uint64, error) {
	withoutPaging := *q
	withoutPaging.Pagination = spisql.Pagination{}
	withoutPaging.Sorts = nil

	b := sq.Select("count(*)").From("users").PlaceholderFormat(sq.Dollar)
	stmt, args, err := spisqlsq.Apply(b, &withoutPaging, sqMapping).ToSql()
	if err != nil {
		return 0, err
	}
	var n uint64
	if err := s.db.QueryRowContext(ctx, stmt, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("query %q args=%v: %w", stmt, args, err)
	}
	return n, nil
}
