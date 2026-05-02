# Examples

Runnable services demonstrating different adapters, transports, and
integration patterns.

## Examples

### [http-postgres](./http-postgres) — `net/http` + Squirrel, no DB needed

Bare-minimum HTTP handler that prints the SQL it would execute. The
fastest way to see what the wire format and the squirrel adapter
produce.

```sh
go run ./main.go
curl 'http://localhost:8080/users?name=john&created_at[gte]=2024-01-01&sort=-created_at&limit=10'
```

### [custom-builder](./custom-builder) — your own adapter, no DB needed

How to plug spisql into a service that doesn't use any of the bundled
adapters. `builder.go` is a hand-written `Build(*spisql.Query, Mapping)
Result` covering the full operator set; `main.go` wires it into a
`net/http` handler that prints the SQL it would execute.

```sh
go run .
curl 'http://localhost:8080/users?name[substringof]=ann&sort=-created_at'
```

### [gin-postgres](./gin-postgres) — PostgreSQL + Squirrel + Gin

Real pgx connections, full pagination response with total count,
schema auto-init, comprehensive error handling. Best when you need a
production-shaped REST API on PostgreSQL.

```sh
docker-compose up -d postgres
go run ./main.go
curl 'http://localhost:8080/users?created_at[gte]=2024-01-01&sort=-created_at&limit=10'
```

### [gin-gorm](./gin-gorm) — PostgreSQL + GORM + Gin

Same shape as `gin-postgres` but with GORM. Pick this when you want
automatic migrations, hooks, or the option to swap the dialect later.

```sh
docker-compose up -d postgres
go run ./main.go
curl 'http://localhost:8080/articles?status=published&sort=-created_at&limit=10'
```

### [gin-mongo](./gin-mongo) — MongoDB + Gin

Real MongoDB execution, BSON filter building, document-style filtering.
Pick this when you're on MongoDB.

```sh
docker-compose up -d mongo
go run ./main.go
curl 'http://localhost:8080/products?category=laptops&price[gte]=1000'
```

## Pagination response shape

The `gin-*` examples all use the same pagination envelope:

```go
type JSONResponseList[T any] struct {
    Total  uint64 `json:"total"`         // count after WHERE, before LIMIT
    Limit  uint64 `json:"limit"`         // echo of request `limit`
    Offset uint64 `json:"offset"`        // echo of request `offset`
    List   []T    `json:"list,omitempty"`
}
```

Both queries — `COUNT(*)` and the page itself — are built from the same
`*spisql.Query`, so they can never disagree about what "matching" means.

## Choosing an example

| Scenario                                         | Example          |
|--------------------------------------------------|------------------|
| Just want to see SQL on stdout                   | `http-postgres`  |
| Writing my own data layer / no bundled adapter   | `custom-builder` |
| REST API on PostgreSQL                           | `gin-postgres`   |
| Need ORM / migrations                            | `gin-gorm`       |
| MongoDB                                          | `gin-mongo`      |
| Different framework (Echo, Fiber, …)             | adapt `gin-postgres` |

## Reference

- [`docs/reference.md`](../docs/reference.md) — operator/adapter/transport tables
- [`docs/parser.md`](../docs/parser.md) — wire format & failure model
