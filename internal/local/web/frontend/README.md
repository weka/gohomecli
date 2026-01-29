# Weka Home CLI Frontend

Simple React frontend for the Weka Home CLI Configurer web interface.

Built with [Vite](https://vitejs.dev/).

## Available Scripts

### `npm run dev` / `npm start`

Runs the app in development mode at [http://localhost:5173](http://localhost:5173).

API requests to `/api/*` are proxied to `http://localhost:8080`.

### `npm run build`

Builds the app for production to the `build` folder.

The build output is embedded into the Go binary via `//go:embed build`.

### `npm run preview`

Preview the production build locally.
