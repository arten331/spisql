# gin-postgres example

A production-ready list endpoint example using:
- **Framework**: Gin
- **Database**: PostgreSQL  
- **Query builder**: Squirrel
- **Feature**: Full pagination with total count

## Features

✓ Proper `JSONResponseList` response structure with metadata  
✓ Total count query for accurate pagination UI  
✓ Comprehensive error handling  
✓ Swagger-compatible struct tags  
✓ Real pgx database execution  
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
# List all users (first 10)
curl 'http://localhost:8080/users'

# Filter by name (substring match, case-insensitive)
curl 'http://localhost:8080/users?name=john'

# Filter by creation date
curl 'http://localhost:8080/users?created_at[gte]=2024-01-01'

# Sort descending by date, then ascending by name
curl 'http://localhost:8080/users?sort=-created_at,name'

# Combine filters and pagination
curl 'http://localhost:8080/users?name=alice&sort=-created_at&limit=5&offset=0'

# With email filter
curl 'http://localhost:8080/users?email=alice@example.com'
```

## Response Structure

```json
{
  "total": 10,
  "limit": 10,
  "offset": 0,
  "list": [
    {
      "id": 1,
      "name": "Alice Johnson",
      "email": "alice@example.com",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

## Key Points

- **Total count**: The `total` field includes count of records matching filters, before pagination
- **Pagination metadata**: `limit` and `offset` echo back the request values for UI builders
- **Silent drops**: Unknown fields or operators are silently ignored (safe by design)
- **Error handling**: Database errors return 500 with a generic message (never expose internals)
- **Timeouts**: All database operations have 5s timeout to prevent hanging

## Cleanup

```sh
docker-compose down
```
