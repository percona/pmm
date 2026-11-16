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
  blockedUnattributed: 'Possibly blocked',
  blockedUnreadable: 'Blocked by unknown',
  tooltipUnattributed:
    "This statement might have been blocked, but its connection moved on to a later statement before the lock check. It isn't included in the Blocked only count. The next refresh checks again.",
  tooltipUnreadable:
    "This statement is waiting for a lock, but this instance doesn't report who holds this lock type. To show the holders of metadata locks, enable the wait/lock/metadata/sql/mdl instrument, then restart the session.",
  tooltipUnknownHolder:
    "This statement is waiting for a lock, but the transaction that holds it isn't listed yet. The next refresh usually shows it.",
};
