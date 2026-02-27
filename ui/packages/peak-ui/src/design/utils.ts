import baseThemeOptions from "./themes/base/BaseTheme";
import pmmThemeOptions from "./themes/pmm/PmmTheme";

export const getThemeOptions = (theme: string) => {
  switch (theme) {
    case "pmm":
      return pmmThemeOptions;
    case "base":
    default:
      return baseThemeOptions;
  }
};
