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

In production, serve `frontend/dist` and the backend on one domain (reverse proxy `/api` → backend) and set
`PUBLIC_URL` to that domain.

## Author

tmazitov · 2026
