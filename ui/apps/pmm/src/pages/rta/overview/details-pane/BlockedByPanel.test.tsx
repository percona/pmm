import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { BlockingTransaction, LockType } from 'types/rta.types';
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
  lockedIndex = 'PRIMARY',
  lockType: LockType | undefined = LockType.row,
  requestedLockMode: string | undefined = 'X,REC_NOT_GAP'
) =>
  render(
    <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
      <BlockedByPanel
        blockers={blockers}
        lockedTable={lockedTable}
        lockedIndex={lockedIndex}
        lockType={lockType}
        requestedLockMode={requestedLockMode}
      />
    </ThemeProvider>
  );

// A metadata-lock pile-up: the transaction holding SHARED_READ heads the chain and the
// DDL that wants EXCLUSIVE is queued behind it, itself blocking everything after.
const MDL_ROOT: BlockingTransaction = {
  blockingConnId: '409',
  blockingQuery: 'SELECT COUNT(*) FROM sbtest2 WHERE id<10',
  blockingCommand: 'Sleep',
  blockingUsername: 'sbtest@172.17.0.1',
  blockerTransactionDuration: '154s',
  root: true,
  blockingLockMode: 'SHARED_READ',
};

const MDL_DDL: BlockingTransaction = {
  blockingConnId: '411',
  blockingQuery: 'ALTER TABLE sbtest2 ADD COLUMN c INT NULL',
  blockingCommand: 'Query',
  blockingUsername: 'sbtest@172.17.0.1',
  blockerTransactionDuration: '20s',
  root: false,
  blockingLockMode: 'SHARED_UPGRADABLE',
};

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

  it('says which locking mechanism the statement is waiting on', () => {
    renderPanel([IDLE_ROOT]);

    expect(screen.getByText('Lock type')).toBeInTheDocument();
    expect(screen.getByText('Row lock (InnoDB)')).toBeInTheDocument();
  });

  it('shows the requested and held lock modes so a gap lock is distinguishable', () => {
    renderPanel([{ ...IDLE_ROOT, blockingLockMode: 'X,GAP' }]);

    expect(screen.getByText('Mode requested')).toBeInTheDocument();
    expect(screen.getByText('X,REC_NOT_GAP')).toBeInTheDocument();
    expect(screen.getByText('Blocker holds')).toBeInTheDocument();
    expect(screen.getByText('X,GAP')).toBeInTheDocument();
  });

  it('labels a metadata lock as one and omits the index it has no meaning for', () => {
    // A metadata lock is taken on the table as a whole, so the agent sends no index.
    renderPanel(
      [MDL_ROOT],
      'sbtest.sbtest2',
      '',
      LockType.metadata,
      'EXCLUSIVE'
    );

    expect(screen.getByText('Metadata lock (MDL)')).toBeInTheDocument();
    expect(screen.getByText('EXCLUSIVE')).toBeInTheDocument();
    expect(screen.queryByText('Locked index')).not.toBeInTheDocument();
  });

  it('words the remedy for a metadata lock rather than for a row lock', () => {
    renderPanel(
      [MDL_ROOT],
      'sbtest.sbtest2',
      '',
      LockType.metadata,
      'EXCLUSIVE'
    );

    expect(
      screen.getByText(/holds a metadata lock on this table/)
    ).toBeInTheDocument();
  });

  it('shows what each queued transaction holds, not just its id', () => {
    // 409 heads the chain and the DDL is queued in front of the waiter: the modes are what
    // make that order legible instead of two interchangeable connection ids.
    renderPanel(
      [MDL_ROOT, MDL_DDL],
      'sbtest.sbtest2',
      '',
      LockType.metadata,
      'SHARED_READ'
    );

    expect(screen.getByText('1 more transaction ahead')).toBeInTheDocument();
    expect(screen.getByText('SHARED_UPGRADABLE')).toBeInTheDocument();
  });

  it('renders no lock type row when the agent could not classify the wait', () => {
    // Rendered without the helper on purpose: its defaults would supply the very values
    // this case is about not having.
    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <BlockedByPanel blockers={[IDLE_ROOT]} lockedTable="sbtest.sbtest1" />
      </ThemeProvider>
    );

    expect(screen.queryByText('Lock type')).not.toBeInTheDocument();
    expect(screen.queryByText('Mode requested')).not.toBeInTheDocument();
    expect(screen.queryByText('Blocker holds')).not.toBeInTheDocument();
  });

  it('does not pair the requested mode with one blocker when several are named', () => {
    // Two independent roots holding different modes: showing one beside "Mode requested"
    // reads as a matched pair and hides that the other holds something else.
    renderPanel(
      [MDL_ROOT, { ...MDL_DDL, root: true, blockingLockMode: 'SHARED_WRITE' }],
      'sbtest.sbtest2',
      '',
      LockType.metadata,
      'EXCLUSIVE'
    );

    expect(screen.getByText('Mode requested')).toBeInTheDocument();
    expect(screen.queryByText('Blocker holds')).not.toBeInTheDocument();
    expect(screen.getByText('SHARED_WRITE')).toBeInTheDocument();
  });

  it('does show the held mode when exactly one transaction is named', () => {
    renderPanel(
      [MDL_ROOT],
      'sbtest.sbtest2',
      '',
      LockType.metadata,
      'EXCLUSIVE'
    );

    expect(screen.getByText('Blocker holds')).toBeInTheDocument();
    expect(screen.getByText('SHARED_READ')).toBeInTheDocument();
  });

  it('does not call an independent holder a transaction queued ahead', () => {
    // "N more transaction ahead" contradicted the hint below, which counts the same blocker
    // among the transactions the statement stays blocked on until every one is resolved.
    renderPanel(
      [MDL_ROOT, { ...MDL_DDL, root: true, blockingLockMode: 'SHARED_WRITE' }],
      'sbtest.sbtest2',
      '',
      LockType.metadata,
      'EXCLUSIVE'
    );

    expect(
      screen.getByText('1 more transaction holding it independently')
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/more transaction ahead/)
    ).not.toBeInTheDocument();
  });

  it('still calls a queued transaction queued', () => {
    renderPanel(
      [MDL_ROOT, MDL_DDL],
      'sbtest.sbtest2',
      '',
      LockType.metadata,
      'SHARED_READ'
    );

    expect(screen.getByText('1 more transaction ahead')).toBeInTheDocument();
    expect(
      screen.queryByText(/holding it independently/)
    ).not.toBeInTheDocument();
  });

  it('does not claim an open transaction when the holder has none', () => {
    // LOCK TABLES holds a metadata lock outside any transaction, so the agent reports no
    // transaction duration; saying "holding a transaction open" would be a fact it does not have.
    renderPanel(
      [
        {
          ...MDL_ROOT,
          blockingQuery: 'LOCK TABLES rta_f WRITE',
          blockerTransactionDuration: null,
          blockingLockMode: 'SHARED_NO_READ_WRITE',
        },
      ],
      'sbtest.rta_f',
      '',
      LockType.metadata,
      'SHARED_READ'
    );

    expect(screen.getByText(/still holding the lock/)).toBeInTheDocument();
    expect(
      screen.queryByText(/holding a transaction open/)
    ).not.toBeInTheDocument();
  });

  it('does say a transaction is open when one is', () => {
    renderPanel([IDLE_ROOT]);

    expect(screen.getByText(/holding a transaction open/)).toBeInTheDocument();
  });
});
