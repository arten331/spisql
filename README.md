# spisql

[![Go Reference](https://pkg.go.dev/badge/github.com/arten331/spisql.svg)](https://pkg.go.dev/github.com/arten331/spisql)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`spisql` принимает list-эндпоинт-запросы (фильтры, сортировки, лимиты,
смещения), приводит их к единому Go-представлению и транслирует в
PostgreSQL, ClickHouse, MongoDB и другие хранилища.

Один формат на проводе, одна модель в Go, много бэкендов БД.

Корневой модуль — zero-dependency. Адаптеры под драйверы и хелперы под
HTTP-фреймворки лежат в подмодулях, поэтому импорт самого `spisql`
никогда не тянет лишних зависимостей.

## Формат запроса

Плоский bracket-формат в духе Stripe / JSON:API. Каждый фильтр — это
свой URL-ключ; URL-кодирование разруливает разделители в значениях;
повторяющиеся ключи дают список значений.

```
GET /users?status=active                        # bare value -> eq
          &created_at[gte]=2024-01-01           # оператор в скобках
          &id[in]=1&id[in]=2&id[in]=3           # повтор ключа = список
          &city[eq]=Paris%2C%20France           # запятая в значении ок
          &sort=-created_at,name                # префикс `-` = DESC
          &limit=25
          &offset=50
```

`limit`, `offset` и `sort` — зарезервированные ключи верхнего уровня.
Неизвестные операторы и битые ключи **молча игнорируются**;
`parser.Parse` не возвращает ошибки.

## Операторы

| Оператор          | Wire-имя       | Значения       |
|-------------------|----------------|----------------|
| Равно             | `eq`           | 1              |
| Не равно          | `ne`           | 1              |
| In / Not In       | `in` / `nin`   | 1+ (повтор ключа) |
| `>` `>=` `<` `<=` | `gt` `gte` `lt` `lte` | 1       |
| Дата `>=` ISO     | `gtedate`      | 1              |
| Между / не между  | `between` / `nbetween` | 2 (через запятую) |
| LIKE-substring    | `substringof`  | 1              |
| LIKE-prefix       | `startswith`   | 1              |
| LIKE-suffix       | `endswith`     | 1              |
| IS [NOT] NULL     | `isnull` / `isnotnull` | 0      |

## Быстрый старт

```go
import (
    "net/http"

    sq "github.com/Masterminds/squirrel"

    spisqlsq "github.com/arten331/spisql/adapter/squirrel"
    "github.com/arten331/spisql/parser"
    spisqlhttp "github.com/arten331/spisql/transport/nethttp"
)

// Маппинг — это whitelist: только эти поля будут отрендерены адаптером.
var mapping = spisqlsq.Mapping{
    "name":       "u.name",
    "email":      "u.email",
    "created_at": "u.created_at",
}

func listUsers(w http.ResponseWriter, r *http.Request) {
    q := spisqlhttp.Parse(r, parser.Options{
        Filter:       parser.FilterOptions{"name": {}, "email": {}, "created_at": {}},
        Sort:         parser.SortOptions{"created_at": {}, "name": {}},
        DefaultLimit: 25,
        MaxLimit:     100,
    })

    builder := sq.Select("u.id", "u.name", "u.email").
        From("users u").
        PlaceholderFormat(sq.Dollar)

    sql, args, _ := spisqlsq.Apply(builder, q, mapping).ToSql()
    // sql / args отправляются в pgx / sqlx / database/sql
    _ = sql; _ = args
}
```

Готовые сервисы (Gin/Postgres, Gin/GORM, Gin/Mongo, чистый `net/http`,
ручной билдер): [`examples/`](./examples/).

## MongoDB

```go
import (
    spisqlmongo "github.com/arten331/spisql/adapter/mongo"
)

var mapping = spisqlmongo.Mapping{
    "name":       "name",
    "email":      "email",
    "created_at": "created_at",
}

res := spisqlmongo.Build(q, mapping)
cur, _ := coll.Find(ctx, res.Filter, res.FindOptions())
```

Для агрегейшна — `res.Pipeline()` отдаёт массив стадий с
`$match`/`$sort`/`$skip`/`$limit`.

## GORM

```go
import (
    spisqlgorm "github.com/arten331/spisql/adapter/gorm"
)

var mapping = spisqlgorm.Mapping{
    "name":       "name",
    "created_at": "created_at",
}

var users []User
spisqlgorm.Apply(db, q, mapping).Find(&users)
```

## Свой билдер

Если адаптера под нужный data-layer нет — просто пройдись по
`*spisql.Query` руками:

```go
for _, f := range q.Filters {
    switch f.Operator {
    case spisql.OpEqual:
        // builder.Where("...", f.Values[0])
    case spisql.OpIn:
        // builder.Where("... IN (...)", f.Values)
    // ...
    }
}
```

Полный пример — [`examples/custom-builder`](./examples/custom-builder/).

## Адаптеры

| Путь                | Бэкенд                       | Точка входа                                                       |
|---------------------|------------------------------|-------------------------------------------------------------------|
| `adapter/raw`       | любой SQL через `database/sql` | `raw.Build(q, m) raw.Result`                                    |
| `adapter/squirrel`  | PostgreSQL, ClickHouse       | `squirrel.Apply(b, q, m) sq.SelectBuilder`                        |
| `adapter/goqu`      | PostgreSQL, MySQL, …         | `goqu.Apply(ds, q, m) *goqu.SelectDataset`                        |
| `adapter/bun`       | PostgreSQL, MySQL, ClickHouse | `bun.Apply(sq, q, m) *bun.SelectQuery`                           |
| `adapter/gorm`      | любой диалект GORM           | `gorm.Apply(db, q, m) *gorm.DB`                                   |
| `adapter/mongo`     | MongoDB                      | `mongo.Build(q, m) mongo.Result` (+ `FindOptions()` / `Pipeline()`) |

## Транспорты

| Путь                | Источник запроса                            |
|---------------------|---------------------------------------------|
| `transport/nethttp` | `*http.Request` (chi, gorilla, …)           |
| `transport/gin`     | `*gin.Context`                              |
| `transport/echo`    | `echo.Context` (v4)                         |
| `transport/fiber`   | `*fiber.Ctx` (v2)                           |
| `transport/grpc`    | `(filter, sort, limit, offset)` — для grpc-gateway |

Если фреймворка нет в списке — достань `url.Values` сам и позови
`parser.Parse(values, opts)`. Обёртка — пять строк.

## gRPC + grpc-gateway

Канонический wire-формат — `spisql.v1.ListQuery` из
`proto/spisqlv1/list.proto`. Встраивай его в свои request-сообщения и
передай четыре поля в `transport/grpc`:

```go
import spisqlgrpc "github.com/arten331/spisql/transport/grpc"

func (s *Server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
    q := spisqlgrpc.Parse(req.Query.Filter, req.Query.Sort, req.Query.Limit, req.Query.Offset, opts)
    // q работает с любым адаптером
    _ = q
    return nil, nil
}
```

`transport/grpc` не импортирует сгенерённый proto-код, поэтому
встраивать `spisql.v1.ListQuery` — соглашение, а не требование.

## Whitelist на двух уровнях

Защита от того, чтобы клиент шарился по скрытым колонкам:

- **`parser.Options.Filter` / `Options.Sort`** — публичный API-гард.
  `nil` означает «принимать всё»; не-nil карта молча отбрасывает
  неизвестные поля.
- **`Mapping` адаптера** — парсер поле уже принял, но адаптер не знает,
  как отрендерить. Незамапленные ключи отбрасываются молча.

Оба слоя дропают тихо — это намеренно.

## Документация

- [`docs/reference.md`](./docs/reference.md) — операторы, адаптеры,
  транспорты, опции, контракт для своего билдера
- [`docs/parser.md`](./docs/parser.md) — wire-формат, whitelist,
  поведение при ошибках, edge-cases
- [`examples/`](./examples/) — рабочие сервисы

## Лицензия

MIT — см. [LICENSE](LICENSE).
