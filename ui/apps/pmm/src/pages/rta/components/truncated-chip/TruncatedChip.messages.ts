export const Messages = {
  label: 'Truncated',
  tooltip:
    'MySQL keeps only the beginning of a long statement, so this text is incomplete. ' +
    'How much it keeps is a MySQL server setting: performance_schema_max_sql_text_length ' +
    '(1024 bytes by default, read at server startup). PMM can show up to that many bytes only ' +
    'while the events_statements_current consumer is enabled, and at most 64 KiB.',
  tooltipPostgreSql:
    'PostgreSQL keeps only the first track_activity_query_size bytes (1024 by default) of a running query, ' +
    'so this text is incomplete. Raising the setting needs a PostgreSQL server restart.',
};
