import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  getMysqlBackupsScheduleWarning,
  getMysqlBackupsTaskExecuteActions,
  getMysqlRestoreConfirmDetails,
  isMysqlRestoreTask,
  MysqlRestoreConfirmContent,
} from './restoreExecuteConfirm';

describe('isMysqlRestoreTask', () => {
  it('detects restores via related-app plugin name', () => {
    expect(isMysqlRestoreTask({ name: 'r1' }, 'mysql_backups/restore')).toBe(
      true
    );
  });

  it('detects restores via stored overwrite_tables', () => {
    expect(
      isMysqlRestoreTask({
        name: 'r1',
        data: { _form: { backup_source: '/b', overwrite_tables: false } },
      })
    ).toBe(true);
  });

  it('detects restores via stored backup_source alone', () => {
    expect(
      isMysqlRestoreTask({
        name: 'r1',
        data: { _form: { backup_source: '/b' } },
      })
    ).toBe(true);
  });

  it('detects restores via top-level overwrite + backup_source', () => {
    expect(
      isMysqlRestoreTask({
        name: 'r1',
        backup_source: '/b',
        overwrite_tables: true,
      })
    ).toBe(true);
  });

  it('ignores backup tasks without restore fields', () => {
    expect(
      isMysqlRestoreTask(
        {
          name: 'b1',
          data: { _form: { task_name: 'b1', hostname: 'db' } },
        },
        'mysql_backups'
      )
    ).toBe(false);
  });
});

describe('getMysqlRestoreConfirmDetails', () => {
  it('prefers stored form source/database and response host:port', () => {
    expect(
      getMysqlRestoreConfirmDetails({
        name: 'r1',
        host: '10.30.50.130',
        port: 3306,
        hostname: 'executor-node',
        data: {
          _form: {
            backup_source: 's3://bucket/run-1',
            schema_id: 'app_db',
            overwrite_tables: true,
            hostname: 'executor-node',
          },
        },
      })
    ).toEqual({
      source: 's3://bucket/run-1',
      targetHost: '10.30.50.130:3306',
      targetDatabase: 'app_db',
      overwriteTables: true,
    });
  });

  it('falls back to Not set and resolves hydrated schema refs', () => {
    expect(
      getMysqlRestoreConfirmDetails({
        name: 'r1',
        data: {
          _form: {
            backup_source: '/local/backup',
            overwrite_tables: false,
            schema_id: { id: 9, name: 'orders' },
          },
        },
      })
    ).toEqual({
      source: '/local/backup',
      targetHost: 'Not set',
      targetDatabase: 'orders',
      overwriteTables: false,
    });
  });
});

describe('getMysqlBackupsTaskExecuteActions', () => {
  it('returns undefined for non-restore tasks', () => {
    expect(
      getMysqlBackupsTaskExecuteActions(
        { name: 'backup-1' },
        { pluginName: 'mysql_backups' }
      )
    ).toBeUndefined();
  });

  it('returns an Execute action for restore plugin even without _form', () => {
    const actions = getMysqlBackupsTaskExecuteActions(
      {
        name: 'test-myloader',
        host: 'db.example',
        port: 3306,
      },
      { pluginName: 'mysql_backups/restore' }
    );

    expect(actions).toHaveLength(1);
    expect(actions?.[0]).toMatchObject({
      label: 'Execute',
      taskName: 'test-myloader',
      testId: 'mysql-restore-execute',
    });
    expect(actions?.[0].confirmContent).toBeTruthy();
  });

  it('returns an Execute action with confirm content for restores', () => {
    const actions = getMysqlBackupsTaskExecuteActions({
      name: 'test-myloader',
      host: 'db.example',
      port: 3306,
      data: {
        _form: {
          backup_source: '/backups/latest',
          schema_id: 'demo',
          overwrite_tables: true,
        },
      },
    });

    expect(actions).toHaveLength(1);
    expect(actions?.[0]).toMatchObject({
      label: 'Execute',
      taskName: 'test-myloader',
      testId: 'mysql-restore-execute',
    });
    expect(actions?.[0].confirmContent).toBeTruthy();
  });
});

describe('getMysqlBackupsScheduleWarning', () => {
  it('returns undefined for non-restore plugins', () => {
    expect(
      getMysqlBackupsScheduleWarning('backup-1', {
        pluginName: 'mysql_backups',
        tasks: [{ name: 'backup-1' }],
      })
    ).toBeUndefined();
  });

  it('returns schedule confirm content for restore plugin tasks', () => {
    const warning = getMysqlBackupsScheduleWarning('test-myloader', {
      pluginName: 'mysql_backups/restore',
      tasks: [
        {
          name: 'test-myloader',
          host: '10.30.50.130',
          port: 3306,
          data: {
            _form: {
              backup_source: '/backups/latest',
              schema_id: 'demo',
              overwrite_tables: true,
            },
          },
        },
      ],
    });

    expect(warning).toBeTruthy();
    render(<>{warning}</>);
    expect(
      screen.getByTestId('mysql-restore-schedule-confirm')
    ).toHaveTextContent('schedule this restore');
    expect(
      screen.getByTestId('mysql-restore-schedule-confirm')
    ).toHaveTextContent('Source backup: /backups/latest');
    expect(
      screen.getByTestId('mysql-restore-overwrite-alert')
    ).toBeInTheDocument();
  });
});

describe('MysqlRestoreConfirmContent', () => {
  it('renders source, target, and overwrite warning when overwriting', () => {
    render(
      <MysqlRestoreConfirmContent
        details={{
          source: '/backups/latest',
          targetHost: '10.30.50.130:3306',
          targetDatabase: 'demo',
          overwriteTables: true,
        }}
      />
    );

    const root = screen.getByTestId('mysql-restore-execute-confirm');
    expect(root).toHaveTextContent('Source backup: /backups/latest');
    expect(root).toHaveTextContent('Target host: 10.30.50.130:3306');
    expect(root).toHaveTextContent('Target database: demo');
    expect(root).toHaveTextContent('Overwrite tables: Yes');
    expect(
      screen.getByTestId('mysql-restore-overwrite-alert')
    ).toBeInTheDocument();
  });

  it('omits overwrite alert when overwrite is off', () => {
    render(
      <MysqlRestoreConfirmContent
        details={{
          source: '/b',
          targetHost: 'h',
          targetDatabase: 'd',
          overwriteTables: false,
        }}
      />
    );

    expect(
      screen.getByTestId('mysql-restore-execute-confirm')
    ).toHaveTextContent('Overwrite tables: No');
    expect(
      screen.queryByTestId('mysql-restore-overwrite-alert')
    ).not.toBeInTheDocument();
  });
});
