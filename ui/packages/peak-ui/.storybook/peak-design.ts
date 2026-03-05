import { create } from "storybook/internal/theming/create";

export default create({
  base: "light",

  brandTitle: "Percona's Peak Design Storybook",
  brandUrl: "./",
  brandImage: "./logo-peak_design.svg",
  brandTarget: "_self",

  fontBase: '"Roboto", -apple-system, BlinkMacSystemFont, "Segoe UI", "Helvetica Neue", Arial, sans-serif',
  fontCode: '"Roboto Mono", "SFMono-Regular", Consolas, "Liberation Mono", Menlo, Courier, monospace',
  textColor: '#2C323E',
  textInverseColor: '#FBFBFB',

  colorPrimary: "#2CBEA2",
  colorSecondary: "#029CFD",

  appBg: '#ffffff',
  appContentBg: '#ffffff',
  appBorderColor: 'rgba(38, 85, 115, 0.2)',
  appBorderRadius: 5,

  barTextColor: 'rgba(44, 50, 62, 0.7)',
  barSelectedColor: '#029CFD',
  barHoverColor: '#2C323E',
  barBg: '#ffffff',

  inputBg: '#ffffff',
  inputBorder: 'rgba(44, 50, 62, 0.25)',
  inputTextColor: '#2C323E',
  inputBorderRadius: 3,
});