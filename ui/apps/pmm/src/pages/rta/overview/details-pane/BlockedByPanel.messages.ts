export const Messages = {
  blockedByOne: (connId: number | string) => `Blocked by ${connId}`,
  blockedByMany: (count: number) => `Blocked by ${count} transactions`,
  blockedUnknownHolder: 'Blocked on a lock',
  waitingFor: (duration: string) => `waiting ${duration}`,
  blockerStatement: "Blocker's statement",
  // Only when a transaction is open: LOCK TABLES holds a metadata lock outside any.
  idleNote:
    'Not executing anything right now — it is holding a transaction open. Shown is the last statement it ran; the lock may have been taken by an earlier statement in the same transaction.',
  idleNoteNoTransaction:
    'Not executing anything right now, but still holding the lock. Shown is the last statement it ran; the lock may have been taken by an earlier one.',
  noStatementIdleInTransaction:
    'No current statement (connection idle in transaction). The server reported no last statement for it either; enable the events_statements_current consumer to see the last statement an idle connection ran.',
  noStatementIdle:
    'No current statement (connection idle). The server reported no last statement for it either; enable the events_statements_current consumer to see the last statement an idle connection ran.',
  noStatement: 'The server did not report a statement for this connection.',
  titles: {
    blockerState: 'Blocker state',
    blockerUser: 'Blocker user',
    lockedTable: 'Locked table',
    lockedIndex: 'Locked index',
    lockType: 'Lock type',
    requestedMode: 'Mode requested',
    blockingMode: 'Blocker holds',
  },
  lockTypes: {
    row: 'Row lock (InnoDB)',
    metadata: 'Metadata lock (MDL)',
  },
  idleInTransaction: (command: string, age: string) =>
    command === 'Sleep'
      ? `Sleep · idle in transaction ${age}`
      : `${command} ${age}`,
  otherBlockers: (count: number) =>
    count === 1
      ? '1 more transaction ahead'
      : `${count} more transactions ahead`,
  otherRootBlockers: (count: number) =>
    count === 1
      ? '1 more transaction holding it independently'
      : `${count} more transactions holding it independently`,
  // Not "ahead": nothing in a cycle clears on its own.
  cycleParticipants: (count: number) =>
    count === 1
      ? '1 more transaction in the cycle'
      : `${count} more transactions in the cycle`,
  root: 'Root',
  // The lock graph and the statement list are read milliseconds apart.
  unknownBlocker:
    'The transaction holding the lock was not in this snapshot. The next refresh should show it.',
  resolveHint: (connId: number | string) =>
    `Resolving conn ${connId} — committing or rolling back its transaction — releases this statement. PMM reports the wait; what to do about it is yours to decide.`,
  resolveHintMetadata: (connId: number | string) =>
    `Conn ${connId} holds a metadata lock on this table; it is released when that statement or transaction ends. PMM reports the wait; what to do about it is yours to decide.`,
  // A PostgreSQL session idle outside any transaction can only hold a session-level lock.
  resolveHintSessionLock: (connId: number | string) =>
    `Conn ${connId} is idle outside any transaction, so its lock is session-level, such as an advisory lock from pg_advisory_lock(): committing or rolling back does not release it. Calling pg_advisory_unlock() in that session, or ending the session, releases this statement. PMM reports the wait; what to do about it is yours to decide.`,
  resolveHintRoots: (count: number, queued: number) =>
    `${count} transactions are holding this statement up independently, so it stays blocked until all ${count} are resolved.${
      queued > 0
        ? ' The rest are queued behind them and clear on their own.'
        : ''
    }`,
  resolveHintCycle:
    'Every transaction involved is itself waiting, so none of them can be singled out as the one to resolve.',
};
