import type { StorybookConfig } from "@storybook/react-vite";
import tsconfigPaths from "vite-tsconfig-paths";
import { MATURITY_TAGS } from "./maturity-tags";

const cjsInteropDeps = [
  "material-react-table",
  "react-is",
  "prop-types",
  "hoist-non-react-statics",
] as const;

const config: StorybookConfig = {
  stories: ["../src/**/*.mdx", "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"],
  staticDirs: ["./static"],
  addons: [
    "@storybook/addon-docs",
    "storybook-addon-pseudo-states",
    "storybook-addon-tag-badges",
  ],
  tags: Object.fromEntries(MATURITY_TAGS.map(({ id }) => [id, {}])),
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  features: {
    sidebarOnboardingChecklist: false,
  },
  viteFinal: async (config) => {
    const path = await import("node:path");
    const { mergeConfig } = await import("vite");
    const root = process.cwd();

    config.optimizeDeps ??= {};
    config.optimizeDeps.include = [
      ...new Set([...(config.optimizeDeps.include ?? []), ...cjsInteropDeps]),
    ];

    return mergeConfig(config, {
      plugins: [tsconfigPaths()],
      // Keep fn/class names through minification so Storybook shows real component names.
      esbuild: { keepNames: true },
      optimizeDeps: { esbuildOptions: { keepNames: true } },
      build: {
        minify: "terser",
        terserOptions: {
          keep_classnames: true,
          keep_fnames: true,
        },
      },
      resolve: {
        dedupe: ["react", "react-dom", "react-is", "prop-types"],
        alias: {
          "material-react-table": path.resolve(
            root,
            "node_modules/material-react-table/dist/index.esm.js",
          ),
          "react-is": path.resolve(root, "node_modules/react-is"),
          "prop-types": path.resolve(root, "node_modules/prop-types"),
        },
      },
    });
  },
};

export default config;
