import { addons } from "storybook/manager-api";
import type { TagBadgeParameters } from "storybook-addon-tag-badges/manager-helpers";
import peakDesign from "./peak-design";

const sidebarOnComponent = [{ type: "component", skipInherited: true } as const];

const tagBadges: TagBadgeParameters = [
  {
    tags: "stable",
    badge: {
      text: "Stable",
      style: "green",
      tooltip: "Production-ready. Safe to use.",
    },
    display: { sidebar: sidebarOnComponent, toolbar: true, mdx: true },
  },
  {
    tags: "experimental",
    badge: {
      text: "Experimental",
      style: "purple",
      tooltip: "Experimental. API may change without notice.",
    },
    display: { sidebar: sidebarOnComponent, toolbar: true, mdx: true },
  },
  {
    tags: "needs-review",
    badge: {
      text: "Needs review",
      style: "orange",
      tooltip: "Not yet reviewed. Use with caution; may change.",
    },
    display: { sidebar: sidebarOnComponent, toolbar: true, mdx: true },
  },
  {
    tags: "deprecated",
    badge: {
      text: "Deprecated",
      style: "red",
      tooltip: "Deprecated. Avoid in new code.",
    },
    display: { sidebar: sidebarOnComponent, toolbar: true, mdx: true },
  },
];

addons.setConfig({
  theme: peakDesign,
  tagBadges,
});
