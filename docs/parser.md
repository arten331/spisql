# Wire Format & Parser

The parser is the single source of truth for what HTTP clients can put on
the wire. This page is the contract: every transport feeds `url.Values`
to `parser.Parse(values, opts) *spisql.Query`, and adapters consume the
result without re-parsing anything.

For operator semantics, the adapter/transport tables, and the contract
for writing your own builder, see [Reference](./reference.md).

## Filter syntax

### Bare key — equality

A query parameter with no brackets is an `eq` filter:

```
?status=active&email=foo%40x.com
   ↓
Filter{Key: "status", Operator: OpEqual, Values: ["active"]}
Filter{Key: "email",  Operator: OpEqual, Values: ["foo@x.com"]}
```

### Bracket form — explicit operator

```
?key[op]=value
```

Examples:

```
?created_at[gte]=2024-01-01    → Operator: OpGreaterThanEqual
?status[ne]=banned             → Operator: OpNotEqual
?title[substringof]=golang     → Operator: OpSubstringOf
```

The bracket regex is strict: `^([^\[\]]+)\[([^\[\]]+)\]$`. Exactly one
`[op]` group, no nesting, no empty parts.

### Repeated keys — multi-value operators

For operators that take a list (`in`, `nin`) or a pair (`between`,
`nbetween`), repeat the same key. Order is preserved:

```
?id[in]=1&id[in]=2&id[in]=3
   → Filter{Key: "id", Operator: OpIn, Values: ["1", "2", "3"]}

?price[between]=10&price[between]=100
   → Filter{Key: "price", Operator: OpBetween, Values: ["10", "100"]}
```

`between` / `nbetween` expect **exactly two** values, **low first**.
The parser doesn't enforce this — adapters drop the filter if the count
is wrong, and `?price[between]=100&price[between]=10` will produce an
adapter clause that treats `100` as the low bound and matches nothing.

### Nullary operators — value is ignored

`isnull` and `isnotnull` take no value. Whatever you send for URL-form
purposes is discarded:

```
?deleted_at[isnull]=        ← empty
?deleted_at[isnull]=1
?deleted_at[isnull]=anything
   all three →
Filter{Key: "deleted_at", Operator: OpIsNull, Values: nil}
```

`Operator.IsNullary()` is the authoritative check.

## Sort syntax

```
?sort=<token>[,<token>...]
```

Each token is an optional `-` (DESC) or `+` (ASC, the default) followed
by a column name:

```
?sort=-created_at,name
   → Sort{Key: "created_at", Direction: DirectionDesc}
   → Sort{Key: "name",       Direction: DirectionAsc}
```

Multiple `sort=` parameters are merged in order:

```
?sort=-created_at&sort=name   ≡   ?sort=-created_at,name
```

Empty tokens (`?sort=,name,`) are skipped.

## Pagination

```
?limit=N&offset=M
```

Rules:

| Input                          | Result                                              |
|--------------------------------|------------------------------------------------------|
| `limit` missing or unparseable | `Options.DefaultLimit` (or `10` if that's `0`)       |
| `limit=0`                      | `Options.DefaultLimit` (`0` is treated as missing)   |
| `limit=N` where `N > MaxLimit` | clamped to `Options.MaxLimit` (or `100` if `0`)      |
| `offset` missing or unparseable| `0`                                                  |
| `offset=N`                     | `N` (no upper bound)                                 |

## Reserved keys

`limit`, `offset`, and `sort` are reserved at the top level — bare keys
with these names never become filters:

```
?limit=25&offset=50&sort=name   → no filters, only Pagination + Sorts
```

If a real database column happens to be called `limit`, you can still
filter on it via the bracket form (it bypasses the reserved-key check):

```
?limit[eq]=42   → Filter{Key: "limit", Operator: OpEqual, Values: ["42"]}
```

## Whitelisting (`Options.Filter` / `Options.Sort`)

```go
type Options struct {
    Filter       FilterOptions   // map[string]FilterOption
    Sort         SortOptions     // map[string]SortOption
    DefaultLimit uint64
    MaxLimit     uint64
}
```

Both maps follow the same rule:

- **`nil`** — every field is accepted (parser is wide open).
- **non-`nil`** — only keys present in the map are accepted; everything
  else is silently dropped.

Whitelisting is applied to **both bare and bracket forms**:

```go
opts := parser.Options{Filter: parser.FilterOptions{"name": {}}}
parser.Parse(values("name=v&secret=x&secret[eq]=y"), opts)
   → only Filter{Key: "name", ...} survives
```

This is the public-API guard so clients can't probe for hidden columns.
The adapter `Mapping` is a second whitelist (a filter whose key is
missing from the mapping is dropped at the adapter layer too) — see
[Reference → Adapters](./reference.md#adapters).

## Failure mode: silent drops

`parser.Parse` does **not** return an error. Every failure path is a
silent drop, by design:

| Input                                     | Result                                |
|-------------------------------------------|---------------------------------------|
| Unknown operator (`?name[unknown]=John`)  | parsed, dropped by adapters           |
| Field not in `Options.Filter` whitelist   | dropped at parse time                 |
| Field not in adapter `Mapping`            | dropped at adapter time               |
| Malformed bracket: `?name[]=v`            | dropped at parse time                 |
| Malformed bracket: `?[eq]=v`              | dropped at parse time                 |
| Malformed bracket: `?col[op][extra]=v`    | dropped at parse time                 |
| Wrong value count for `between`           | parsed, dropped by adapters           |
| Empty key (`?=value`)                     | dropped at parse time                 |
| Unparseable `limit` / `offset`            | falls back to default (not a drop)    |

If you want to *reject* unknown fields with a 400 instead of silently
dropping them, that's a deliberate departure from the library's
contract — wrap `parser.Parse` and inspect `url.Values` yourself before
calling it.

## Determinism

The parser sorts filters by their raw query-string key before emitting
them. Two requests with the same query parameters in different orders
produce the same `*spisql.Query`. This matters when you cache or hash
the resulting SQL.

## Worked examples

A small gallery of wire input → `*spisql.Query` shape (only the
interesting fields shown):

```
GET /users?name[substringof]=ann&created_at[gte]=2024-01-01&sort=-created_at&limit=20

  Filters:
    {Key: "created_at", Operator: OpGreaterThanEqual, Values: ["2024-01-01"]}
    {Key: "name",       Operator: OpSubstringOf,     Values: ["ann"]}
  Sorts:
    {Key: "created_at", Direction: DirectionDesc}
  Pagination: {Limit: 20, Offset: 0}
```

```
GET /orders?status[in]=pending&status[in]=shipped&total[between]=10&total[between]=500

  Filters:
    {Key: "status", Operator: OpIn,      Values: ["pending", "shipped"]}
    {Key: "total",  Operator: OpBetween, Values: ["10", "500"]}
  Sorts:        []
  Pagination:   {Limit: 10 (default), Offset: 0}
```

```
GET /articles?published_at[isnotnull]=&author=alice&sort=published_at,-id

  Filters:
    {Key: "author",       Operator: OpEqual,     Values: ["alice"]}
    {Key: "published_at", Operator: OpIsNotNull, Values: nil}
  Sorts:
    {Key: "published_at", Direction: DirectionAsc}
    {Key: "id",           Direction: DirectionDesc}
```
