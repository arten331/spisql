# gin-mongo example

A production-ready list endpoint example using:
- **Framework**: Gin
- **Database**: MongoDB
- **Feature**: Full pagination with total count

## Features

✓ Proper `JSONResponseList` response structure with metadata  
✓ Total count query for accurate pagination UI  
✓ Comprehensive error handling  
✓ Real MongoDB execution via mongo-driver  
✓ Schema initialization with sample data  

## Running

1. Start MongoDB:
```sh
docker-compose up -d mongo
```

2. Run the server:
```sh
go run ./main.go
```

3. Try requests:
```sh
# List all products (first 10)
curl 'http://localhost:8080/products'

# Filter by name (substring match)
curl 'http://localhost:8080/products?name=laptop'

# Filter by category (exact match)
curl 'http://localhost:8080/products?category=laptops'

# Filter by minimum price
curl 'http://localhost:8080/products?price[gte]=1000'

# Sort descending by creation date
curl 'http://localhost:8080/products?sort=-created_at'

# Combine filters and pagination
curl 'http://localhost:8080/products?category=laptops&price[gte]=1000&limit=5&offset=0'
```

## Response Structure

```json
{
  "total": 5,
  "limit": 10,
  "offset": 0,
  "list": [
    {
      "id": "1",
      "name": "MacBook Pro",
      "category": "laptops",
      "price": 2499.99,
      "created_at": "2024-02-15T10:30:00Z"
    }
  ]
}
```

## Key Points

- **Total count**: The `total` field is the count before pagination for accurate UI
- **Pagination metadata**: Echo back `limit` and `offset` for client pagination builders
- **Field mapping**: Map API field names to MongoDB document field names
- **Error safety**: Generic error messages never expose internals

## Cleanup

```sh
docker-compose down
```
