import type { StorybookConfig } from "@storybook/react-vite";
import tsconfigPaths from "vite-tsconfig-paths";

const config: StorybookConfig = {
  stories: ["../src/**/*.mdx", "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"],
  staticDirs: ["./static"],
  addons: [
    "@storybook/addon-docs",
    "storybook-addon-pseudo-states",
  ],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  features: {
    sidebarOnboardingChecklist: false,
  },
  viteFinal: async (config) => {
    const { mergeConfig } = await import("vite");
    return mergeConfig(config, {
      plugins: [tsconfigPaths()],
      // Preserve function/class names through minification so Storybook's code
      // panel can resolve real component names (e.g. `IconButton`) instead of
      // `React.ForwardRef` in the production build.
      esbuild: { keepNames: true },
      optimizeDeps: { esbuildOptions: { keepNames: true } },
      build: {
        minify: "terser",
        terserOptions: {
          keep_classnames: true,
          keep_fnames: true,
        },
      },
    });
  },
};
export default config;
