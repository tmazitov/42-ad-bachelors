# Backend

Go + [Fiber v3](https://gofiber.io) + SQLite service for the 42 AD Bachelor's Tracker.
It replaces the Cloudflare Worker (see `../docs/ARCHITECTURE.md` §4 and §9) and tracks student progress.

## Run

```sh
cp .env.example .env   # fill in FT_CLIENT_ID / FT_CLIENT_SECRET, register the callback on 42
make run               # http://localhost:8080
make test
```

Go 1.26+ is required. With `GOTOOLCHAIN=auto` (the default), an older `go` downloads it automatically.
The SQLite driver is pure Go (`modernc.org/sqlite`), so no CGO or system libsqlite is needed.

Docker: in production use the root `compose.yaml` (nginx entry point, frontend, backend). The image's
`/server healthcheck` subcommand probes `/healthz` for the container health check.

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/healthz` | `{"status":"ok"}`; 503 if the database is unreachable |
| GET | `/api/v1/curriculum` | Modules grouped by category, with credits and 42 project IDs |
| GET | `/api/v1/auth/login` | Redirects to 42 to sign in (sets an `oauth_state` cookie) |
| POST | `/api/v1/auth/callback` | `{"code","state","error"}` from the app's `/auth/callback` page; sets the httpOnly `session` cookie (204), or 400 `{"error":"denied"\|"state"\|"failed"}` |
| POST | `/api/v1/auth/logout` | Ends the session (204) |
| GET | `/api/v1/me` 🔒 | Signed-in student: profile, progress, validated projects |
| GET | `/api/v1/students/:login` 🔒 | Any 42 student; fetched from 42 and stored if unknown. 404 if 42 has no such login |
| GET | `/api/v1/students?search=&page=` 🔒 | Registered students matching login or display name, 10 per page |
| GET | `/api/v1/leaderboard` 🔒 | Synced registered students ranked by credits (ties share a rank) |
| GET | `/api/v1/avatar?url=` | Session required. Proxies a `https://cdn.intra.42.fr/users/...` image. The CDN requires auth, so `<img>` can't load it directly. 400 for other URLs. Cached in a 50 MB LRU keyed by the URL without query/fragment; not-found answers cached 10 min |

🔒 = needs the session cookie (401 otherwise).

Errors are always JSON: `{"error": "message"}`.

## Architecture (DDD)

```
cmd/server/            Composition root: config → DB → migrations → HTTP; graceful shutdown
internal/
  domain/              Business rules. Pure Go: no Fiber, no SQL, no HTTP clients.
    curriculum/        Module, Category, Curriculum, Progress; AD Bachelor data; credit calculation
    student/           Student aggregate (+ validated projects, registration) + Repository port
    session/           Sign-in sessions (hashed tokens) + Repository port
    leaderboard/       Ranking of students by credits
  application/         Use cases that orchestrate domain + ports
    auth/              OAuth sign-in, sessions, logout; Provider port
    students/          Profiles and search from the DB, refreshed from 42 when stale; ProfileSource port
    progress/          Project sync, leaderboard; ProjectSource port
    avatar/            42 CDN image proxy with a byte-capped LRU cache; Source port
  infrastructure/      Adapters to the outside world
    ft/                42 intra API client, rate-limited, cached app token (implements auth.Provider,
                       students.ProfileSource, progress.ProjectSource, avatar.Source)
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

### Auth and freshness
Sign-in is done entirely by the backend (`application/auth`): it exchanges the OAuth code, registers the
student, syncs their projects and stores a session (only the SHA-256 of the cookie token). The browser never
sees a 42 token and never calls the 42 API. The frontend must be served from `PUBLIC_URL`, the same origin as
the API, so the cookie is first-party. 42 redirects to the app's `/auth/callback` page (it shows a loader while
the backend signs the user in); register `PUBLIC_URL/auth/callback` on the 42 app.

`application/students` serves profiles from SQLite and refreshes them from 42 with the app token:
- registered students are answered from the database at once; data older than `STUDENT_MAX_AGE` (default 60m) is
  refreshed in the background
- other students (only viewed) are refreshed before responding when their data is older than `VISITOR_MAX_AGE`
  (default 5m); if 42 fails, the stored data is returned. They are stored but not registered.

Concurrent requests share one refresh (singleflight). Logins 42 does not know are remembered for 1 hour
(`application/cache`), so repeated lookups of them cost no API calls.

### 42 API budget
All 42 calls share one rate limiter (`FT_RATE_PER_SECOND`, `FT_RATE_PER_HOUR`).
- A call waits at most 5s for a token or a 429 `Retry-After`. Longer waits fail at once with `upstream.ErrBusy`,
  which the API answers as 503 (`{"error":"busy"}` on the sign-in callback).
- Background work (refreshes, lookups, the sync; marked with `upstream.Background`) leaves the last 10% of the
  hourly budget to sign-ins.

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
