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
    dedupe: [
      'react',
      'react-dom',
      'react/jsx-runtime',
      'react/jsx-dev-runtime',
      '@emotion/react',
      '@emotion/styled',
    ],
    alias: {
      // Force React to resolve from main project's node_modules
      react: path.resolve(__dirname, '../../node_modules/react'),
      'react-dom': path.resolve(__dirname, '../../node_modules/react-dom'),
      'react/jsx-runtime': path.resolve(
        __dirname,
        '../../node_modules/react/jsx-runtime'
      ),
      'react/jsx-dev-runtime': path.resolve(
        __dirname,
        '../../node_modules/react/jsx-dev-runtime'
      ),
      '@emotion/react': path.resolve(
        __dirname,
        '../../node_modules/@emotion/react'
      ),
      '@emotion/styled': path.resolve(
        __dirname,
        '../../node_modules/@emotion/styled'
      ),
    },
  },
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      'react/jsx-runtime',
      'react/jsx-dev-runtime',
    ],
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: 'src/setupTests.ts',
    server: {
      deps: {
        fallbackCJS: true,
        inline: ['@percona/percona-ui'],
      },
    },
  },
});
