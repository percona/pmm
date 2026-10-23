import Button from '@mui/material/Button';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { createTheme, ThemeProvider, useTheme } from '@mui/material/styles';
import { baseThemeOptions, pmmThemeOptions } from '@percona/peak-ui';
import { render, screen } from '@testing-library/react';
import type { PaletteMode } from '@mui/material';
import type { ReactNode } from 'react';
import { ThemeScope } from './ThemeScope';

const Primary = ({ id }: { id: string }) => {
  const { palette } = useTheme();

  return (
    <span data-testid={id} data-mode={palette.mode}>
      {palette.primary.main}
    </span>
  );
};

const inPmm = (mode: PaletteMode, children: ReactNode) =>
  render(
    <ThemeProvider theme={createTheme(pmmThemeOptions(mode))}>
      {children}
    </ThemeProvider>
  );

describe('ThemeScope', () => {
  it.each<PaletteMode>(['light', 'dark'])(
    'gives its area the Base theme in the page’s %s mode, and leaves the rest of the page alone',
    (mode) => {
      inPmm(
        mode,
        <>
          <Primary id="page" />
          <ThemeScope theme="base">
            <Primary id="scoped" />
          </ThemeScope>
        </>
      );

      const base = createTheme(baseThemeOptions(mode)).palette.primary.main;
      const pmm = createTheme(pmmThemeOptions(mode)).palette.primary.main;

      expect(base).not.toBe(pmm);
      expect(screen.getByTestId('scoped')).toHaveTextContent(base);
      expect(screen.getByTestId('scoped')).toHaveAttribute('data-mode', mode);
      expect(screen.getByTestId('page')).toHaveTextContent(pmm);
    }
  );

  it.each<[PaletteMode, PaletteMode]>([
    ['light', 'dark'],
    ['dark', 'light'],
  ])('draws an inverse area of a %s page in %s mode', (page, inverse) => {
    inPmm(
      page,
      <ThemeScope theme="pmm" mode="inverse">
        <Primary id="scoped" />
      </ThemeScope>
    );

    expect(screen.getByTestId('scoped')).toHaveAttribute('data-mode', inverse);
  });

  it('reaches what its area opens in a portal, such as a menu', () => {
    const MenuProbe = () => {
      const { palette } = useTheme();

      return <MenuItem>{palette.primary.main}</MenuItem>;
    };

    inPmm(
      'dark',
      <ThemeScope theme="base">
        <Button>Add MongoDB</Button>
        <Menu open anchorEl={document.body}>
          <MenuProbe />
        </Menu>
      </ThemeScope>
    );

    expect(screen.getByRole('menuitem')).toHaveTextContent(
      createTheme(baseThemeOptions('dark')).palette.primary.main
    );
  });
});
