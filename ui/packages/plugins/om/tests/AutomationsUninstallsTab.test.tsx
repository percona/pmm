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

import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AutomationsPage } from '../src/AutomationsPage';
import type {
  OmGetOperationResponse,
  OmOperation,
  OmUninstallReport,
} from '../src/types';

const {
  useOmBootstrapRuns,
  useOmOperations,
  useOmOperation,
  useUninstallReport,
} = vi.hoisted(() => ({
  useOmBootstrapRuns: vi.fn(),
  useOmOperations: vi.fn(),
  useOmOperation: vi.fn(),
  useUninstallReport: vi.fn(),
}));

vi.mock('../src/inventoryHooks', () => ({ useOmBootstrapRuns }));
vi.mock('../src/operationHooks', async () => ({
  ...(await vi.importActual<typeof import('../src/operationHooks')>(
    '../src/operationHooks'
  )),
  useOmOperations,
  useOmOperation,
  useUninstallReport,
}));
vi.mock('../src/AutomationsScansTab', () => ({
  AutomationsScansTab: () => null,
}));

const operation = (overrides: Partial<OmOperation> = {}): OmOperation => ({
  id: 'op-1',
  kind: 'uninstall',
  kind_version: 3,
  cluster_id: 'cluster-1',
  status: 'succeeded',
  version: 9,
  params_revision: 1,
  trigger: 'user',
  created_at: '2026-11-27T20:25:00Z',
  started_at: '2026-11-27T20:25:58Z',
  finished_at: '2026-11-27T20:28:58Z',
  residual_hosts: [],
  ...overrides,
});

const report = (): OmUninstallReport => ({
  operation_id: 'op-1',
  cluster_id: 'cluster-1',
  cluster_name: 'rs1',
  confirm_phrase: 'REMOVE rs1 aceafc',
  report_digest: 'digest',
  ready: false,
  drifted: false,
  members: [
    {
      member_id: 'm0',
      host: 'pmm-client-node02',
      packages: ['percona-server-mongodb'],
      readiness: 'ready',
    },
  ],
});

const step = (name: string, status: string) => ({
  id: `s-${name}`,
  target_kind: 'member',
  target_id: 'm0',
  phase: 'host',
  step: name,
  attempt: 1,
  status,
  epoch: 1,
});

const detail = (
  steps: ReturnType<typeof step>[],
  overrides: Partial<OmOperation> = {}
): OmGetOperationResponse => ({ operation: operation(overrides), steps });

const open = (search = '?tab=uninstalls') =>
  render(
    <MemoryRouter initialEntries={[`/operations/automations${search}`]}>
      <AutomationsPage />
    </MemoryRouter>
  );

beforeEach(() => {
  useOmBootstrapRuns.mockReturnValue({
    data: [],
    isLoading: false,
    error: null,
  });
  useOmOperations.mockReturnValue({
    data: [operation()],
    isLoading: false,
    error: null,
  });
  useUninstallReport.mockReturnValue({ data: report() });
  useOmOperation.mockReturnValue({
    data: detail([
      step('uninstall_stop', 'succeeded'),
      step('uninstall_remove', 'succeeded'),
      step('uninstall_forget', 'running'),
    ]),
  });
});

describe('AutomationsPage uninstalls', () => {
  it('lists a confirmed uninstall with its status, cluster and node count', () => {
    open();
    const row = screen.getByText('rs1').closest('tr') as HTMLElement;
    expect(within(row).getByText('Succeeded')).toBeInTheDocument();
    expect(within(row).getByText('1')).toBeInTheDocument();
  });

  it('leaves out operations of another kind and uninstalls that never started', () => {
    useOmOperations.mockReturnValue({
      data: [
        operation({ id: 'op-1' }),
        operation({ id: 'op-2', kind: 'install' }),
        operation({
          id: 'op-3',
          status: 'cancelled',
          started_at: null,
          finished_at: null,
        }),
      ],
      isLoading: false,
      error: null,
    });
    open();
    expect(screen.getAllByText('rs1')).toHaveLength(1);
  });

  it('opens the operation named by ?expand and shows each step per node', () => {
    useOmOperations.mockReturnValue({
      data: [operation({ status: 'running', finished_at: null })],
      isLoading: false,
      error: null,
    });
    useOmOperation.mockReturnValue({
      data: detail(
        [
          step('uninstall_stop', 'succeeded'),
          step('uninstall_remove', 'succeeded'),
          step('uninstall_forget', 'running'),
        ],
        { status: 'running', finished_at: null }
      ),
    });
    open('?tab=uninstalls&expand=op-1');
    expect(screen.getByText('pmm-client-node02')).toBeInTheDocument();
    expect(screen.getByText('Stopping mongod')).toBeInTheDocument();
    expect(
      screen.getByText('Removing packages, files, logs and data')
    ).toBeInTheDocument();
    expect(
      screen.getByText('Forgetting the service in the inventory')
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText('Forgetting the service in the inventory: Running')
    ).toBeInTheDocument();
    expect(screen.getByText('2 of 3 steps done.')).toBeInTheDocument();
  });

  it('shows every planned step from the start, the undispatched ones as Pending', () => {
    useOmOperations.mockReturnValue({
      data: [operation({ status: 'running', finished_at: null })],
      isLoading: false,
      error: null,
    });
    useOmOperation.mockReturnValue({
      data: detail([step('uninstall_stop', 'running')], {
        status: 'running',
        finished_at: null,
      }),
    });
    open('?tab=uninstalls&expand=op-1');
    expect(
      screen.getByLabelText('Stopping mongod: Running')
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText('Removing packages, files, logs and data: Pending')
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText('Forgetting the service in the inventory: Pending')
    ).toBeInTheDocument();
    expect(screen.getByText('0 of 3 steps done.')).toBeInTheDocument();
  });

  it('says why an uninstall failed, and marks the failed step', () => {
    useOmOperations.mockReturnValue({
      data: [operation({ status: 'failed', error_code: 'step_failed' })],
      isLoading: false,
      error: null,
    });
    useOmOperation.mockReturnValue({
      data: detail(
        [
          step('uninstall_stop', 'succeeded'),
          step('uninstall_remove', 'failed'),
          step('uninstall_forget', 'planned'),
        ],
        { status: 'failed', error_code: 'step_failed' }
      ),
    });
    open('?tab=uninstalls&expand=op-1');
    expect(
      screen.getByText(/The uninstall did not finish/)
    ).toBeInTheDocument();
    expect(screen.getByText('step_failed')).toBeInTheDocument();
    expect(
      screen.getByLabelText('Removing packages, files, logs and data: Failed')
    ).toBeInTheDocument();
  });

  it('explains an empty history', () => {
    useOmOperations.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    });
    open();
    expect(screen.getByText('No uninstalls yet')).toBeInTheDocument();
  });

  it('reports a failed load instead of an empty table', () => {
    useOmOperations.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('boom'),
    });
    open();
    expect(screen.getByText('Could not load uninstalls')).toBeInTheDocument();
    expect(screen.queryByText('No uninstalls yet')).toBeNull();
  });
});
