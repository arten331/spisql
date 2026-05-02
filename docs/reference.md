# Reference

One page. Three tables. Wire format → adapter → transport.

For wire-format edge cases (whitelisting, malformed input, the
silent-drop model), see [Wire Format & Parser](./parser.md).
For runnable services, see [`examples/`](../examples/).

## The pipeline

```
HTTP query string → transport.Parse → *spisql.Query → adapter → DB
```

You wire two things:

```go
opts := parser.Options{
    Filter:       parser.FilterOptions{"name": {}, "email": {}}, // nil = allow any
    Sort:         parser.SortOptions{"created_at": {}},          // nil = allow any
    DefaultLimit: 25,                                             // 0 → 10
    MaxLimit:     200,                                            // 0 → 100
}

mapping := someAdapter.Mapping{
    "name":       "users.name",          // API field → DB column expression
    "email":      "users.email",
    "created_at": "users.created_at",
}
```

Then in your handler:

```go
q := transport.Parse(req, opts)        // turn the request into *spisql.Query
out := someAdapter.Apply(b, q, mapping) // or .Build(q, mapping) — see table below
```

## Operators

| Operator      | Wire example                                       | What it does                                               |
|---------------|----------------------------------------------------|------------------------------------------------------------|
| `eq` *(default)* | `?status=active` or `?status[eq]=active`         | Equality                                                   |
| `ne`          | `?status[ne]=banned`                               | Not equal                                                  |
| `gt`          | `?views[gt]=100`                                   | Greater than                                               |
| `gte`         | `?created_at[gte]=2024-01-01`                      | Greater than or equal                                      |
| `lt` / `lte`  | `?price[lt]=50`, `?price[lte]=50`                  | Less than / or equal                                       |
| `gtedate`     | `?date[gtedate]=2024-01-15`                        | Like `gte`; on Mongo strictly parsed as `YYYY-MM-DD`        |
| `in`          | `?id[in]=1&id[in]=2`                               | Match any of the values *(repeat the key)*                 |
| `nin`         | `?role[nin]=guest&role[nin]=banned`                | None of                                                    |
| `substringof` | `?title[substringof]=golang`                       | Case-insensitive contains                                  |
| `startswith`  | `?name[startswith]=An`                             | Case-insensitive prefix                                    |
| `endswith`    | `?file[endswith]=.pdf`                             | Case-insensitive suffix                                    |
| `isnull`      | `?deleted_at[isnull]=`                             | IS NULL *(value ignored)*                                  |
| `isnotnull`   | `?published_at[isnotnull]=`                        | IS NOT NULL *(value ignored)*                              |
| `between`     | `?price[between]=10&price[between]=100`            | Inclusive range, **low first**                             |
| `nbetween`    | `?price[nbetween]=10&price[nbetween]=100`          | Outside range                                              |

**Sort:** `?sort=-created_at,name` — comma-separated, prefix `-` for DESC.
Multiple `sort=` parameters are merged.

**Pagination:** `?limit=N&offset=M`. `limit` is clamped to `MaxLimit`;
`limit=0` or missing falls back to `DefaultLimit`.

**Reserved keys:** `limit`, `offset`, `sort` can't be used as bare filter
keys — use the bracket form (`?limit[eq]=42`) if you really have a column
named `limit`.

## Adapters

All adapters declare `type Mapping map[spisql.Column]string`. Filters
and sorts whose key is missing from the mapping are silently dropped.

| Adapter    | Targets                       | Import                                       | Entry point                                                                 |
|------------|-------------------------------|----------------------------------------------|-----------------------------------------------------------------------------|
| `raw`      | any SQL via `database/sql`    | `github.com/arten331/spisql/adapter/raw`     | `raw.Build(q, m) raw.Result`                                                |
| `squirrel` | Postgres, ClickHouse          | `github.com/arten331/spisql/adapter/squirrel`| `squirrel.Apply(b, q, m) sq.SelectBuilder`                                  |
| `goqu`     | Postgres, MySQL, SQLite, ...  | `github.com/arten331/spisql/adapter/goqu`    | `goqu.Apply(ds, q, m) *goqu.SelectDataset`                                  |
| `bun`      | Postgres, MySQL, ClickHouse   | `github.com/arten331/spisql/adapter/bun`     | `bun.Apply(sq, q, m) *bun.SelectQuery`                                      |
| `gorm`     | any GORM dialect              | `github.com/arten331/spisql/adapter/gorm`    | `gorm.Apply(db, q, m) *gorm.DB`                                             |
| `mongo`    | MongoDB                       | `github.com/arten331/spisql/adapter/mongo`   | `mongo.Build(q, m) mongo.Result` *(also exposes `FindOptions()` / `Pipeline()`)* |

### Builder-style (`Apply`)

```go
b := sq.Select("id", "name").From("users").PlaceholderFormat(sq.Dollar)
b = sqlsq.Apply(b, q, mapping)
sql, args, _ := b.ToSql()
```

The same shape works for `goqu`, `bun`, `gorm` — substitute the builder.

### Result-style (`Build`)

```go
res := sqlraw.Build(q, mapping)
// res.Where, res.Args, res.OrderBy, res.Limit, res.Offset
// For Postgres: query := sqlraw.Dollar(query) to convert ? → $N
```

```go
res := sqlmongo.Build(q, mapping)
cur, _ := coll.Find(ctx, res.Filter, res.FindOptions())
// or, for aggregation: append({"$match": res.Filter}, res.Pipeline()...)
```

## Transports

All return `*spisql.Query`. None of them fail.

| Transport | Framework                       | Import                                       | Call                                                       |
|-----------|---------------------------------|----------------------------------------------|------------------------------------------------------------|
| `nethttp` | `net/http`, chi, gorilla, ...   | `github.com/arten331/spisql/transport/nethttp` | `nethttp.Parse(r, opts)`                                  |
| `gin`     | Gin                             | `github.com/arten331/spisql/transport/gin`     | `sqlgin.Parse(c, opts)`                                   |
| `echo`    | Echo v4                         | `github.com/arten331/spisql/transport/echo`    | `sqlecho.Parse(c, opts)`                                  |
| `fiber`   | Fiber v2                        | `github.com/arten331/spisql/transport/fiber`   | `sqlfiber.Parse(c, opts)`                                 |
| `grpc`    | grpc-gateway                    | `github.com/arten331/spisql/transport/grpc`    | `sqlgrpc.Parse(filter, sort, limit, offset, opts)`        |

If your framework exposes `*http.Request`, just use `nethttp` — that's
chi, gorilla/mux, httprouter, kit, goa, etc. Otherwise the wrapper is
five lines: extract `url.Values`, call `parser.Parse(values, opts)`.

## Writing your own builder

You don't need any of the bundled adapters. The contract is:

```go
type Mapping map[spisql.Column]string

func Build(q *spisql.Query, m Mapping) YourResult {
    for _, f := range q.Filters {
        col, ok := m[f.Key]
        if !ok { continue }
        switch f.Operator {
        case spisql.OpEqual:    /* ... */
        case spisql.OpIn:       /* ... */
        case spisql.OpIsNull:   /* ... */
        // ...
        }
    }
    // q.Sorts, q.Pagination.{Limit,Offset} likewise
}
```

Two rules: drop on unknown mapping key, drop on unknown operator. A
runnable end-to-end version lives in
[`examples/custom-builder/`](../examples/custom-builder/) — the
`builder.go` file there is the full reference implementation; `main.go`
wires it into a `net/http` handler. No database is required to run it.
