# CloudBees Feature Flag Integration

This document explains how VinylVault uses [CloudBees Feature Management](https://www.cloudbees.com/capabilities/feature-management) to control UI features at runtime — no code changes, no redeployment required.

## What is CloudBees Feature Management?

CloudBees Feature Management (formerly Rollout) is an enterprise feature flag platform. Feature flags let you deploy code dark and then turn features on or off for specific environments or users from a dashboard — decoupling deployment from release.

This integration uses the `rox-browser` JavaScript SDK to connect the React frontend to the CloudBees platform.

## How It Works

```
CloudBees Dashboard
       │
       │  flag values (True/False per environment)
       ▼
rox-browser SDK  ←── polls https://rox-conf.cloudbees.io every ~30s
       │
       │  isEnabled()
       ▼
React useFlag() hook  ←── polls flag value every 5s
       │
       ▼
Component renders feature ON or OFF
```

1. On app startup, `initFlags()` in `src/flags.ts` registers the flags and calls `Rox.setup()` with the SDK key
2. The SDK fetches the current flag configuration from CloudBees
3. Each component uses the `useFlag()` hook to read its flag value
4. The hook polls every 5 seconds — flag changes appear in the app automatically without a page refresh

## The Three Flags

| Flag Name | Component | Effect |
|---|---|---|
| `dark_mode` | App-wide | Switches the entire UI to a dark purple theme |
| `grid_view` | Catalog tab | Switches the record list from a table to a card grid layout |
| `export_csv` | Collection tab | Shows an Export CSV button to download the customer's collection |

Each flag also shows a purple **CloudBees flag badge** in the UI when active, making it obvious during a demo which features are flag-controlled.

## Project Structure

```
web/
├── src/
│   ├── flags.ts          # Flag definitions + SDK initialization
│   ├── useFlag.ts        # React hook to read flag values
│   ├── App.tsx           # dark_mode flag consumer
│   ├── CatalogView.tsx   # grid_view flag consumer
│   └── CollectionView.tsx # export_csv flag consumer
├── nginx.conf            # CSP configured to allow cloudbees.io + rollout.io
└── .npmrc                # Points to public npm registry (not Roku Artifactory)
```

### `src/flags.ts`

Defines all flags and initializes the SDK:

```typescript
import Rox from 'rox-browser';

export const Flags = {
  dark_mode: new Rox.Flag(false),   // default: off
  grid_view: new Rox.Flag(false),
  export_csv: new Rox.Flag(false),
};

export async function initFlags() {
  Rox.register('', Flags);
  await Rox.setup(import.meta.env.VITE_CLOUDBEES_SDK_KEY);
}
```

Flag names in code must match exactly what is created in the CloudBees dashboard.

### `src/useFlag.ts`

A lightweight React hook that polls the flag value every 5 seconds:

```typescript
export function useFlag(flag: { isEnabled: () => boolean }): boolean {
  const [value, setValue] = useState(() => flag.isEnabled());

  useEffect(() => {
    const interval = setInterval(() => {
      const current = flag.isEnabled();
      setValue((prev) => (prev !== current ? current : prev));
    }, 5000);
    return () => clearInterval(interval);
  }, [flag]);

  return value;
}
```

## Setup

### Prerequisites

- A [CloudBees Feature Management](https://www.cloudbees.com/products/feature-management/pricing) account (free Community Edition works)
- Three flags created in the CloudBees dashboard: `dark_mode`, `grid_view`, `export_csv` (all Boolean type)
- An environment created (e.g. `development`) with an SDK key

### Environment Variable

Add your SDK key to `web/.env`:

```
VITE_CLOUDBEES_SDK_KEY=your-sdk-key-here
```

For Docker, the key is passed as a build argument in `docker-compose.yml`:

```yaml
web:
  build:
    context: ./web
    args:
      VITE_CLOUDBEES_SDK_KEY: your-sdk-key-here
```

> **Note:** The SDK key is a client-side key used to read flag values only — it does not grant write access to your CloudBees account. It is safe to include in a frontend build.

### Running Locally

```bash
# Dev server (flags work, hot reload)
cd web && npm run dev

# Full Docker stack (all services + flags)
docker compose up --build
```

### Turning a Flag On

1. Log into [cloudbees.io](https://cloudbees.io)
2. Go to **Feature Management → Flags**
3. Click the flag (e.g. `dark_mode`)
4. Click the **Configure** tab
5. Select your environment (e.g. `development`)
6. Set the toggle to **On** and **Set to: True**
7. Click **Save configuration**

The app picks up the change within 5 seconds — no redeploy needed.

## Content Security Policy

The CloudBees SDK makes outbound requests to `rox-conf.cloudbees.io` and `rox-state.cloudbees.io`. The nginx config (`web/nginx.conf`) includes both domains in the CSP `connect-src` and `script-src` directives to allow this.

## Why Feature Flags?

Feature flags solve a real problem: **separating deployment from release**.

Without flags, turning a feature on means deploying new code — which is risky and slow. With flags:

- Deploy the code dark (flag off) → no user impact
- Test in production with a small group before full rollout
- Turn the feature on instantly for all users when ready
- Turn it off immediately if something goes wrong — no rollback needed

For VinylVault, this demonstrates how an enterprise team would safely roll out UI changes like a new layout (`grid_view`) or a new capability (`export_csv`) to users without touching the codebase.
