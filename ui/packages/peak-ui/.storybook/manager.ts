import { addons } from "storybook/internal/manager-api";
import peakDesign from "./peak-design";

addons.setConfig({
  layoutCustomisations: {
    showSidebar: ({ viewMode }) => viewMode === "docs",
  },
  theme: peakDesign,
});