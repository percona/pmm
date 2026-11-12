import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    include: ['src/**/*.spec.ts', 'src/**/*.spec.tsx', '.storybook/**/*.spec.ts', '.storybook/**/*.spec.tsx'],
  },
});
