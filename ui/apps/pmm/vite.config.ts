import fs from 'fs';
import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react';
import { defineConfig, loadEnv } from 'vite';
import svgr from 'vite-plugin-svgr';
import basicSsl from '@vitejs/plugin-basic-ssl';

// Vite exposes `.env` files to client code as `import.meta.env`, but never to
// this config file's `process.env` — so the dev-server settings below would
// otherwise have to be exported in whichever shell launches the dev server, and
// silently fall back to their defaults when they are not. Load the files here so
// a gitignored `apps/pmm/.env.local` works. A real environment variable still
// wins, which is what CI and the devcontainer rely on.
// Mode is pinned to 'development': every value read here configures the dev
// server only, and `.env.local` is loaded for every mode regardless.
const env = {
  ...loadEnv('development', import.meta.dirname, ''),
  ...process.env,
};

const CERT_KEY = '/srv/nginx/certificate.key';
const CERT_CRT = '/srv/nginx/certificate.crt';
const hasNginxCerts = fs.existsSync(CERT_KEY) && fs.existsSync(CERT_CRT);
const port = hasNginxCerts ? 5173 : 5174;
const target =
  env.PMM_SERVER_URL ||
  (hasNginxCerts ? 'https://localhost:8443' : 'https://localhost');
// pnpm link'd packages don't bump the lockfile hash, so Vite's dep cache can go stale for them.

// SEP backend. The dev server proxies SEP's single `/sep` mount point to it so
// the migrated SEP plugins get real data, mirroring the shipped topology where
// pmm-server's nginx exposes the side-car under that one location (see
// SEP_BASE_PATH in @sep/api). The prefix is forwarded unstripped by default, for
// a SEP serving it itself — so PMM_DEV_SEP_BACKEND_URL has to point at a backend
// started with `SEP__ROOT_PATH=/sep`, or PMM_DEV_SEP_STRIP_PREFIX has to be set
// (below).
//
// Residual interim auth: if PMM_DEV_SEP_INTERNAL_TOKEN is set, inject it
// server-side as a Bearer token so no secret reaches the browser. Both variables
// are dev-server-only, hence the PMM_DEV_ prefix.
//
// The browser now mints its own bearer by exchanging the PMM session (see
// src/sep/bootstrap.ts), so the injection is only a fallback for a SEP instance
// whose Grafana provider is not wired up yet. It must never cover the OAuth
// routes: overwriting Authorization there would authenticate the exchange as
// SEP's internal service principal and mask whether the cookie path works at
// all. Retiring the injection entirely is a follow-up.
// Mirrors SEP_BASE_PATH in @sep/api. Declared locally rather than imported:
// this config is evaluated by Node before any of the app's module resolution
// applies, and pulling the browser client in for one string is not worth it.
const SEP_BASE_PATH = '/sep';
const sepBackendUrl = env.PMM_DEV_SEP_BACKEND_URL || 'http://localhost:8000';
const sepInternalToken = env.PMM_DEV_SEP_INTERNAL_TOKEN;
// Escape hatch for a SEP that is not serving the `/sep` prefix itself. Setting
// PMM_DEV_SEP_STRIP_PREFIX=1 strips it before forwarding.
//
// The supported setup does not need this. SEP-1794 added a `SEP.ROOT_PATH`
// setting, passed to the `FastAPI(root_path=…)` constructor, so a SEP started
// with `SEP__ROOT_PATH=/sep` serves the prefix and the proxy hands it over
// untouched. Reach for the strip only against a SEP that predates that change
// or runs with `ROOT_PATH` unset.
//
// Do not pair it with uvicorn's `--root-path`. That flag *prepends* root_path
// to the request path (`h11_impl.py`: `full_path = root_path + path`) instead
// of declaring the mount, so SEP sees `/sep/sep/…`; the strip cancels it out,
// but `SEP__ROOT_PATH` is the setting that means what it says.
//
// Off by default: the default belongs to whatever the server-side nginx
// location ends up doing, which does not exist in this repo yet. Stripping is
// also lossy against a SEP with no `ROOT_PATH` — `request.url_for()` then emits
// prefix-less absolute URLs inside JSON payloads, so any link it hands back
// (file downloads in particular) escapes the mount point.
const sepStripPrefix = env.PMM_DEV_SEP_STRIP_PREFIX === '1';
// Matched against the proxied request URL. Vite applies `rewrite` to `req.url`
// before the proxy sees it, so with the strip enabled the prefix is already
// gone by the time this runs — both forms have to match, or the token would be
// injected onto the OAuth routes it must never cover.
const isSepAuthPath = (url: string | undefined) =>
  !!url &&
  (url.startsWith(`${SEP_BASE_PATH}/api/oauth/`) ||
    url.startsWith('/api/oauth/'));
const sepProxy = () => ({
  target: sepBackendUrl,
  secure: false,
  changeOrigin: true,
  ...(sepStripPrefix
    ? { rewrite: (path: string) => path.slice(SEP_BASE_PATH.length) || '/' }
    : {}),
  configure: (proxy: {
    on: (e: string, cb: (...a: unknown[]) => void) => void;
  }) => {
    if (!sepInternalToken) {
      return;
    }
    proxy.on('proxyReq', (proxyReq: unknown, req: unknown) => {
      if (isSepAuthPath((req as { url?: string }).url)) {
        return;
      }
      (proxyReq as { setHeader: (k: string, v: string) => void }).setHeader(
        'Authorization',
        `Bearer ${sepInternalToken}`
      );
    });
  },
});

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    tsconfigPaths({ root: '.' }),
    react(),
    svgr(),
    ...(hasNginxCerts ? [] : [basicSsl()]),
  ],
  base: '/pmm-ui',
  resolve: {
    dedupe: [
      'react',
      'react-dom',
      'react-is',
      '@emotion/react',
      '@emotion/styled',
      '@mui/material',
      '@mui/system',
      '@mui/styled-engine',
      '@mui/utils',
    ],
  },
  optimizeDeps: {
    // Uncomment when using pnpm link for @percona/peak-ui locally
    // exclude: ['@percona/peak-ui'],
    force: true,
  },
  server: {
    https: hasNginxCerts
      ? { key: fs.readFileSync(CERT_KEY), cert: fs.readFileSync(CERT_CRT) }
      : undefined,
    watch: {
      // Watch the linked package for changes (negated pattern means "don't ignore")
      ignored: ['!**/node_modules/@percona/peak-ui/**'],
    },
    proxy: {
      '/v1': {
        target,
        secure: false,
        changeOrigin: true,
      },
      '/graph': {
        target,
        secure: false,
        changeOrigin: true,
      },
      '/logs.zip': {
        target,
        secure: false,
        changeOrigin: true,
      },
      [SEP_BASE_PATH]: sepProxy(),
    },
    host: '0.0.0.0',
    port,
    strictPort: true,
    hmr: {
      protocol: 'wss',
      // Don't force clientPort: in the devcontainer flow the browser loads Vite
      // from the docker-mapped host port (PMM_PORT_VITE), which may differ from
      // the container-internal `port`. Let Vite infer the port from
      // window.location so HMR connects to whatever port served the page.
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: 'src/setupTests.ts',
    server: {
      deps: {
        fallbackCJS: true,
      },
    },
  },
});
