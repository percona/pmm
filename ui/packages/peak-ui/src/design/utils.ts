import baseThemeOptions from './themes/base/BaseTheme';
import pmmThemeOptions from './themes/pmm/PmmTheme';
import sepThemeOptions from './themes/sep/SepTheme';

export const getThemeOptions = (theme: string) => {
  switch (theme) {
    case 'base':
      return baseThemeOptions;
    case 'pmm':
      return pmmThemeOptions;
    case 'sep':
      return sepThemeOptions;
    default:
      return baseThemeOptions;
  }
};
