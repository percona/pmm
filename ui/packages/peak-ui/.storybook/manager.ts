import { addons } from "storybook/manager-api";
import type { TagBadgeParameters } from "storybook-addon-tag-badges/manager-helpers";
import peakDesign from "./peak-design";
import { MATURITY_TAGS, badgeStyle } from "./maturity-tags";

const sidebarDisplay = [
  { type: "component", skipInherited: true } as const,
  { type: "group", skipInherited: false } as const,
];

const SIDEBAR_FONT_SIZE = 8;
const TOOLBAR_FONT_SIZE = 12;

const display = { sidebar: sidebarDisplay, toolbar: true, mdx: true };

const tagBadges: TagBadgeParameters = MATURITY_TAGS.map(({ id, label, description, colors }) => ({
  tags: id,
  badge: ({ context }: { context: "sidebar" | "toolbar" | "mdx" }) => ({
    text: label,
    style: badgeStyle(colors, context === "toolbar" ? TOOLBAR_FONT_SIZE : SIDEBAR_FONT_SIZE),
    tooltip: description,
  }),
  display,
}));

addons.setConfig({
  theme: peakDesign,
  tagBadges,
});
