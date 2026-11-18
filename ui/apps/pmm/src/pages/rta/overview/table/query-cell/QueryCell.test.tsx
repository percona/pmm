import { render, screen } from '@testing-library/react';
import { afterEach, describe, it, expect, vi } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import QueryCell from './QueryCell';

const mockWidths = (scrollWidth: number, clientWidth: number) => {
  vi.spyOn(HTMLElement.prototype, 'scrollWidth', 'get').mockReturnValue(
    scrollWidth
  );
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(
    clientWidth
  );
};

const renderCell = (query: string) =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <QueryCell query={query} language="sql" />
    </ThemeProvider>
  );

describe('QueryCell', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('does not fade a statement that fits', () => {
    // A column sized to its content leaves no slack; fading anyway hid "T" of "COMMIT".
    mockWidths(80, 80);
    const { container } = renderCell('COMMIT');

    expect(screen.getByText('COMMIT')).toBeInTheDocument();
    expect(container.querySelector('pre')).toHaveAttribute(
      'data-overflowing',
      'false'
    );
  });

  it('fades a statement that is clipped', () => {
    mockWidths(900, 300);
    const { container } = renderCell('SELECT * FROM a_very_long_table_name');

    expect(container.querySelector('pre')).toHaveAttribute(
      'data-overflowing',
      'true'
    );
  });
});
