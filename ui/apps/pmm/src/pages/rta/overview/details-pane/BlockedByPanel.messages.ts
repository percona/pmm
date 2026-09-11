export const Messages = {
  // One transaction is responsible, so it can be named.
  blockedByOne: (connId: number | string) => `Blocked by ${connId}`,
  // Several independent transactions are holding this statement up; naming one would
  // wrongly imply that resolving it is enough.
  blockedByMany: (count: number) => `Blocked by ${count} transactions`,
  blockedUnknownHolder: 'Blocked on a lock',
  waitingFor: (duration: string) => `waiting ${duration}`,
  blockerStatement: "Blocker's statement",
  // Only when a transaction really is open. LOCK TABLES holds a metadata lock outside any
  // transaction, so claiming one would send the reader looking for something that is not there.
  idleNote:
    'Not executing anything right now — it is holding a transaction open. This is the statement that took the lock.',
  idleNoteNoTransaction:
    'Not executing anything right now, but still holding the lock. This is the statement that took it.',
  titles: {
    blockerState: 'Blocker state',
    blockerUser: 'Blocker user',
    lockedTable: 'Locked table',
    lockedIndex: 'Locked index',
    lockType: 'Lock type',
    requestedMode: 'Mode requested',
    blockingMode: 'Blocker holds',
  },
  // The two mechanisms are freed differently, so the reader is told which one this is
  // rather than left to infer it from whether an index is named.
  lockTypes: {
    row: 'Row lock (InnoDB)',
    metadata: 'Metadata lock (MDL)',
  },
  idleInTransaction: (age: string) => `Sleep · idle in transaction ${age}`,
  // Queued in front of this statement, but not the cause: they clear on their own.
  otherBlockers: (count: number) =>
    count === 1
      ? '1 more transaction ahead'
      : `${count} more transactions ahead`,
  // Independent holders. Calling these "ahead" contradicted the hint that counts them as
  // transactions the statement stays blocked on until every one is resolved.
  otherRootBlockers: (count: number) =>
    count === 1
      ? '1 more transaction holding it independently'
      : `${count} more transactions holding it independently`,
  root: 'Root',
  // Shown when the statement is flagged as waiting but the holder was not in the same
  // snapshot — the lock graph and the statement list are read milliseconds apart.
  unknownBlocker:
    'The transaction holding the lock was not in this snapshot. The next refresh should show it.',
  // Only true when a single transaction is responsible.
  resolveHint: (connId: number | string) =>
    `Resolving conn ${connId} — committing or rolling back its transaction — releases this statement. PMM reports the wait; what to do about it is yours to decide.`,
  // A metadata lock is held for as long as the holder's statement or transaction touches
  // the table, so the remedy is worded for that rather than for a row lock.
  resolveHintMetadata: (connId: number | string) =>
    `Conn ${connId} holds a metadata lock on this table; it is released when that statement or transaction ends. PMM reports the wait; what to do about it is yours to decide.`,
  // Only the transactions at the head of the chain hold the statement up independently; the
  // rest are queued behind them and clear on their own.
  resolveHintRoots: (count: number, queued: number) =>
    `${count} transactions are holding this statement up independently, so it stays blocked until all ${count} are resolved.${
      queued > 0
        ? ' The rest are queued behind them and clear on their own.'
        : ''
    }`,
  // No transaction in the graph is free of waiting — a cycle, or a graph read only in part.
  // Nothing here can honestly be pointed at as the one to resolve.
  resolveHintCycle:
    'Every transaction involved is itself waiting, so none of them can be singled out as the one to resolve.',
};
