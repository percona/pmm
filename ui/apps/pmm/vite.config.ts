import fs from 'fs';
import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react-swc';
import { loadEnv } from 'vite';
import { defineConfig } from 'vitest/config';
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

// SEP backend. The dev server proxies SEP's single `/sep` mount point to it so
// the migrated SEP plugins get real data, mirroring the shipped topology where
// pmm-server's nginx exposes the side-car under that one location (see
// SEP_BASE_PATH in @sep/api). The prefix is passed through unstripped, because
// SEP serves it itself via `root_path` — so PMM_DEV_SEP_BACKEND_URL has to point
// at a backend configured the same way.
// Interim auth (Option D): if PMM_DEV_SEP_INTERNAL_TOKEN is set, inject it
// server-side as a Bearer token so no secret reaches the browser. Both variables
// are dev-server-only, hence the PMM_DEV_ prefix.
// Replaced by the token-exchange provider (Option B) later — see src/sep/bootstrap.ts.
const sepBackendUrl = env.PMM_DEV_SEP_BACKEND_URL || 'http://localhost:8000';
const sepInternalToken = env.PMM_DEV_SEP_INTERNAL_TOKEN;
const sepProxy = () => ({
  target: sepBackendUrl,
  secure: false,
  changeOrigin: true,
  configure: (proxy: {
    on: (e: string, cb: (...a: unknown[]) => void) => void;
  }) => {
    if (!sepInternalToken) {
      return;
    }
    proxy.on('proxyReq', (proxyReq: unknown) => {
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
      '/sep': sepProxy(),
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
