import { addons } from "storybook/manager-api";
import type { TagBadgeParameters } from "storybook-addon-tag-badges/manager-helpers";
import peakDesign from "./peak-design";

const sidebarOnComponent = [{ type: "component", skipInherited: true } as const];

const chip = (bg: string, fg: string, border: string) => ({
  backgroundColor: bg,
  color: fg,
  borderColor: border,
  fontSize: 10,
  fontWeight: 500,
  padding: "1px 6px",
  borderRadius: "4px",
  letterSpacing: 0,
});

const STATUS = {
  stable: chip("rgba(46, 125, 50, 0.14)", "#2e7d32", "rgba(46, 125, 50, 0.3)"),
  experimental: chip("rgba(106, 63, 209, 0.14)", "#6a3fd1", "rgba(106, 63, 209, 0.3)"),
  "needs-review": chip("rgba(180, 83, 9, 0.14)", "#b45309", "rgba(180, 83, 9, 0.3)"),
  deprecated: chip("rgba(198, 40, 40, 0.14)", "#c62828", "rgba(198, 40, 40, 0.3)"),
};

const display = { sidebar: sidebarOnComponent, toolbar: true, mdx: true };

const tagBadges: TagBadgeParameters = [
  {
    tags: "stable",
    badge: { text: "Stable", style: STATUS.stable, tooltip: "Production-ready. Safe to use." },
    display,
  },
  {
    tags: "experimental",
    badge: {
      text: "Experimental",
      style: STATUS.experimental,
      tooltip: "Experimental. API may change without notice.",
    },
    display,
  },
  {
    tags: "needs-review",
    badge: {
      text: "Needs review",
      style: STATUS["needs-review"],
      tooltip: "Not yet reviewed. Use with caution; may change.",
    },
    display,
  },
  {
    tags: "deprecated",
    badge: {
      text: "Deprecated",
      style: STATUS.deprecated,
      tooltip: "Deprecated. Avoid in new code.",
    },
    display,
  },
];

addons.setConfig({
  theme: peakDesign,
  tagBadges,
});
