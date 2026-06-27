export const Messages = {
  titles: {
    operationId: 'Operation ID',
    elapsedExecTime: 'Elapsed exec. time',
    planSummary: 'Plan summary',
    databaseName: 'Database name',
    collection: 'Collection',
    operation: 'Operation',
    username: 'User name',
    dbInstanceAddress: 'DB instance address',
    clientAppName: 'Client app name',
    operationStartTime: 'Operation start time',
    dataCaptureTime: 'Data capture time',
    clientAddress: 'Client address',
    service: 'Service',
    command: 'Command',
    state: 'State',
    programName: 'Program name',
    rowsExamined: 'Rows examined',
    rowsSent: 'Rows sent',
    fullScan: 'Full scan',
  },
  tooltips: {
    operationId: "The database's internal identifier for this operation.",
    elapsedExecTime: 'How long this operation has been running for.',
    planSummary:
      'High-level summary of how the database is executing this query. For example, using an index or scanning the full collection.',
    databaseName: 'The database/schema where this operation is running.',
    collection: 'The MongoDB collection targeted by this operation',
    operation:
      'The type of action the database is performing, such as query, insert, update, or command.',
    username: 'The database user who started this operation.',
    clientAddress:
      'The IP address and port of the application sending this query.',
    service: 'The PMM service name for this database instance.',
    clientAppName:
      'The name of the application or driver that started this operation.',
    operationStartTime:
      'The exact timestamp when the database started executing this operation.',
    dataCaptureTime:
      'When PMM took this snapshot. Compare with Operation start time to calculate how long the operation has been running so far.',
    dbInstanceAddress:
      'The server hostname and port where this operation is running.',
    command:
      'The type of command the connection is executing, such as Query or Execute.',
    state: 'The current state of the thread executing this statement.',
    programName:
      'The client program connected to MySQL that started this statement.',
    rowsExamined:
      'The number of rows the statement has examined so far. A high value relative to rows sent can indicate an inefficient query.',
    rowsSent: 'The number of rows the statement has returned so far.',
    fullScan:
      'Whether the statement performed a full table scan instead of using an index.',
  },
};
