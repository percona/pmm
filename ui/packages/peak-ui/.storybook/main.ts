import type { StorybookConfig } from "@storybook/react-vite";
import tsconfigPaths from "vite-tsconfig-paths";

const cjsInteropDeps = [
  "material-react-table",
  "react-is",
  "prop-types",
  "hoist-non-react-statics",
] as const;

const config: StorybookConfig = {
  stories: ["../src/**/*.mdx", "../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"],
  staticDirs: ["./static"],
  addons: ["@storybook/addon-docs", "storybook-addon-pseudo-states"],
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
