import { addons } from "storybook/manager-api";
import type { TagBadgeParameters } from "storybook-addon-tag-badges/manager-helpers";
import peakDesign from "./peak-design";
import { MATURITY_TAGS, badgeStyle, type BadgeContext } from "./maturity-tags";

const display = {
  sidebar: [
    { type: "component", skipInherited: true } as const,
    { type: "group", skipInherited: false } as const,
  ],
  toolbar: true,
  mdx: true,
};

const tagBadges: TagBadgeParameters = MATURITY_TAGS.map(({ id, label, description, colors }) => ({
  tags: id,
  badge: ({ context }: { context: BadgeContext }) => ({
    text: label,
    style: badgeStyle(colors, context),
    tooltip: description,
  }),
  display,
}));

addons.setConfig({
  theme: peakDesign,
  tagBadges,
});
