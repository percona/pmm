import { createTheme } from '@mui/material';
import { logsTheme, severityColor } from './logsTheme';

describe('logsTheme', () => {
  it('puts the dark Logs page on black', () => {
    const theme = logsTheme(createTheme({ palette: { mode: 'dark' } }));
    expect(theme.palette.background.default).toBe('#000');
    expect(theme.palette.background.paper).toBe('#0d0d0f');
  });

  it('keeps the light theme as it is', () => {
    const light = createTheme();
    expect(logsTheme(light)).toBe(light);
  });
});

describe('severityColor', () => {
  it('uses lighter severity colours on black than on white', () => {
    expect(severityColor('error', 'dark')).toBe('#f87171');
    expect(severityColor('error', 'light')).toBe('#dc2626');
  });
});
