export const Messages = {
  label: 'Truncated',
  tooltip:
    'MySQL keeps only the first 1024 bytes of a statement by default, so this text is incomplete. ' +
    'To see up to 64 KiB, raise performance_schema_max_sql_text_length, restart MySQL, ' +
    'and turn on the events_statements_current consumer.',
};
