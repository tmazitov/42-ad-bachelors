# 42 AD Bachelor's Tracker
A web app for 42 school students to track their progress toward the AD Bachelor's degree.

Log in with your 42 account to see which modules you've completed, how many credits you've earned per category, and how close you are to the 132-credit graduation requirement.

![42 AD](./frontend/public/42-bachelors.png)

## Stack

- Frontend: Vue 3 + TypeScript + Vite, PrimeVue + Tailwind CSS v4, Chart.js
- Backend: Go + Fiber v3 + SQLite (`backend/`), 42 Network OAuth 2.0 API

## Setup

1. Create a 42 OAuth app and register the redirect URI `http://localhost:5173/auth/callback`.
2. Backend:
   ```sh
   cd backend
   cp .env.example .env   # set FT_CLIENT_ID and FT_CLIENT_SECRET
   make run               # http://localhost:8080
   ```
3. Frontend (no env needed; Vite proxies `/api` to the backend):
   ```sh
   cd frontend
   npm install
   npm run dev            # http://localhost:5173
   ```

## Deploy (Docker Compose, e.g. Coolify)

`compose.yaml` runs three containers. Only **nginx** is an entry point; it routes by URL prefix:

| Prefix | Service |
|---|---|
| `/api/*` | **backend** (Go + SQLite on the `backend-data` volume) |
| `/*` | **frontend** (static SPA) |

No service publishes host ports and the frontend and backend are only reachable from nginx (separate Docker
networks). nginx speaks plain HTTP: the platform's proxy (Coolify) terminates HTTPS and forwards to it.

On Coolify:
1. Create a resource from this repository using the **Docker Compose** build pack (`compose.yaml`).
2. Assign your domain to the **nginx** service with its port, e.g. `https://<your-domain>:8088` (`NGINX_PORT`,
   default 8088, so it never collides with the Coolify dashboard on the host's 8000; users still connect on 443). Coolify issues the Let's Encrypt certificate.
3. Set the environment variables from `.env.example`: `PUBLIC_URL=https://<your-domain>`, `FT_CLIENT_ID`,
   `FT_CLIENT_SECRET` (optional tuning values have defaults).
4. Register `https://<your-domain>/auth/callback` as the redirect URI of the 42 app, then deploy.

Elsewhere: copy `.env.example` to `.env`, fill it in, put a TLS proxy in front of nginx (or add a `ports:`
mapping for nginx in an override file) and run `docker compose up -d --build`.

The database lives in the `backend-data` volume (`/data/app.db` in the container); back it up with e.g.
`docker compose cp backend:/data/app.db ./backup.db`.

## Author

tmazitov · 2026
