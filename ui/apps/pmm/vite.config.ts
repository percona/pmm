import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vitest/config';
import svgr from 'vite-plugin-svgr'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [tsconfigPaths({ root: '.' }), react(), svgr()],
  base: '/pmm-ui',
  server: {
    proxy: {
      '/v1': {
        target: '/',
      },
    },
    host: '0.0.0.0',
    strictPort: true,
    hmr: {
      clientPort: 5173,
    },
    allowedHosts: ['host.docker.internal'],
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
