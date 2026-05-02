# custom-builder

Shows how to write your own spisql adapter for plain `database/sql` —
no Squirrel, no GORM, no driver dependency in the example itself.

The interesting file is [`builder.go`](./builder.go): a single `Build`
function that walks `*spisql.Query`, looks each field up in a
`Mapping`, and switches on `f.Operator` to emit a `WHERE`-clause
fragment plus positional args. That's the entire contract every
adapter — bundled or hand-written — implements.

[`main.go`](./main.go) wires it into a `net/http` handler and prints the
SQL the handler would execute. No database is required to run it.

```sh
go run .
curl 'http://localhost:8080/users?name[substringof]=ann&created_at[gte]=2024-01-01&sort=-created_at&limit=20'
```

Output:

```
SQL : SELECT u.id, u.name, u.email FROM users u WHERE u.created_at >= $1 AND u.name LIKE $2 ORDER BY u.created_at DESC LIMIT 20
args: [2024-01-01 %ann%]
```

The two ground rules `Build` follows:

- If `mapping[f.Key]` is missing → drop the filter silently.
- If `f.Operator` isn't handled by the switch → drop the filter silently.

Both are deliberate — they're what makes the wire format safe to expose
publicly without leaking schema details.

For the full contract (every type the builder consumes), see
[`docs/reference.md`](../../docs/reference.md) and
[`docs/parser.md`](../../docs/parser.md).
