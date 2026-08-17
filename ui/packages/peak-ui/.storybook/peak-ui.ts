import { create } from "storybook/theming/create";
import { CHROME } from "./chrome-tokens";

export default create({
  base: "light",

  brandTitle: "Percona's Peak UI Storybook",
  brandUrl: "./",
  brandImage: "./logo-peak-ui.svg",
  brandTarget: "_self",

  fontBase: '"Roboto", -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif',
  fontCode: '"Roboto Mono", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace',
  textColor: CHROME.textPrimary,
  textInverseColor: CHROME.textPrimaryDark,

  colorPrimary: CHROME.brand,
  colorSecondary: CHROME.brand,

  appBg: CHROME.surfaceElevation1,
  appContentBg: CHROME.surfaceElevation1,
  appBorderColor: CHROME.contour,
  appBorderRadius: CHROME.radiusSm,

  barTextColor: CHROME.textSecondary,
  barSelectedColor: CHROME.brand,
  barHoverColor: CHROME.textPrimary,
  barBg: CHROME.surfaceElevation1,

  inputBg: CHROME.surfaceElevation1,
  inputBorder: CHROME.divider,
  inputTextColor: CHROME.textPrimary,
  inputBorderRadius: CHROME.radiusXs,
});
