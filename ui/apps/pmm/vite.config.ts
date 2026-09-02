import fs from 'fs';
import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';
import svgr from 'vite-plugin-svgr';
import basicSsl from '@vitejs/plugin-basic-ssl';

const CERT_KEY = '/srv/nginx/certificate.key';
const CERT_CRT = '/srv/nginx/certificate.crt';
const hasNginxCerts = fs.existsSync(CERT_KEY) && fs.existsSync(CERT_CRT);
const port = hasNginxCerts ? 5173 : 5174;
const target = hasNginxCerts ? 'https://localhost:8443' : 'https://localhost';
// pnpm link'd packages don't bump the lockfile hash, so Vite's dep cache can go stale for them.

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
