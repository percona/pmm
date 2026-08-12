import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react-swc';
import { defineConfig } from 'vitest/config';
import svgr from 'vite-plugin-svgr';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [tsconfigPaths({ root: '.' }), react(), svgr()],
  base: '/pmm-ui',
  resolve: {
    // Collapse to a single instance of each. Under pnpm, react-dom and app code
    // otherwise resolve `react` to different symlink paths, giving vite-node two
    // React module records → "Invalid hook call". `dedupe` funnels every specifier
    // to one resolved copy; no hardcoded node_modules paths (which broke under pnpm).
    dedupe: [
      'react',
      'react-dom',
      'react/jsx-runtime',
      'react/jsx-dev-runtime',
      '@emotion/react',
      '@emotion/styled',
    ],
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: 'src/setupTests.ts',
    server: {
      deps: {
        fallbackCJS: true,
        // Route these React consumers through Vite's transform so their `react`
        // imports go through `dedupe` and share the app's single React instance.
        inline: ['@percona/peak-ui', 'react-dom', '@testing-library/react'],
      },
    },
  },
});
