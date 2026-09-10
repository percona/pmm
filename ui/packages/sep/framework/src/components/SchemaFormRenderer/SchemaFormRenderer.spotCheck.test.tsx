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
 * snippet-parameter path. Confirms help icons appear only on described fields
 * and core inputs still mount — a regression guard for the framework-global
 * label change.
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

function expectHelp(label: string, present: boolean) {
  // Ignore the notched-outline legend clone so a legend-only render fails.
  // Bool icons sit outside <label>, so we cannot scope to label alone.
  const selector = `[data-help-for="${escapeAttrSelectorValue(label)}"]`;
  const matches = [...document.querySelectorAll(selector)];
  const visible = matches.filter(
    (el) => !el.closest('.MuiOutlinedInput-notchedOutline')
  );
  if (present) {
    expect(visible.length).toBeGreaterThan(0);
  } else {
    expect(matches).toHaveLength(0);
  }
}

describe('SchemaFormRenderer — cross-plugin help-icon spot-check', () => {
  it('mysql_backups-like create form: icons on described fields only', () => {
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

    expectHelp('Database Host', true);
    expectHelp('Server Alias', false);
    expectHelp('Compress backup data', true);
    expectHelp('Logging directory', false);
    expectHelp('Safe replica backup', true);
    expectHelp('Kill-queries timeout (s)', false);
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
    expect(
      screen.getByTestId('text-input-overrides.snip0.path')
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('text-input-overrides.snip0.threshold')
    ).toBeInTheDocument();

    expectHelp('Execution Host', false);
    expectHelp('Run with sudo', true);
    expectHelp('Lookback minutes', true);
    expectHelp('Operator note', false);
    expectHelp('Path', true);
    expectHelp('Threshold %', false);
  });

  it('snippet-execution form: user-authored params drive help icons', () => {
    // SnippetExecutionAccordion: user-authored params + Execution (sudo described, host not).
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
            // >3 choices use the select shell (help icon); ≤3 use radios + caption.
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

    expectHelp('Table Name', true);
    expectHelp('Database Name', false);
    expectHelp('Output format', true);
    expectHelp('Verbose', true);
    expectHelp('Row limit', false);
    expectHelp('Execution Host', false);
    expectHelp('Run with sudo', true);
  });
});
