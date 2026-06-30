export type MaturityStatus = "stable" | "experimental" | "needs-review" | "deprecated";

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
    description:
      "Production-ready and reviewed. Safe to use.",
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
    description:
      "On the way out. Avoid in new code.",
    colors: { bg: "#B42B21", fg: "#FFFFFF", border: "rgba(0, 0, 0, 0.05)" },
  },
];

export const badgeStyle = (colors: MaturityTagColors, fontSize: number) => ({
  display: "inline-block",
  whiteSpace: "nowrap" as const,
  backgroundColor: colors.bg,
  color: colors.fg,
  border: `0.0625em solid ${colors.border}`,
  fontSize,
  fontWeight: 600,
  textTransform: "uppercase" as const,
  padding: "0.125em 0.375em",
  borderRadius: "999px",
  letterSpacing: "0.0625em",
  lineHeight: 1,
  boxShadow: "none",
});
