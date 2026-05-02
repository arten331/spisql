//go:build integration

package integration

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcmongo "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	mongoOnce sync.Once
	mongoCli  *mongo.Client
	mongoErr  error
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

func mongoColl(t *testing.T) *mongo.Collection {
	t.Helper()
	mongoOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		c, err := tcmongo.Run(ctx, "mongo:7")
		if err != nil {
			mongoErr = err
			return
		}
		uri, err := c.ConnectionString(ctx)
		if err != nil {
			mongoErr = err
			return
		}
		cli, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err != nil {
			mongoErr = err
			return
		}
		if err := cli.Ping(ctx, nil); err != nil {
			mongoErr = err
			return
		}
		mongoCli = cli
	})
	if mongoErr != nil {
		t.Skipf("mongo unavailable: %v", mongoErr)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	coll := mongoCli.Database("test").Collection("users")
	if err := coll.Drop(ctx); err != nil {
		t.Fatalf("drop: %v", err)
	}
	t1, _ := time.Parse(time.DateOnly, "2024-01-15")
	t2, _ := time.Parse(time.DateOnly, "2024-06-01")
	t3, _ := time.Parse(time.DateOnly, "2024-09-20")
	t4, _ := time.Parse(time.DateOnly, "2025-02-11")
	docs := []any{
		bson.D{{Key: "name", Value: "vasya"}, {Key: "email", Value: "v@x"}, {Key: "status", Value: "active"}, {Key: "created_at", Value: t1}},
		bson.D{{Key: "name", Value: "petya"}, {Key: "email", Value: "p@x"}, {Key: "status", Value: "active"}, {Key: "created_at", Value: t2}},
		bson.D{{Key: "name", Value: "masha"}, {Key: "email", Value: nil}, {Key: "status", Value: "deleted"}, {Key: "created_at", Value: t3}},
		bson.D{{Key: "name", Value: "kostya"}, {Key: "email", Value: "k@x"}, {Key: "status", Value: "banned"}, {Key: "created_at", Value: t4}},
	}
	if _, err := coll.InsertMany(ctx, docs); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return coll
}
