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
    lockTime: 'Lock time',
    waitEvent: 'Wait event',
    queryId: 'Query ID',
    transactionStartTime: 'Transaction start time',
  },
  tooltips: {
    operationId: "The database's internal identifier for this operation.",
    operationIdMySql:
      'The MySQL connection id running this statement. MySQL identifies connections, not individual statements, so consecutive statements on the same connection share this value.',
    operationIdPostgreSql:
      'The PostgreSQL backend process ID (pid) running this query. Consecutive queries on the same connection share this value.',
    elapsedExecTime:
      'How long this operation has been running for. For a PostgreSQL session idle in transaction, how long its transaction has been open.',
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
    state:
      'The current state of the thread (MySQL) or session (PostgreSQL) executing this statement.',
    programName:
      'The client program connected to MySQL that started this statement.',
    rowsExamined:
      'The number of rows the statement has examined so far. A high value relative to rows sent can indicate an inefficient query.',
    rowsSent: 'The number of rows the statement has returned so far.',
    fullScan:
      'Whether the statement performed a full table scan instead of using an index.',
    lockTime:
      'How long the statement has waited for table locks, in milliseconds (LOCK_TIME in performance_schema).',
    waitEvent:
      'What the PostgreSQL session is waiting for (wait_event_type: wait_event), if anything.',
    queryId:
      'PostgreSQL query identifier. Set on PostgreSQL 14+ with compute_query_id enabled.',
    transactionStartTime:
      'When the current transaction started. Long-open transactions hold locks and block vacuum.',
  },
};
