# gin-gorm example

A production-ready list endpoint example using:
- **Framework**: Gin
- **Database**: PostgreSQL
- **ORM**: GORM
- **Feature**: Full pagination with total count

## Features

✓ Proper `JSONResponseList` response structure with metadata  
✓ Total count query for accurate pagination UI  
✓ GORM for automatic schema management  
✓ Comprehensive error handling  
✓ Real database execution with timeouts  
✓ Schema initialization with sample data  

## Running

1. Start PostgreSQL:
```sh
docker-compose up -d postgres
```

2. Run the server:
```sh
go run ./main.go
```

3. Try requests:
```sh
# List all articles (first 10)
curl 'http://localhost:8080/articles'

# Filter by status
curl 'http://localhost:8080/articles?status=published'

# Filter by author (substring)
curl 'http://localhost:8080/articles?author=alice'

# Filter by creation date
curl 'http://localhost:8080/articles?created_at[gte]=2024-01-01'

# Sort descending by creation date
curl 'http://localhost:8080/articles?sort=-created_at'

# Combine filters with pagination
curl 'http://localhost:8080/articles?status=published&sort=-created_at&limit=5&offset=0'
```

## Response Structure

```json
{
  "total": 5,
  "limit": 10,
  "offset": 0,
  "list": [
    {
      "id": 1,
      "title": "Getting Started with Go",
      "status": "published",
      "author": "Alice",
      "created_at": "2024-02-15T10:30:00Z",
      "updated_at": "2024-02-15T10:30:00Z"
    }
  ]
}
```

## Key Points

- **GORM integration**: Use `sqlgorm.Apply()` to apply filters and sorts to GORM queries
- **Total count**: Query count using a filtered scope, without pagination applied
- **Error handling**: Safe error messages that don't expose internals
- **Pagination**: Both `limit` and `offset` work with GORM's `Limit()` and `Offset()` methods

## Cleanup

```sh
docker-compose down
```
