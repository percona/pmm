import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { BlockingTransaction } from 'types/rta.types';
import BlockedChip from './BlockedChip';

const blocker = (connId: string, root: boolean): BlockingTransaction => ({
  blockingConnId: connId,
  blockingQuery: 'SELECT 1 FOR UPDATE',
  blockingCommand: 'Sleep',
  blockingUsername: 'u@h',
  root,
});

const renderChip = (blockers: BlockingTransaction[]) =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <BlockedChip blockers={blockers} />
    </ThemeProvider>
  );

describe('BlockedChip', () => {
  it('names the connection when one transaction is responsible', () => {
    renderChip([blocker('409', true)]);

    expect(screen.getByTestId('blocked-chip')).toHaveTextContent(
      'Blocked by 409'
    );
  });

  it('counts transactions rather than printing a bare number', () => {
    // "Blocked by 2" would read as connection id 2.
    renderChip([blocker('409', true), blocker('410', true)]);

    expect(screen.getByTestId('blocked-chip')).toHaveTextContent(
      'Blocked by 2 transactions'
    );
  });

  it('uses the singular for one unnameable blocker', () => {
    renderChip([blocker('409', false)]);

    expect(screen.getByTestId('blocked-chip')).toHaveTextContent(
      'Blocked by 1 transaction'
    );
  });

  it('never claims the lock is held by zero transactions', () => {
    renderChip([]);

    const chip = screen.getByTestId('blocked-chip');
    expect(chip).toHaveTextContent('Blocked');
    expect(chip).not.toHaveTextContent('0');
  });
});
