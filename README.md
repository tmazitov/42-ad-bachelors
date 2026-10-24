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

## Deploy (VPS, Docker Compose)

`compose.yaml` runs two containers: **web** (Caddy serving the built frontend, proxying `/api` to the backend,
with automatic HTTPS) and **backend** (Go + SQLite on a named volume, not exposed).

1. Point your domain's DNS at the VPS and open ports 80 and 443.
2. Register `https://<your-domain>/auth/callback` as the redirect URI of the 42 app.
3. Configure and start:
   ```sh
   cp .env.example .env                   # SITE_ADDRESS=<your-domain>, PUBLIC_URL=https://<your-domain>
   cp backend/.env.example backend/.env   # FT_CLIENT_ID, FT_CLIENT_SECRET
   docker compose up -d --build
   ```
4. Update: `git pull && docker compose up -d --build`. Logs: `docker compose logs -f backend`.

The database lives in the `backend-data` volume (`/data/app.db` in the container); back it up with e.g.
`docker compose cp backend:/data/app.db ./backup.db`.

## Author

tmazitov · 2026
