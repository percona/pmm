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

import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { UninstallDialog } from '../src/components/UninstallDialog';
import type { OmGetOperationResponse, OmUninstallReport } from '../src/types';

const {
  useTriggerUninstall,
  useOmOperation,
  useUninstallReport,
  useConfirmOperation,
  useCancelOperation,
  useConfirmUninstallLivePlan,
} = vi.hoisted(() => ({
  useTriggerUninstall: vi.fn(),
  useOmOperation: vi.fn(),
  useUninstallReport: vi.fn(),
  useConfirmOperation: vi.fn(),
  useCancelOperation: vi.fn(),
  useConfirmUninstallLivePlan: vi.fn(),
}));

vi.mock('../src/operationHooks', async () => ({
  ...(await vi.importActual<typeof import('../src/operationHooks')>(
    '../src/operationHooks'
  )),
  useTriggerUninstall,
  useOmOperation,
  useUninstallReport,
  useConfirmOperation,
  useConfirmUninstallLivePlan,
  useCancelOperation,
}));

const triggerMutate = vi.fn();
const confirmMutate = vi.fn();
const cancelMutate = vi.fn();
const adoptMutate = vi.fn();

const operation = (
  overrides: Partial<OmGetOperationResponse['operation']> = {}
): OmGetOperationResponse => ({
  operation: {
    id: 'op-1',
    kind: 'uninstall',
    kind_version: 1,
    cluster_id: 'cluster-1',
    status: 'awaiting_confirmation',
    version: 4,
    params_revision: 1,
    trigger: 'user',
    created_at: '2026-11-27T10:00:00Z',
    residual_hosts: [],
    ...overrides,
  },
  steps: [],
});

const report = (
  overrides: Partial<OmUninstallReport> = {}
): OmUninstallReport => ({
  operation_id: 'op-1',
  cluster_id: 'cluster-1',
  cluster_name: 'rs-prod',
  confirm_phrase: 'REMOVE rs-prod ab12cd',
  report_digest: 'digest-1',
  ready: true,
  drifted: false,
  members: [
    {
      member_id: 'm0',
      host: 'node00',
      packages: ['percona-server-mongodb'],
      data_path: '/var/lib/mongodb',
      readiness: 'ready',
    },
  ],
  ...overrides,
});

beforeEach(() => {
  triggerMutate.mockImplementation((_clusterId, opts) =>
    opts?.onSuccess?.({ operation_id: 'op-1' })
  );
  useTriggerUninstall.mockReturnValue({
    mutate: triggerMutate,
    isError: false,
    error: null,
  });
  useOmOperation.mockReturnValue({ data: operation() });
  useUninstallReport.mockReturnValue({ data: report(), isError: false });
  useConfirmOperation.mockReturnValue({
    mutate: confirmMutate,
    isPending: false,
    isError: false,
    error: null,
  });
  useCancelOperation.mockReturnValue({
    mutate: cancelMutate,
    isPending: false,
  });
  useConfirmUninstallLivePlan.mockReturnValue({
    mutate: adoptMutate,
    isPending: false,
  });
});

afterEach(() => {
  vi.clearAllMocks();
});

const onClose = vi.fn();

const LocationProbe = () => {
  const { pathname, search } = useLocation();
  return <output data-testid="location">{`${pathname}${search}`}</output>;
};

const open = () =>
  render(
    <MemoryRouter initialEntries={['/operations/nodes']}>
      <UninstallDialog
        clusterId="cluster-1"
        clusterName="rs-prod"
        onClose={onClose}
      />
      <LocationProbe />
    </MemoryRouter>
  );

describe('UninstallDialog', () => {
  it('triggers the uninstall once on open', () => {
    open();
    expect(triggerMutate).toHaveBeenCalledTimes(1);
    expect(triggerMutate).toHaveBeenCalledWith('cluster-1', expect.anything());
  });

  it('renders the report: cluster, member host, paths and the confirm phrase', () => {
    open();
    expect(screen.getByText('node00')).toBeInTheDocument();
    expect(screen.getByText(/\/var\/lib\/mongodb/)).toBeInTheDocument();
    expect(screen.getByText('REMOVE rs-prod ab12cd')).toBeInTheDocument();
    expect(
      screen.getByText(/No backup is taken or verified/i)
    ).toBeInTheDocument();
  });

  it('enables Remove only once the exact confirm phrase is typed, then confirms with the version', () => {
    open();
    const remove = screen.getByRole('button', { name: /Remove cluster/i });
    expect(remove).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText('REMOVE rs-prod ab12cd'), {
      target: { value: 'REMOVE rs-prod wrong' },
    });
    expect(remove).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText('REMOVE rs-prod ab12cd'), {
      target: { value: 'REMOVE rs-prod ab12cd' },
    });
    expect(remove).toBeEnabled();

    fireEvent.click(remove);
    expect(confirmMutate).toHaveBeenCalledWith(
      {
        operationId: 'op-1',
        version: 4,
        confirmPhrase: 'REMOVE rs-prod ab12cd',
      },
      expect.anything()
    );
  });

  it('hands over to the Uninstalls tab, expanded on this operation, once the confirm lands', () => {
    confirmMutate.mockImplementationOnce((_variables, opts) =>
      opts?.onSuccess?.()
    );
    open();
    fireEvent.change(screen.getByPlaceholderText('REMOVE rs-prod ab12cd'), {
      target: { value: 'REMOVE rs-prod ab12cd' },
    });
    expect(screen.getByTestId('location')).toHaveTextContent(
      '/operations/nodes'
    );

    fireEvent.click(screen.getByRole('button', { name: /Remove cluster/i }));

    expect(onClose).toHaveBeenCalled();
    expect(screen.getByTestId('location')).toHaveTextContent(
      '/operations/automations?tab=uninstalls&expand=op-1'
    );
  });

  it('stays on the dialog when the confirm is refused', () => {
    open();
    fireEvent.change(screen.getByPlaceholderText('REMOVE rs-prod ab12cd'), {
      target: { value: 'REMOVE rs-prod ab12cd' },
    });
    fireEvent.click(screen.getByRole('button', { name: /Remove cluster/i }));

    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByTestId('location')).toHaveTextContent(
      '/operations/nodes'
    );
  });

  it('keeps Remove disabled and shows the refusal when a member is not ready', () => {
    useUninstallReport.mockReturnValue({
      data: report({
        ready: false,
        members: [
          {
            member_id: 'm0',
            host: 'node00',
            packages: [],
            readiness: 'marker_mismatch',
          },
        ],
      }),
      isError: false,
    });
    useOmOperation.mockReturnValue({
      data: operation({ status: 'failed', error_code: 'marker_mismatch' }),
    });
    open();
    expect(screen.getByText(/cannot be removed yet/i)).toBeInTheDocument();
    // The phrase input is not offered when the report is not ready, so Remove stays off.
    expect(
      screen.getByRole('button', { name: /Remove cluster/i })
    ).toBeDisabled();
    expect(
      screen.queryByPlaceholderText('REMOVE rs-prod ab12cd')
    ).not.toBeInTheDocument();
  });

  it('offers adopting the live packages when readiness blocked on drift', () => {
    useUninstallReport.mockReturnValue({
      data: report({
        ready: false,
        drifted: true,
        members: [
          {
            member_id: 'm0',
            host: 'node00',
            packages: ['percona-server-mongodb'],
            readiness: 'drift',
          },
        ],
      }),
      isError: false,
    });
    useOmOperation.mockReturnValue({
      data: operation({ status: 'failed', error_code: 'drift' }),
    });
    open();
    fireEvent.click(
      screen.getByRole('button', { name: /Adopt live packages/i })
    );
    expect(adoptMutate).toHaveBeenCalledWith({
      operationId: 'op-1',
      version: 4,
    });
  });

  it('cancels an operation still awaiting confirmation when dismissed', () => {
    open();
    fireEvent.click(screen.getByRole('button', { name: /^Cancel$/i }));
    expect(cancelMutate).toHaveBeenCalledWith(
      { operationId: 'op-1', version: 4 },
      expect.anything()
    );
  });

  it('does not cancel a running teardown on dismiss', () => {
    useOmOperation.mockReturnValue({ data: operation({ status: 'running' }) });
    open();
    fireEvent.click(screen.getByRole('button', { name: /^Cancel$/i }));
    expect(cancelMutate).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });
});
