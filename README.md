# 42 AD Bachelor's Tracker
A web app for 42 school students to track their progress toward the AD Bachelor's degree.

Log in with your 42 account to see which modules you've completed, how many credits you've earned per category, and how close you are to the 132-credit graduation requirement.

![42 AD](./public/42-bachelors.png)

## Stack

- Vue 3 + TypeScript + Vite
- PrimeVue (UI components) + Tailwind CSS v4
- Chart.js (progress doughnut)
- 42 Network OAuth 2.0 API

## Setup

Create a `.env` file with your 42 API credentials:

```
VITE_42_CLIENT_ID=your_client_id
VITE_42_CLIENT_SECRET=your_client_secret
VITE_42_REDIRECT_URI=http://localhost:5173/auth/callback
```

Then install and run:

```sh
npm install
npm run dev
```

## Author

tmazitov · 2026
