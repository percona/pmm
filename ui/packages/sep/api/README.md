# @sep/api

Centralized HTTP client, React Query configuration, and OpenAPI type codegen
for the SEP frontend.

This package is consumed by `@sep/shell`, `@sep/framework`, and plugin
packages. Anything that talks to the backend should go through here.

## What's in the package

- **`apiClient`** — preconfigured axios instance with interceptors for Bearer
  token injection, unauthorized handling, dev logging, and structured error
  normalization into `ApiError`. Request/response bodies are passed through
  verbatim — field casing matches the OpenAPI spec.
- **Typed clients** — `mainApi`, `sepApi` are
  [`openapi-fetch`](https://openapi-ts.dev/openapi-fetch/) clients typed
  against the generated `paths` under `src/generated/`. They share
  Bearer-token and unauthorized handling with `apiClient` via a middleware
  that reads the same `setTokenProvider` state.
- **Generated types** — `src/generated/{main,inventory,tasks,sep}.ts` are
  emitted by `pnpm --filter @sep/api codegen` from the four FastAPI specs.
  Files are committed so builds don't require a live backend; they are
  marked `linguist-generated` in `frontend/.gitattributes` to keep GitHub
  diffs collapsed.
- **`createQueryClient()`** — factory that returns a `QueryClient` with the
  dashboard-tuned defaults: 30s stale time, exponential back-off retry that
  skips 4xx responses, `refetchOnWindowFocus: true`.
- **`ApiError`** — structured error class surfaced by the client. Every
  rejection from `apiClient` is an `ApiError`; `throwOnApiError()` adapts
  the `openapi-fetch` `{ data, error }` tuple to throw the same shape.
- **Token accessor pattern** — `setTokenProvider()` and `setOnUnauthorized()`
  let the auth layer plug in without the API package depending on auth state.
- **Hooks** — `usePluginSchema`, `usePluginTasks`, `usePluginTask`,
  `useCreatePluginTask` (generic, predate codegen) and `useCurrentUser`
  (sample of the typed-hook pattern).
- **Auth functions** — `postLogin`, `postRefresh`, `fetchCurrentUser`.
  Thin request wrappers consumed by the `AuthProvider` in `@sep/shell`.

## Usage

### Wire up the query client in the app root

```tsx
import { QueryClientProvider } from '@tanstack/react-query';
import { createQueryClient } from '@sep/api';

const queryClient = createQueryClient();

<QueryClientProvider client={queryClient}>
  <App />
</QueryClientProvider>;
```

### Wire up auth (from `@sep/shell`'s `AuthProvider`)

```ts
import { setTokenProvider, setOnUnauthorized } from '@sep/api';

setTokenProvider(() => currentAccessToken);
setOnUnauthorized(() => redirectToLogin());
```

### Call the API

```ts
import { apiClient, ApiError } from '@sep/api';

try {
  const { data } = await apiClient.get('/apps/checksums/');
} catch (err) {
  if (err instanceof ApiError && err.status === 404) {
    // handle not found
  }
}
```

### Handling errors in React Query

```ts
const { data, error } = useQuery({
  queryKey: ['x'],
  queryFn: () => apiClient.get('/x'),
});

if (error instanceof ApiError) {
  // error.kind, error.status, error.message are all reliable
}
```

## Conventions

### Field casing

The backend is FastAPI. Most wire payloads are snake_case; a few schemas
(notably `CasdoorUser`) are served with a camelCase alias generator. Either
way, the canonical names live in the OpenAPI spec and are reflected in
`src/generated/*.ts`. Read field names from the generated types — do not
apply client-side case conversion.

## Codegen

`pnpm --filter @sep/api codegen` reads the committed spec fixtures in
`specs/{main,inventory,tasks,sep}.json` and writes one typed client per spec
into `src/generated/`. No running backend is required by default.

```bash
# Default: regenerate the clients from the committed specs/*.json fixtures.
pnpm --filter @sep/api codegen
```

The specs are committed (and marked `linguist-generated`) so codegen, CI, and
local builds never need a live backend.

### Updating after a backend contract change

The fixtures are committed, so a contract change is a two-step regen — dump the
specs from the backend, then regenerate the clients:

```bash
# 1. From the repo root: dump the four whole-app OpenAPI specs (no server needed).
python scripts/dump_openapi.py
# 2. Regenerate the typed clients from the refreshed fixtures, then format them
#    (CI compares against the formatted output).
pnpm --filter @sep/api codegen
pnpm --filter @sep/api exec oxfmt --write src/generated
```

To regenerate directly from a live backend instead of the fixtures, point
codegen at it (format afterward, as the CI guard does):

```bash
CODEGEN_BASE_URL=http://localhost:8000 pnpm --filter @sep/api codegen
pnpm --filter @sep/api exec oxfmt --write src/generated
```

### Freshness is enforced in CI

Two guards keep the committed clients honest across the spec handoff:

- A backend test (`tests/app/test_openapi_specs_fresh.py`) fails if the
  committed `specs/*.json` drift from a fresh `python scripts/dump_openapi.py`
  dump (the spec ↔ backend link).
- The frontend `checks` job regenerates and formats the clients from the
  committed specs and fails on any `src/generated/` diff (the TS ↔ spec link):

```bash
pnpm --filter @sep/api codegen
pnpm --filter @sep/api exec oxfmt --write src/generated
git diff --exit-code -- packages/api/src/generated/
```

## How to add a new typed hook

1. Regenerate types if you're calling a newly-added endpoint.
2. Pick the client matching the spec: `mainApi` or `sepApi`.
3. Wrap the call in `throwOnApiError` so errors propagate as `ApiError`:

```ts
import { useQuery } from '@tanstack/react-query';
import { sepApi, throwOnApiError, type SepComponents } from '@sep/api';

type Task = SepComponents['schemas']['BaseTaskResponse'];

export function useChecksumTask(name: string) {
  return useQuery<Task>({
    queryKey: ['checksums', name],
    queryFn: () =>
      throwOnApiError(
        sepApi.GET('/api/apps/checksums/{task_name}', {
          params: { path: { task_name: name } },
        })
      ),
  });
}
```

See `src/hooks/useCurrentUser.ts` for a minimal reference.

## Testing

Tests run under vitest with MSW:

```bash
pnpm --filter @sep/api test
```

Covered:

- Bearer token attachment (present and absent)
- `ApiError` normalization for 4xx / 5xx / network errors
- Unauthorized handler invocation on 401 (skipped for refresh endpoint)
- Retry predicate: skip 4xx, retry 5xx and network, cap at 3 attempts
- `retryDelay` exponential back-off and ceiling

## Known follow-ups

- `postLogin` / `postRefresh` / `fetchCurrentUser` in `src/auth.ts` are
  still hand-written wrappers kept as-is to avoid perturbing
  `AuthProvider`. They can be replaced with typed calls via `mainApi`
  once the SPA auth flow (SEP-961 + SEP-1029) settles.
