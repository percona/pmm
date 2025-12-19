import baseThemeOptions from "./themes/base/BaseTheme";
import everestThemeOptions from "./themes/everest/EverestTheme";

export const getThemeOptions = (theme: string) => {
  switch (theme) {
    case "base":
      return baseThemeOptions;
    case "everest":
      return everestThemeOptions;
    default:
      return baseThemeOptions;
  }
};
