# Backend

Go + [Fiber v3](https://gofiber.io) + SQLite service for the 42 AD Bachelor's Tracker.
It replaces the Cloudflare Worker (see `../docs/ARCHITECTURE.md` §4 and §9) and tracks student progress.

## Run

```sh
cp .env.example .env   # optional, defaults work for local dev
make run               # http://localhost:8080
make test
```

Go 1.26+ is required. With `GOTOOLCHAIN=auto` (the default), an older `go` downloads it automatically.
The SQLite driver is pure Go (`modernc.org/sqlite`), so no CGO or system libsqlite is needed.

Docker: `make docker && docker run -p 8080:8080 -v ad-data:/data 42-ad-bachelors-backend`

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | `{"status":"ok"}`; 503 if the database is unreachable |
| GET | `/api/v1/curriculum` | Modules grouped by category, with credits and 42 project IDs |

Errors are always JSON: `{"error": "message"}`.

## Architecture (DDD)

```
cmd/server/            Composition root: config → DB → migrations → HTTP; graceful shutdown
internal/
  domain/              Business rules. Pure Go: no Fiber, no SQL, no HTTP clients.
    curriculum/        Module, Category, Curriculum, Progress; AD Bachelor data; credit calculation
  application/         (next) Use cases that orchestrate domain + ports, e.g. login, sync, leaderboard
  infrastructure/      Adapters to the outside world
    sqlite/            Connection, embedded migrations, repository implementations
  interfaces/http/     Fiber app, middleware, handlers, request/response DTOs
  config/              Env-based configuration
```

Dependency rule: `interfaces` → `application` → `domain` ← `infrastructure`.
- The **domain** defines repository and gateway interfaces (ports) next to the aggregates that need them,
  for example `student.Repository`.
- **Infrastructure** implements them: `sqlite.StudentRepository`, and a 42 API client in `infrastructure/ft`.
- **Application** services take the interfaces in their constructors. `cmd/server` wires the concrete types.
- **Handlers** convert HTTP to and from DTOs and call application services. They never touch SQL or domain internals.

### Conventions
- One package per bounded context or aggregate under `domain/`. Name packages after the concept, not the layer.
- Handlers are constructors returning `fiber.Handler`, closing over their dependencies.
- Return `fiber.NewError(code, msg)` for client-facing errors. Any other error is logged and becomes a 500.
- Migrations: add `internal/infrastructure/sqlite/migrations/NNNN_description.sql`. They are embedded in the
  binary and applied once, in order, at startup. Never edit an applied migration.
- SQLite runs in WAL mode with foreign keys on and a single connection (one writer).
