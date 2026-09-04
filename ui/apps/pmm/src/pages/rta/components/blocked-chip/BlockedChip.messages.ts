export const Messages = {
  // One transaction is responsible, so it can be named.
  blockedBy: (connId: number | string) => `Blocked by ${connId}`,
  // Several transactions are holding the statement up, or the graph does not identify a
  // single one. Naming one would imply resolving it is enough, which it is not.
  blockedByMany: (count: number) => `Blocked by ${count}`,
  tooltip: (connId: number | string) =>
    `This statement is waiting for a row lock held by connection ${connId}.`,
  tooltipMany: (count: number) =>
    `This statement is waiting for a row lock held by ${count} transactions. Open the row to see them.`,
};
