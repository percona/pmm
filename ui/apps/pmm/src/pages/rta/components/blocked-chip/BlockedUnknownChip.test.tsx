import { fireEvent, render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import BlockedUnknownChip from './BlockedUnknownChip';

describe('BlockedUnknownChip', () => {
  it('says blocking is unknown rather than blocked or not blocked', () => {
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <BlockedUnknownChip />
      </ThemeProvider>
    );

    expect(screen.getByTestId('blocked-unknown-chip')).toHaveTextContent(
      'Blocked: unknown'
    );
  });

  it('explains that the connection moved on between the two reads', async () => {
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <BlockedUnknownChip />
      </ThemeProvider>
    );

    fireEvent.mouseOver(screen.getByTestId('blocked-unknown-chip'));

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      /moved on from this statement to a later one/
    );
  });
});
