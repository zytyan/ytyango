# ytyango Frontend (Vue + Vite + SSG)

This Vite project renders a static Telegram WebApp UI for the `http_backend` APIs. It assumes the Telegram WebApp runtime injects `initData`, which is forwarded to the backend via the `Authorization: Telegram <initData>` header.

## Scripts

- `npm run dev` – start the dev server.
- `npm run build` – generate static HTML with Vite SSG.
- `npm run preview` – preview the built output.

Set `VITE_API_BASE_URL` to point at the HTTP backend (defaults to `http://127.0.0.1:4021`).
