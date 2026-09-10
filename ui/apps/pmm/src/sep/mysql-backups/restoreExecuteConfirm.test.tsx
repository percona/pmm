import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  getMysqlBackupsScheduleWarning,
  getMysqlBackupsTaskExecuteActions,
  getMysqlRestoreConfirmDetails,
  isMysqlRestoreTask,
  MysqlRestoreConfirmContent,
} from './restoreExecuteConfirm';

const { mockUseSchemas } = vi.hoisted(() => ({ mockUseSchemas: vi.fn() }));

vi.mock('@sep/framework', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/framework')>()),
  useSchemas: (...args: unknown[]) => mockUseSchemas(...args),
}));

beforeEach(() => {
  mockUseSchemas.mockReset();
});

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

  it('does not read restore fields off the task response itself', () => {
    expect(
      isMysqlRestoreTask({
        name: 'r1',
        backup_source: '/b',
        overwrite_tables: true,
      })
    ).toBe(false);
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
  it('prefers stored form source, response host:port and the inventory schema', () => {
    expect(
      getMysqlRestoreConfirmDetails({
        name: 'r1',
        host: '10.30.50.130',
        port: 3306,
        hostname: 'executor-node',
        data: {
          _form: {
            backup_source: 's3://bucket/run-1',
            service_id: '4',
            schema_id: '9',
            overwrite_tables: true,
            hostname: 'executor-node',
          },
        },
      })
    ).toEqual({
      source: 's3://bucket/run-1',
      targetHost: '10.30.50.130:3306',
      targetDatabase: 'Unknown (inventory ID 9)',
      targetSchema: { serviceId: 4, schemaId: 9 },
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

  it('does not treat executor hostname as the target host', () => {
    expect(
      getMysqlRestoreConfirmDetails({
        name: 'r1',
        hostname: 'executor-node',
        data: {
          _form: {
            backup_source: '/b',
            hostname: 'executor-node',
            overwrite_tables: true,
          },
        },
      })
    ).toEqual({
      source: '/b',
      targetHost: 'Not set',
      targetDatabase: 'Same databases as in the backup',
      overwriteTables: true,
    });
  });

  it('reads a typed database name as the backup databases, as the restore does', () => {
    expect(
      getMysqlRestoreConfirmDetails({
        name: 'r1',
        data: {
          _form: {
            backup_source: '/b',
            service_id: '4',
            schema_id: 'app_db',
            overwrite_tables: false,
          },
        },
      }).targetDatabase
    ).toBe('Same databases as in the backup');
  });

  it('does not show a bare port as the target host', () => {
    expect(
      getMysqlRestoreConfirmDetails({
        name: 'r1',
        port: 3306,
        data: { _form: { backup_source: '/b', overwrite_tables: false } },
      }).targetHost
    ).toBe('Not set');
  });

  it('reports overwrite as unknown when the task has no stored form', () => {
    expect(
      getMysqlRestoreConfirmDetails({
        name: 'r1',
        host: 'db.example',
        port: 3306,
      })
    ).toEqual({
      source: 'Not set',
      targetHost: 'db.example:3306',
      targetDatabase: 'Not set',
      overwriteTables: undefined,
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

  it('warns when the selected restore is missing from the task list', () => {
    const warning = getMysqlBackupsScheduleWarning('missing-restore', {
      pluginName: 'mysql_backups/restore',
      tasks: [{ name: 'other-restore' }],
    });

    expect(warning).toBeTruthy();
    render(<>{warning}</>);
    expect(
      screen.getByTestId('mysql-restore-schedule-confirm-incomplete')
    ).toHaveTextContent(
      "Could not load details for restore task 'missing-restore'"
    );
    expect(
      screen.queryByTestId('mysql-restore-schedule-confirm')
    ).not.toBeInTheDocument();
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
    expect(
      screen.queryByTestId('mysql-restore-overwrite-unknown-alert')
    ).not.toBeInTheDocument();
  });

  it('warns instead of reading an unrecorded overwrite as No', () => {
    render(
      <MysqlRestoreConfirmContent
        details={{
          source: 'Not set',
          targetHost: 'h',
          targetDatabase: 'Not set',
          overwriteTables: undefined,
        }}
      />
    );

    expect(
      screen.getByTestId('mysql-restore-execute-confirm')
    ).toHaveTextContent('Overwrite tables: Unknown');
    expect(
      screen.getByTestId('mysql-restore-overwrite-unknown-alert')
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('mysql-restore-overwrite-alert')
    ).not.toBeInTheDocument();
  });

  it('names the target database resolved from inventory', () => {
    mockUseSchemas.mockReturnValue({
      data: [
        { id: 8, name: 'billing' },
        { id: 9, name: 'orders' },
      ],
      isLoading: false,
    });

    render(
      <MysqlRestoreConfirmContent
        details={{
          source: '/b',
          targetHost: 'h:3306',
          targetDatabase: 'Unknown (inventory ID 9)',
          targetSchema: { serviceId: 4, schemaId: 9 },
          overwriteTables: false,
        }}
      />
    );

    expect(mockUseSchemas).toHaveBeenCalledWith({ serviceId: 4 });
    expect(
      screen.getByTestId('mysql-restore-execute-confirm')
    ).toHaveTextContent('Target database: orders');
  });

  it('keeps the inventory ID when the schema no longer resolves', () => {
    mockUseSchemas.mockReturnValue({ data: [], isLoading: false });

    render(
      <MysqlRestoreConfirmContent
        details={{
          source: '/b',
          targetHost: 'h:3306',
          targetDatabase: 'Unknown (inventory ID 9)',
          targetSchema: { serviceId: 4, schemaId: 9 },
          overwriteTables: false,
        }}
      />
    );

    expect(
      screen.getByTestId('mysql-restore-execute-confirm')
    ).toHaveTextContent('Target database: Unknown (inventory ID 9)');
  });
});
