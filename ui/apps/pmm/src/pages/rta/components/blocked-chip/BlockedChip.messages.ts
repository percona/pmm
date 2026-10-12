export const Messages = {
  blockedBy: (connId: number | string) => `Blocked by ${connId}`,
  // Spelled out: a bare number would read as a connection id.
  blockedByMany: (count: number) =>
    count === 1
      ? 'Blocked by 1 transaction'
      : `Blocked by ${count} transactions`,
  blockedUnknownHolder: 'Blocked',
  // "a lock", not "a row lock": the chip also covers metadata locks.
  tooltip: (connId: number | string) =>
    `This statement is waiting for a lock held by connection ${connId}.`,
  tooltipMany: (count: number) =>
    count === 1
      ? 'This statement is waiting for a lock. Open the row to see which transaction holds it.'
      : `This statement is waiting for a lock held by ${count} transactions. Open the row to see them.`,
  blockedUnattributed: 'Blocked: unknown',
  tooltipUnattributed:
    'This connection was waiting for a lock, but by the time PMM read the locks it had moved on from this statement to a later one. PMM cannot tell whether this statement was blocked, so it is not counted as blocked. The next refresh reads both again.',
  tooltipUnreadable:
    "This statement is waiting for a lock, but PMM can't read that lock type on this instance, so it can't name the holder. For metadata locks, enable the wait/lock/metadata/sql/mdl instrument and restart the Real-Time Analytics session.",
  tooltipUnknownHolder:
    'This statement is waiting for a lock. The transaction holding it was not in this snapshot; the next refresh should show it.',
};
