//go:build integration

package grpcgateway

import (
	"context"
	"database/sql"
	"net"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/jackc/pgx/v5/stdlib"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	usersv1 "github.com/arten331/spisql/tests/grpcgateway/proto/usersv1"
)

const seedSQL = `
CREATE TABLE IF NOT EXISTS users (
  id          BIGSERIAL PRIMARY KEY,
  name        TEXT NOT NULL,
  email       TEXT,
  status      TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
TRUNCATE users RESTART IDENTITY;
INSERT INTO users (name, email, status, created_at) VALUES
  ('vasya',  'v@x',  'active',  '2024-01-15'),
  ('petya',  'p@x',  'active',  '2024-06-01'),
  ('masha',  NULL,   'deleted', '2024-09-20'),
  ('kostya', 'k@x',  'banned',  '2025-02-11');
`

var (
	pgOnce sync.Once
	pgDB   *sql.DB
	pgErr  error
)

func pg(t *testing.T) *sql.DB {
	t.Helper()
	pgOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		c, err := tcpg.Run(ctx, "postgres:16-alpine",
			tcpg.WithDatabase("test"),
			tcpg.WithUsername("test"),
			tcpg.WithPassword("test"),
			tcpg.BasicWaitStrategies(),
		)
		if err != nil {
			pgErr = err
			return
		}
		dsn, err := c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			pgErr = err
			return
		}
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			pgErr = err
			return
		}
		if err := db.PingContext(ctx); err != nil {
			pgErr = err
			return
		}
		if _, err := db.ExecContext(ctx, seedSQL); err != nil {
			pgErr = err
			return
		}
		pgDB = db
	})
	if pgErr != nil {
		t.Skipf("postgres unavailable: %v", pgErr)
	}
	return pgDB
}

type stack struct {
	httpURL string
	close   func()
}

func startStack(t *testing.T, db *sql.DB) *stack {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	grpcSrv := grpc.NewServer()
	usersv1.RegisterUsersServer(grpcSrv, &usersServer{db: db})
	go func() { _ = grpcSrv.Serve(lis) }()

	mux := runtime.NewServeMux()
	ctx, cancel := context.WithCancel(context.Background())
	if err := usersv1.RegisterUsersHandlerFromEndpoint(
		ctx, mux, lis.Addr().String(),
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	); err != nil {
		cancel()
		grpcSrv.Stop()
		_ = lis.Close()
		t.Fatalf("register gateway: %v", err)
	}

	httpSrv := httptest.NewServer(mux)
	return &stack{
		httpURL: httpSrv.URL,
		close: func() {
			httpSrv.Close()
			cancel()
			grpcSrv.GracefulStop()
		},
	}
}
