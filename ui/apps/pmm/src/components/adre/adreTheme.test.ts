import { Theme, createTheme } from '@mui/material';
import { adreColors } from './adreAnswerSx';
import { adreTheme } from './adreTheme';

describe('adreTheme', () => {
  it('turns the surrounding theme dark with the Assistant colours', () => {
    const theme = adreTheme(createTheme({ palette: { mode: 'light' } }));
    expect(theme.palette.mode).toBe('dark');
    expect(theme.palette.background.paper).toBe(adreColors.panel);
    expect(theme.palette.primary.main).toBe(adreColors.accent);
  });

  it('works without a surrounding theme', () => {
    const theme = adreTheme({} as Theme);
    expect(theme.palette.mode).toBe('dark');
    expect(theme.palette.text.primary).toBe(adreColors.text);
  });
});
