/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

/**
 * Spot-check: one representative form per SchemaFormRenderer consumer.
 *
 * Direct consumers: mysql_backups (via SchemaDrivenPlugin) and atw
 * (CollectPane); SnippetExecutionAccordion covers the synthesised
 * snippet-parameter path. Confirms field descriptions appear once as helper
 * text (never as a restating help icon) and core inputs still mount.
 */

import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { SchemaFormRenderer } from './SchemaFormRenderer';
import type { FormSection } from './types';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn() },
  useAlertConfig: () => ({ data: undefined, isLoading: false }),
}));

function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

function escapeAttrSelectorValue(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}

/** Assert a field never paints description as a help-icon tooltip. */
function expectNoHelpIcon(label: string) {
  const selector = `[data-help-for="${escapeAttrSelectorValue(label)}"]`;
  expect(document.querySelectorAll(selector)).toHaveLength(0);
}

describe('SchemaFormRenderer — cross-plugin helper-text spot-check', () => {
  it('mysql_backups-like create form: helper text on described fields only', () => {
    const sections: FormSection[] = [
      {
        title: 'Task',
        fields: [
          {
            type: 'string',
            name: 'db_host',
            label: 'Database Host',
            description: 'Host the backup connects to on the executor node.',
          },
          { type: 'string', name: 'server_alias', label: 'Server Alias' },
        ],
      },
      {
        title: 'General',
        fields: [
          {
            type: 'choice',
            name: 'storage_type',
            label: 'Storage Type',
            choices: [
              { label: 'S3-compatible', value: 's3' },
              { label: 'Filesystem', value: 'filesystem' },
            ],
          },
          {
            type: 'bool',
            name: 'compress',
            label: 'Compress backup data',
            description: 'Compress the backup stream as it is written.',
          },
          { type: 'string', name: 'log_dir', label: 'Logging directory' },
        ],
      },
      {
        title: 'XtraBackup',
        fields: [
          {
            type: 'bool',
            name: 'xtrabackup_stop_replica',
            label: 'Safe replica backup',
            description:
              'Passes --safe-slave-backup so xtrabackup pauses the replica SQL thread during the backup.',
          },
          {
            type: 'integer',
            name: 'xtrabackup_kill_queries_timeout',
            label: 'Kill-queries timeout (s)',
          },
        ],
      },
    ];

    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    expect(screen.getByTestId('text-input-db_host')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-server_alias')).toBeInTheDocument();
    expect(screen.getByTestId('switch-input-compress')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-log_dir')).toBeInTheDocument();
    expect(
      screen.getByTestId('switch-input-xtrabackup_stop_replica')
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('text-input-xtrabackup_kill_queries_timeout')
    ).toBeInTheDocument();

    expect(
      screen.getByText('Host the backup connects to on the executor node.')
    ).toBeInTheDocument();
    expect(
      screen.getByText('Compress the backup stream as it is written.')
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        'Passes --safe-slave-backup so xtrabackup pauses the replica SQL thread during the backup.'
      )
    ).toBeInTheDocument();

    expectNoHelpIcon('Database Host');
    expectNoHelpIcon('Server Alias');
    expectNoHelpIcon('Compress backup data');
    expectNoHelpIcon('Logging directory');
    expectNoHelpIcon('Safe replica backup');
    expectNoHelpIcon('Kill-queries timeout (s)');
  });

  it('atw CollectPane-like form: shared section plus namespaced per-snippet overrides', () => {
    // CollectPane shape: shared params, then namespaced per-snippet overrides.
    const sections: FormSection[] = [
      {
        title: 'Shared parameters',
        fields: [
          {
            type: 'host',
            name: 'executor_host',
            label: 'Execution Host',
            required: true,
          },
          {
            type: 'bool',
            name: 'sudo',
            label: 'Run with sudo',
            description:
              'Prepend sudo to the interpreter when the snippet is executed.',
          },
          {
            type: 'integer',
            name: 'minutes',
            label: 'Lookback minutes',
            description: 'Shared window applied to every selected snippet.',
          },
          { type: 'string', name: 'note', label: 'Operator note' },
        ],
      },
      {
        title: 'Disk usage check',
        description: 'Reports free space on the executor host.',
        collapsible: true,
        fields: [
          {
            type: 'string',
            name: 'overrides.snip0.path',
            label: 'Path',
            description: 'Filesystem path to inspect for this snippet only.',
          },
          {
            type: 'integer',
            name: 'overrides.snip0.threshold',
            label: 'Threshold %',
          },
        ],
      },
    ];

    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    expect(screen.getByLabelText(/Execution Host/i)).toBeInTheDocument();
    expect(screen.getByTestId('switch-input-sudo')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-minutes')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-note')).toBeInTheDocument();
    // Collapsible section starts expanded only when collapsed_by_default is unset;
    // this fixture leaves it open so override fields mount.
    expect(
      screen.getByTestId('text-input-overrides.snip0.path')
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('text-input-overrides.snip0.threshold')
    ).toBeInTheDocument();

    expect(
      screen.getByText(
        'Prepend sudo to the interpreter when the snippet is executed.'
      )
    ).toBeInTheDocument();
    expect(
      screen.getByText('Shared window applied to every selected snippet.')
    ).toBeInTheDocument();
    expect(
      screen.getByText('Filesystem path to inspect for this snippet only.')
    ).toBeInTheDocument();
    // Section prose must not restate field help.
    expect(
      screen.queryByText('Reports free space on the executor host.')
    ).not.toBeInTheDocument();

    expectNoHelpIcon('Execution Host');
    expectNoHelpIcon('Run with sudo');
    expectNoHelpIcon('Lookback minutes');
    expectNoHelpIcon('Operator note');
    expectNoHelpIcon('Path');
    expectNoHelpIcon('Threshold %');
  });

  it('snippet-execution form: user-authored params drive helper text once', () => {
    // SnippetExecutionAccordion: user-authored params + Execution.
    const sections: FormSection[] = [
      {
        title: 'Parameters',
        fields: [
          {
            type: 'string',
            name: 'table_name',
            label: 'Table Name',
            description: 'Table to inspect on the executor host.',
          },
          { type: 'string', name: 'database_name', label: 'Database Name' },
          {
            type: 'choice',
            name: 'format',
            label: 'Output format',
            description: 'How to render the snippet result.',
            // >3 choices use the select shell; ≤3 use radios + caption.
            choices: [
              { label: 'Plain text', value: 'text' },
              { label: 'JSON', value: 'json' },
              { label: 'CSV', value: 'csv' },
              { label: 'YAML', value: 'yaml' },
            ],
          },
          {
            type: 'bool',
            name: 'verbose',
            label: 'Verbose',
            description: 'Increase output verbosity.',
          },
          { type: 'integer', name: 'limit', label: 'Row limit' },
        ],
      },
      {
        title: 'Execution',
        fields: [
          {
            type: 'host',
            name: 'executor_host',
            label: 'Execution Host',
            required: true,
          },
          {
            type: 'bool',
            name: 'sudo',
            label: 'Run with sudo',
            description:
              'Prepend sudo to the interpreter when the snippet is executed.',
          },
        ],
      },
    ];

    renderWithProviders(
      <SchemaFormRenderer sections={sections} onSubmit={() => {}} />
    );

    expect(screen.getByTestId('text-input-table_name')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-database_name')).toBeInTheDocument();
    expect(screen.getByTestId('select-format-button')).toBeInTheDocument();
    expect(screen.getByTestId('switch-input-verbose')).toBeInTheDocument();
    expect(screen.getByTestId('text-input-limit')).toBeInTheDocument();
    expect(screen.getByLabelText(/Execution Host/i)).toBeInTheDocument();
    expect(screen.getByTestId('switch-input-sudo')).toBeInTheDocument();

    expect(
      screen.getByText('Table to inspect on the executor host.')
    ).toBeInTheDocument();
    expect(
      screen.getByText('How to render the snippet result.')
    ).toBeInTheDocument();
    expect(screen.getByText('Increase output verbosity.')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Prepend sudo to the interpreter when the snippet is executed.'
      )
    ).toBeInTheDocument();

    expectNoHelpIcon('Table Name');
    expectNoHelpIcon('Database Name');
    expectNoHelpIcon('Output format');
    expectNoHelpIcon('Verbose');
    expectNoHelpIcon('Row limit');
    expectNoHelpIcon('Execution Host');
    expectNoHelpIcon('Run with sudo');
  });
});
