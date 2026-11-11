export const Messages = {
  blockedByOne: (connId: number | string) => `Blocked by ${connId}`,
  blockedByMany: (count: number) => `Blocked by ${count} transactions`,
  blockedUnknownHolder: 'Blocked on a lock',
  waitingFor: (duration: string) => `waiting ${duration}`,
  blockerStatement: "Blocker's statement",
  // Only when a transaction is open: LOCK TABLES holds a metadata lock outside any.
  idleNote:
    'The blocker ran this statement last and is now idle, but its open transaction still holds the lock. The lock may come from an earlier statement. To release it, commit or roll back the transaction, or end the connection.',
  idleNoteNoTransaction:
    'The blocker ran this statement last and is now idle, but it still holds the lock, for example after LOCK TABLES. The lock is released when that connection unlocks it, for example with UNLOCK TABLES, or when the connection ends.',
  noStatementIdleInTransaction:
    'No current statement: the connection is idle in a transaction, and the server reported no last statement. To see the last statement of idle connections, enable the events_statements_current consumer.',
  noStatementIdle:
    'No current statement: the connection is idle, and the server reported no last statement. To see the last statement of idle connections, enable the events_statements_current consumer.',
  noStatement: 'The server did not report a statement for this connection.',
  titles: {
    blockerState: 'Blocker state',
    blockerUser: 'Blocker user',
    lockedTable: 'Locked table',
    lockedIndex: 'Locked index',
    lockType: 'Lock type',
    requestedMode: 'Requested mode',
    blockingMode: "Blocker's mode",
  },
  lockTypes: {
    row: 'Row lock (InnoDB)',
    metadata: 'Metadata lock (MDL)',
  },
  idleInTransaction: (age: string) => `Sleep · idle in transaction ${age}`,
  otherBlockers: (count: number) =>
    count === 1
      ? '1 more transaction ahead'
      : `${count} more transactions ahead`,
  otherRootBlockers: (count: number) =>
    count === 1
      ? '1 more transaction holding it independently'
      : `${count} more transactions holding it independently`,
  root: 'Root',
  // The lock graph and the statement list are read milliseconds apart.
  unknownBlocker:
    'The transaction holding the lock was not in this snapshot. The next refresh should show it.',
  resolveHint: (connId: number | string) =>
    `Committing or rolling back the transaction of connection ${connId} releases this statement.`,
  resolveHintMetadata: (connId: number | string) =>
    `Connection ${connId} holds a metadata lock on this table. The lock is released when that statement or transaction ends.`,
  resolveHintRoots: (count: number, queued: number) =>
    `${count} transactions hold locks that this statement waits for, so it stays blocked until all ${count} finish.${
      queued > 0
        ? ' The others are queued behind them and clear on their own.'
        : ''
    }`,
  // A blocker's own row shows what it waits on, one step closer to the root.
  resolveHintWaiting: (connId: number | string) =>
    `Connection ${connId} is itself waiting for a lock, so it can't finish until that wait ends. To see what it waits for, open its row.`,
  resolveHintAllWaiting: (count: number) =>
    `All ${count} transactions that block this statement are themselves waiting for locks. To see what they wait for, open their rows.`,
};
