import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vitest/config';
import svgr from 'vite-plugin-svgr';
import path from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [tsconfigPaths({ root: '.' }), react(), svgr()],
  base: '/pmm-ui',
  resolve: {
    preserveSymlinks: true,
    dedupe: [
      'react',
      'react-dom',
      '@mui/material',
      '@mui/utils',
      '@mui/system',
      '@mui/icons-material',
    ],
    alias: {
      react: path.resolve(__dirname, '../../node_modules/react'),
      'react-dom': path.resolve(__dirname, '../../node_modules/react-dom'),
      '@emotion/react': path.resolve(
        __dirname,
        '../../node_modules/@emotion/react'
      ),
      '@emotion/styled': path.resolve(
        __dirname,
        '../../node_modules/@emotion/styled'
      ),
      '@mui/system': path.resolve(__dirname, '../../node_modules/@mui/system'),
      '@mui/utils': path.resolve(__dirname, '../../node_modules/@mui/utils'),
      '@mui/material': path.resolve(
        __dirname,
        '../../node_modules/@mui/material'
      ),
      '@mui/icons-material': path.resolve(
        __dirname,
        '../../node_modules/@mui/icons-material'
      ),
    },
  },
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      '@emotion/react',
      '@emotion/styled',
      'prop-types',
      'react-is',
      '@mui/system',
      '@mui/material',
    ],
    exclude: ['@percona/percona-ui'],
    esbuildOptions: {
      // Handle CommonJS modules properly
      mainFields: ['module', 'main'],
    },
    force: true, // Force re-optimization on startup
  },
  server: {
    watch: {
      // Watch the linked package for changes (negated pattern means "don't ignore")
      ignored: ['!**/node_modules/@percona/percona-ui/**'],
    },
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
