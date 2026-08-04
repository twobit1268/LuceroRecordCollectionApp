# VinylVault web UI

React + TypeScript (Vite) frontend for VinylVault. See the [root README](../README.md#web-ui) for how this fits into the system and how to run it — this file just covers this directory's own commands.

```bash
npm install
npm run dev      # dev server at :5173, hits localhost:8081/8082/8083 by default
npm run lint      # oxlint
npm run build     # type-check + production build to dist/
```

`.env.example` documents the `VITE_*` env vars for pointing at different backend URLs — copy to `.env` to override the defaults.
