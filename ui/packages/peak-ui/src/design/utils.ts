import baseThemeOptions from "./themes/base/BaseTheme";


export const getThemeOptions = (theme: string) => {
  switch (theme) {
    case "base":
      return baseThemeOptions;
    default:
      return baseThemeOptions;
  }
};
