export const Messages = {
  // One transaction is responsible, so it can be named.
  blockedBy: (connId: number | string) => `Blocked by ${connId}`,
  // Several transactions are responsible, or the graph names no single one. The word matters:
  // a bare number here would read as a connection id rather than a count.
  blockedByMany: (count: number) =>
    count === 1
      ? 'Blocked by 1 transaction'
      : `Blocked by ${count} transactions`,
  // The statement is waiting but no holder came with this snapshot.
  blockedUnknownHolder: 'Blocked',
  tooltip: (connId: number | string) =>
    `This statement is waiting for a row lock held by connection ${connId}.`,
  tooltipMany: (count: number) =>
    count === 1
      ? 'This statement is waiting for a row lock. Open the row to see which transaction holds it.'
      : `This statement is waiting for a row lock held by ${count} transactions. Open the row to see them.`,
  tooltipUnknownHolder:
    'This statement is waiting for a row lock. The transaction holding it was not in this snapshot; the next refresh should show it.',
};
