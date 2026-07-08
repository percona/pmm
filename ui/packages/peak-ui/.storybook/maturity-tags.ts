// Single source of truth for component maturity tags: their ids, labels,
// descriptions, colors, and badge styling. Consumed by main.ts (filter
// registration), manager.ts (sidebar/toolbar badges), and Introduction.mdx
// (the homepage legend). Add or edit a status here and every surface follows.

export type MaturityStatus = "stable" | "experimental" | "needs-review" | "deprecated";

export type BadgeContext = "sidebar" | "toolbar" | "mdx";

export interface MaturityTagColors {
  bg: string;
  fg: string;
  border: string;
}

export interface MaturityTag {
  id: MaturityStatus;
  label: string;
  description: string;
  colors: MaturityTagColors;
}

export const MATURITY_TAGS: MaturityTag[] = [
  {
    id: "stable",
    label: "Stable",
    description: "Production-ready and reviewed. Safe to use.",
    colors: { bg: "#F1FCF2", fg: "#175025", border: "rgba(0, 0, 0, 0.05)" },
  },
  {
    id: "experimental",
    label: "Beta",
    description:
      "Intentionally early. The definitions and API may change without notice. Expect churn.",
    colors: { bg: "#F1F9FE", fg: "#144866", border: "rgba(0, 0, 0, 0.05)" },
  },
  {
    id: "needs-review",
    label: "To review",
    description:
      "Functional but not yet vetted against the whole design system. Usable, but may change.",
    colors: { bg: "#FFFBE0", fg: "#7D5A08", border: "rgba(0, 0, 0, 0.05)" },
  },
  {
    id: "deprecated",
    label: "Deprecated",
    description: "On the way out. Avoid in new code.",
    colors: { bg: "#B42B21", fg: "#FFFFFF", border: "rgba(0, 0, 0, 0.05)" },
  },
];

// Badges shrink in the dense sidebar and read full-size in docs/toolbars.
// Padding, border, and radius use `em`, so they scale with these sizes.
const FONT_SIZE: Record<BadgeContext, number> = { sidebar: 8, toolbar: 12, mdx: 12 };

export const badgeStyle = (colors: MaturityTagColors, context: BadgeContext) => ({
  display: "inline-block",
  whiteSpace: "nowrap" as const,
  backgroundColor: colors.bg,
  color: colors.fg,
  border: `0.0625em solid ${colors.border}`,
  fontSize: FONT_SIZE[context],
  fontWeight: 600,
  textTransform: "uppercase" as const,
  padding: "0.125em 0.375em",
  borderRadius: "999px",
  letterSpacing: "0.0625em",
  lineHeight: 1,
  boxShadow: "none",
});
