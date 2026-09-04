import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { BlockingTransaction } from 'types/rta.types';
import BlockedByPanel from './BlockedByPanel';

// Modelled on a real pile-up: 409 sits idle inside an open transaction and heads the
// chain, 412 is queued in the middle of it and is waiting itself.
const IDLE_ROOT: BlockingTransaction = {
  blockingConnId: '409',
  blockingQuery: 'SELECT id,k FROM sbtest1 WHERE id=1 FOR UPDATE',
  blockingCommand: 'Sleep',
  blockingUsername: 'sbtest@172.17.0.1',
  waitDuration: '134s',
  blockerTransactionDuration: '154s',
  root: true,
};

const MIDDLE_OF_CHAIN: BlockingTransaction = {
  blockingConnId: '412',
  blockingQuery: 'UPDATE sbtest1 SET k=k+1 WHERE id=1',
  blockingCommand: 'Query',
  blockingUsername: 'sbtest@172.17.0.1',
  waitDuration: '120s',
  blockerTransactionDuration: '121s',
  root: false,
};

const renderPanel = (
  blockers: BlockingTransaction[],
  lockedTable = 'sbtest.sbtest1',
  lockedIndex = 'PRIMARY'
) =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <BlockedByPanel
        blockers={blockers}
        lockedTable={lockedTable}
        lockedIndex={lockedIndex}
      />
    </ThemeProvider>
  );

describe('BlockedByPanel', () => {
  it('names the blocking connection and how long the wait has run', () => {
    renderPanel([IDLE_ROOT]);

    expect(screen.getByTestId('blocked-by-heading')).toHaveTextContent(
      'Blocked by 409'
    );
    expect(screen.getByTestId('blocked-wait-duration')).toHaveTextContent(
      'waiting 2m 14s'
    );
  });

  it('explains that an idle blocker is not running the statement shown', () => {
    renderPanel([IDLE_ROOT]);

    expect(
      screen.getByText(/it is holding a transaction open/)
    ).toBeInTheDocument();
    expect(
      screen.getByText('Sleep · idle in transaction 2m 34s')
    ).toBeInTheDocument();
  });

  it('omits the idle explanation for a blocker that is actually executing', () => {
    renderPanel([MIDDLE_OF_CHAIN]);

    expect(
      screen.queryByText(/it is holding a transaction open/)
    ).not.toBeInTheDocument();
    expect(screen.getByText('Query')).toBeInTheDocument();
  });

  it('leads with the head of the chain rather than the nearest blocker', () => {
    // Deliberately ordered with the non-root blocker first.
    renderPanel([MIDDLE_OF_CHAIN, IDLE_ROOT]);

    expect(screen.getByTestId('blocked-by-heading')).toHaveTextContent(
      'Blocked by 409'
    );
    expect(screen.getByText('1 more transaction ahead')).toBeInTheDocument();
    expect(screen.getByText('412')).toBeInTheDocument();
  });

  it('names nobody in a lock cycle with several blockers', () => {
    // Every participant is waiting, so no single transaction can be pointed at.
    renderPanel([
      { ...MIDDLE_OF_CHAIN, root: false },
      { ...IDLE_ROOT, root: false },
    ]);

    expect(screen.getByTestId('blocked-by-heading')).toHaveTextContent(
      'Blocked by 2 transactions'
    );
  });

  it('declines to name a lone blocker that is itself waiting', () => {
    // root=false means the agent saw that connection waiting too, so resolving it is not
    // guaranteed to free this statement. The pane must agree with the chip and the CSV,
    // which both decline here.
    renderPanel([{ ...IDLE_ROOT, root: false }]);

    expect(screen.getByTestId('blocked-by-heading')).toHaveTextContent(
      'Blocked by 1 transaction'
    );
    expect(
      screen.getByText(/Every transaction involved is itself waiting/)
    ).toBeInTheDocument();
  });

  it('refuses to name one culprit when several transactions are responsible', () => {
    renderPanel([
      { ...MIDDLE_OF_CHAIN, root: true },
      { ...IDLE_ROOT, root: true },
    ]);

    expect(screen.getByTestId('blocked-by-heading')).toHaveTextContent(
      'Blocked by 2 transactions'
    );
    expect(
      screen.getByText(
        /2 transactions are holding this statement up independently/
      )
    ).toBeInTheDocument();
  });

  it('renders the unknown-holder state when no blocker came with the snapshot', () => {
    renderPanel([]);

    expect(screen.getByTestId('blocked-by-panel')).toBeInTheDocument();
    expect(screen.getByText(/was not in this snapshot/)).toBeInTheDocument();
  });

  it('omits the wait duration rather than rendering a dangling label', () => {
    renderPanel([{ ...IDLE_ROOT, waitDuration: null }]);

    expect(
      screen.queryByTestId('blocked-wait-duration')
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/^waiting\s*$/)).not.toBeInTheDocument();
  });

  it('shows the contended table and index', () => {
    renderPanel([IDLE_ROOT]);

    expect(screen.getByText('sbtest.sbtest1')).toBeInTheDocument();
    expect(screen.getByText('PRIMARY')).toBeInTheDocument();
  });
});
