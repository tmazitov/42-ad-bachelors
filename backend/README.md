# Backend

Go + [Fiber v3](https://gofiber.io) + SQLite service for the 42 AD Bachelor's Tracker.
It replaces the Cloudflare Worker (see `../docs/ARCHITECTURE.md` §4 and §9) and tracks student progress.

## Run

```sh
cp .env.example .env   # fill in FT_CLIENT_ID / FT_CLIENT_SECRET
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
| POST | `/api/v1/oauth/token` | `{"code"}` → 42 token JSON. Registers the student. 400 if the code is invalid or expired |
| POST | `/api/v1/oauth/refresh` | `{"refresh_token"}` → 42 token JSON. 401 if the refresh token is invalid or expired |
| GET | `/api/v1/leaderboard` | Synced students ranked by credits (ties share a rank) |
| GET | `/api/v1/avatar?url=` | Proxies a `https://cdn.intra.42.fr/users/...` image. The CDN requires auth, so `<img>` can't load it directly. 400 for other URLs |

Errors are always JSON: `{"error": "message"}`.

## Architecture (DDD)

```
cmd/server/            Composition root: config → DB → migrations → HTTP; graceful shutdown
internal/
  domain/              Business rules. Pure Go: no Fiber, no SQL, no HTTP clients.
    curriculum/        Module, Category, Curriculum, Progress; AD Bachelor data; credit calculation
    student/           Student aggregate (+ validated projects) + Repository port
    leaderboard/       Ranking of students by credits
  application/         Use cases that orchestrate domain + ports
    auth/              Login (code exchange + student registration), token refresh; Provider port
    progress/          Project sync, leaderboard; ProjectSource port
    avatar/            42 CDN image proxy with in-memory cache; Source port
  infrastructure/      Adapters to the outside world
    ft/                42 intra API client, rate-limited, cached app token (implements auth.Provider,
                       progress.ProjectSource, avatar.Source)
    sqlite/            Connection, embedded migrations, repository implementations
  interfaces/http/     Fiber app, middleware, handlers, request/response DTOs
  interfaces/scheduler/ Background jobs on an interval (progress sync)
  config/              Env-based configuration
```

Dependency rule: `interfaces` → `application` → `domain` ← `infrastructure`.
- The **domain** defines repository interfaces next to their aggregates (`student.Repository`).
- The **application** layer defines ports for external services its use cases need (`auth.Provider`).
- **Infrastructure** implements them: `sqlite.StudentRepository` and `ft.Client`.
- **Application** services take the interfaces in their constructors. `cmd/server` wires the concrete types.
- **Handlers** convert HTTP to and from DTOs and call application services. They never touch SQL or domain internals.

### Auth
The frontend stores 42's own tokens and keeps calling the 42 API with them. The backend only holds the
client secret, the same role the Cloudflare Worker had. The redirect URI comes from `FT_REDIRECT_URI`;
a `redirect_uri` sent by the client is ignored. On the frontend, set
`VITE_42_TOKEN_PROXY=http://localhost:8080/api/v1/oauth/token`; the refresh URL is derived from it.

### Background sync
`interfaces/scheduler` runs `progress.SyncAll` at startup and then every `SYNC_INTERVAL` (default `1h`; `0` turns it off).
Each run syncs, one at a time with a cached 42 app token (`client_credentials`), only the students whose last
sync is at least `SYNC_MAX_AGE` old (default `24h`) or who were never synced. Logging in syncs a student
immediately, so active students are skipped and only those who haven't logged in for a day are refreshed.
A failed student is logged and skipped; the run stops on shutdown and the server waits for it before closing the DB.
Logins also sync the signed-in student right away.

All 42 API calls go through two limiters in `ft.Client`: `FT_RATE_PER_SECOND` (default 2) and an hourly budget
`FT_RATE_PER_HOUR` (default 1200). A 429 is retried up to 3 times after `Retry-After`. Each synced student costs at
least one request, so syncing N students takes about N/2 seconds and N requests of the hourly budget.
CDN image requests are not counted.

### Conventions
- One package per bounded context or aggregate under `domain/`. Name packages after the concept, not the layer.
- Handlers are constructors returning `fiber.Handler`, closing over their dependencies.
- Return `fiber.NewError(code, msg)` for client-facing errors. Any other error is logged and becomes a 500.
- Migrations: add `internal/infrastructure/sqlite/migrations/NNNN_description.sql`. They are embedded in the
  binary and applied once, in order, at startup. Never edit an applied migration.
- SQLite runs in WAL mode with foreign keys on and a single connection (one writer).
