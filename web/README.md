# VinylVault web UI

React + TypeScript (Vite) frontend for VinylVault. See the [root README](../README.md#web-ui) for how this fits into the system and how to run it — this file just covers this directory's own commands.

```bash
npm install
npm run dev            # dev server at :5173, hits localhost:8081/8082/8083 by default
npm run lint            # oxlint
npm run build           # type-check + production build to dist/
npm run test:e2e        # Playwright — needs the full stack running (see e2e/README below)
npm run typecheck:e2e   # type-check web/e2e without running tests
```

`.env.example` documents the `VITE_*` env vars for pointing at different backend URLs — copy to `.env` to override the defaults.

## E2E tests

`e2e/` has a Playwright suite (Page Object Model — see `e2e/pages/`) that drives a real browser against the real stack. It needs everything running first, not just this dev server:

```bash
docker compose up --build -d   # from the repo root
npx playwright install chromium   # first time only
npm run test:e2e
```

`e2e/end-to-end-flow.spec.ts` is the one worth reading first — it walks the entire distributed flow (catalog create → collection add → async Pub/Sub-driven activity feed) through actual UI interactions, the browser equivalent of `scripts/smoke-test.sh` at the repo root.
