import { fireEvent, render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import BlockedUnknownChip from './BlockedUnknownChip';

describe('BlockedUnknownChip', () => {
  it('says blocking is unknown rather than blocked or not blocked', () => {
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <BlockedUnknownChip reason="unattributed" />
      </ThemeProvider>
    );

    expect(screen.getByTestId('blocked-unknown-chip')).toHaveTextContent(
      'Possibly blocked'
    );
  });

  it('explains that the connection moved on between the two reads', async () => {
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <BlockedUnknownChip reason="unattributed" />
      </ThemeProvider>
    );

    fireEvent.mouseOver(screen.getByTestId('blocked-unknown-chip'));

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      /moved on to a later statement before the lock check/
    );
  });

  it('explains that the lock type cannot be read on this instance', async () => {
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <BlockedUnknownChip reason="unreadable" />
      </ThemeProvider>
    );

    const chip = screen.getByTestId('blocked-unknown-chip');
    expect(chip).toHaveTextContent('Blocked by unknown');
    expect(chip).toHaveAttribute('data-reason', 'unreadable');

    fireEvent.mouseOver(chip);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      /doesn't report who holds this lock type.*wait\/lock\/metadata\/sql\/mdl/
    );
  });
});
