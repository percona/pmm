import { stepsMarkdown } from './remediationSteps.utils';

describe('stepsMarkdown', () => {
  it('puts each command under the step it belongs to, as code', () => {
    const md = stepsMarkdown([
      'Make the replica read-only right away:',
      'SET PERSIST read_only = ON;',
      'SET PERSIST super_read_only = ON;',
      'Check for errant GTIDs:',
      '- on each server',
      'SELECT @@GLOBAL.gtid_executed;',
      '- then',
      "SELECT GTID_SUBTRACT('<replica_gtid_executed>', '<source_gtid_executed>');",
      'Check the shop schema for drift:',
      'pt-table-checksum --databases=shop h=ps_1,u=<user>,p=<pwd>',
    ]);

    expect(md).toBe(
      [
        '1. Make the replica read-only right away:',
        '',
        '   ```sql',
        '   SET PERSIST read_only = ON;',
        '   SET PERSIST super_read_only = ON;',
        '   ```',
        '',
        '2. Check for errant GTIDs:',
        '',
        '   on each server',
        '',
        '   ```sql',
        '   SELECT @@GLOBAL.gtid_executed;',
        '   ```',
        '',
        '   then',
        '',
        '   ```sql',
        "   SELECT GTID_SUBTRACT('<replica_gtid_executed>', '<source_gtid_executed>');",
        '   ```',
        '',
        '3. Check the shop schema for drift:',
        '',
        '   ```bash',
        '   pt-table-checksum --databases=shop h=ps_1,u=<user>,p=<pwd>',
        '   ```',
      ].join('\n')
    );
  });

  it('keeps plain steps plain and drops stray fences', () => {
    expect(
      stepsMarkdown([' Restart the agent ', '', '```bash', 'Check the logs'])
    ).toBe('1. Restart the agent\n\n2. Check the logs');
  });

  it('indents under two-digit numbers too', () => {
    const md = stepsMarkdown([
      ...Array.from({ length: 9 }, (_, i) => `Step ${i + 1}`),
      'Confirm:',
      'SHOW REPLICA STATUS\\G',
    ]);
    expect(
      md.endsWith(
        '10. Confirm:\n\n    ```sql\n    SHOW REPLICA STATUS\\G\n    ```'
      )
    ).toBe(true);
  });
});
