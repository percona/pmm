import baseThemeOptions from "./themes/base/BaseTheme";
import pmmThemeOptions from "./themes/pmm/PmmTheme";

export const getThemeOptions = (theme: string) => {
  switch (theme) {
    case "base":
      return baseThemeOptions;
    case "pmm":
      return pmmThemeOptions;
    default:
      return baseThemeOptions;
  }
};
