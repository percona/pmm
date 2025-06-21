import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vitest/config';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [tsconfigPaths({ root: '.' }), react()],
  base: '/pmm-ui',
  server: {
    proxy: {
      '/v1/chat': {
        target: 'http://localhost:3001',
        changeOrigin: true,
      },
      '/v1': {
        target: '/',
      },
    },
    host: '0.0.0.0',
    strictPort: true,
    hmr: {
      clientPort: 5173,
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
