# Profile Aggregator (Go)

Small HTTP service that aggregates user profile data from multiple mocked external sources, merging fields by priority.

Requirements satisfied:
- GET endpoint over HTTP, accepts `id` in either path `/profile/{id}` or query `/profile?id={id}`
- `id` validated as UUID (github.com/google/uuid)
- 4 mocked data sources with different fields and per-field priorities; lower number = higher priority (0 is highest)
- If a source is unavailable (timeout/cancel), it returns empty data; aggregator logs the error and continues
- Easy to add a 5th source by implementing `DataSource` and wiring it in main
- Returns JSON; unknown fields are supported without code changes (dynamic map)
- Can be run with `go run ./cmd/server` and tested in a browser

## Commands

### Server
Runs the profile aggregator service:
```bash
go run ./cmd/server
```

### Cache Cleanup
A standalone command to remove stale profiles from Redis (intended to be run via cron):
```bash
go run ./cmd/cleanup
```
It deletes entries older than 4 hours by default.

## Architecture and Scaling for High Load

This system is designed considering the processing of millions of profiles and Clean Architecture requirements.

### Scaling to Millions of Users

To scale for millions of users during the day:

1.  **Redis Cluster**: It is recommended to use Redis Cluster for horizontal scaling of the cache storage. This allows data to be distributed across multiple nodes and ensures high availability.
2.  **Memory Optimization**: Since profiles can be numerous, it is important to monitor memory usage. Using "smeared cleanup" (Cleanup Command) helps keep only relevant data.
3.  **Concurrency Control**: The aggregator uses goroutines for parallel requests to sources, ensuring minimal response time.

### Security and Legal Compliance

1.  **Protected Connection**: TLS support for the Redis connection ensures data encryption during transit.

### Storage Optimization

1.  **Data Compression**: Profile data is compressed using GZIP before being saved in Redis. This significantly reduces the memory required to store millions of records and lowers infrastructure costs.

### Multi-Tenancy / Multi-Product

The system supports data and logic isolation for different clients/products:
-   **Cache Isolation**: Redis keys include `clientID`, which guarantees data separation.
-   **Source Configuration**: The `SetClientSources` method allows defining a specific set of sources for each product without changing the main API contract.

### Pre-warming (Caching)

Profiles can be pre-loaded into the cache:
1.  **Client-driven**: The first request from a client initiates aggregation and storage.
2.  **Event-driven**: `EventBusConsumer` allows "warming up" the cache based on system events (e.g., user registration or data changes in the monolith).

- Path param:
  http://localhost:8080/profile/550e8400-e29b-41d4-a716-446655440000

- Query param:
  http://localhost:8080/profile?id=550e8400-e29b-41d4-a716-446655440000

Expected response (values aggregated by priorities):

```
{
  "avatar_url": "https://i.pravatar.cc/300",
  "email": "test@test.com",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "John Foo",
  "unknown": "alien"
}
```

Notes:
- Priorities are applied per field across sources. Lower number wins.
- Each source is called concurrently with a per-source timeout of 200ms; the overall request timeout is 500ms.
- To add a 5th source, create a new type that implements:

```go
Fetch(ctx context.Context, id uuid.UUID) (map[string]DataPoint, error)
Name() string
```

and add it to `NewAggregator(...)` in `cmd/server/main.go`.
