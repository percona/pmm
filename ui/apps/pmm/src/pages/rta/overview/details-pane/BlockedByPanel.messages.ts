export const Messages = {
  // One transaction is responsible, so it can be named.
  blockedByOne: (connId: number | string) => `Blocked by ${connId}`,
  // Several independent transactions are holding this statement up; naming one would
  // wrongly imply that resolving it is enough.
  blockedByMany: (count: number) => `Blocked by ${count} transactions`,
  blockedUnknownHolder: 'Blocked on a row lock',
  waitingFor: (duration: string) => `waiting ${duration}`,
  blockerStatement: "Blocker's statement",
  idleNote:
    'Not executing anything right now — it is holding a transaction open. This is the statement that took the lock.',
  titles: {
    blockerState: 'Blocker state',
    blockerUser: 'Blocker user',
    lockedTable: 'Locked table',
    lockedIndex: 'Locked index',
  },
  idleInTransaction: (age: string) => `Sleep · idle in transaction ${age}`,
  otherBlockers: (count: number) =>
    count === 1
      ? '1 more transaction ahead'
      : `${count} more transactions ahead`,
  root: 'Root',
  // Shown when the statement is flagged as waiting but the holder was not in the same
  // snapshot — the lock graph and the statement list are read milliseconds apart.
  unknownBlocker:
    'The transaction holding the lock was not in this snapshot. The next refresh should show it.',
  // Only true when a single transaction is responsible.
  resolveHint: (connId: number | string) =>
    `Resolving conn ${connId} — committing or rolling back its transaction — releases this statement. PMM reports the wait; what to do about it is yours to decide.`,
  // Only the transactions at the head of the chain hold the statement up independently; the
  // rest are queued behind them and clear on their own.
  resolveHintRoots: (count: number) =>
    `${count} transactions are holding this statement up independently, so it stays blocked until all ${count} are resolved. The others listed are queued behind them.`,
  // No transaction in the graph is free of waiting — a cycle, or a graph read only in part.
  // Nothing here can honestly be pointed at as the one to resolve.
  resolveHintCycle:
    'Every transaction involved is itself waiting, so none of them can be singled out as the one to resolve.',
};
