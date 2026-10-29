import { useMemo, type ReactNode } from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { getThemeOptions } from '../design';

interface ThemedModePanelProps {
  /** Color mode of the panel, independent of the Storybook toolbar. */
  mode: 'light' | 'dark';
  /** Peak theme to render under, such as `base`, `pmm` or `sep`. */
  themeName: string;
  /** Overline above the content; defaults to "Light mode" or "Dark mode". */
  label?: ReactNode;
  /** Content rendered under the panel's theme and mode. */
  children: ReactNode;
}

/** Side-by-side story panel: renders its children under one Peak theme in a fixed color mode. */
const ThemedModePanel = ({ mode, themeName, label, children }: ThemedModePanelProps) => {
  const theme = useMemo(() => createTheme(getThemeOptions(themeName)(mode)), [mode, themeName]);

  return (
    <ThemeProvider theme={theme}>
      <Box
        sx={{
          flex: 1,
          p: 2,
          bgcolor: 'background.paper',
          color: 'text.primary',
          border: 1,
          borderColor: 'divider',
          borderRadius: 1,
        }}
      >
        <Typography variant="overline" component="p">
          {label ?? `${mode === 'light' ? 'Light' : 'Dark'} mode`}
        </Typography>
        {children}
      </Box>
    </ThemeProvider>
  );
};

export default ThemedModePanel;
