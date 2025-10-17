import tsconfigPaths from 'vite-tsconfig-paths';
import react from '@vitejs/plugin-react-swc';
import svgr from 'vite-plugin-svgr';
import { defineConfig, loadEnv } from 'vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const target = env.VITE_API_URL || 'http://localhost';

  return {
    plugins: [
      tsconfigPaths({ root: '.' }),
      svgr({
        svgrOptions: { exportType: 'default' },
      }),
      react(),
    ],
    base: '/pmm-ui',
    server: {
      host: '0.0.0.0',
      strictPort: true,
      hmr: { clientPort: 5173 },
      allowedHosts: ['host.docker.internal', 'localhost', '127.0.0.1'],
      proxy: {
        '/graph': {
          target,
          changeOrigin: true,
          secure: false,
          ws: true,
          // make Grafana cookies valid for localhost:5173
          cookieDomainRewrite: '', // <— add
          cookiePathRewrite: '/', // <— add
          // drop Secure on HTTP in dev (optional but useful)
          configure: (proxy) => {
            proxy.on('proxyRes', (proxyRes) => {
              const setCookie = proxyRes.headers['set-cookie'];
              if (Array.isArray(setCookie)) {
                proxyRes.headers['set-cookie'] = setCookie.map(
                  (c) =>
                    c
                      .replace(/;\s*Path=\/graph/gi, '; Path=/') // <— force /
                      .replace(/;\s*Secure/gi, '') // <— drop Secure in dev
                );
              }
            });
          },
        },

        '/qan-api': { target, changeOrigin: true, secure: false },
        '/inventory-api': { target, changeOrigin: true, secure: false },
        '/prometheus': { target, changeOrigin: true, secure: false },
        '/vmalert': { target, changeOrigin: true, secure: false },
        '/alertmanager': { target, changeOrigin: true, secure: false },
        '/v1': {
          target,
          changeOrigin: true,
          secure: false,
          cookieDomainRewrite: '',
        },
      },
    },
  };
});
