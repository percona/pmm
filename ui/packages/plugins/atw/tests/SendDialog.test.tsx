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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { SendDialog } from '../src/SendDialog';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn(), post: vi.fn() },
}));

import { apiClient } from '@sep/api';

const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

const EXECUTIONS = [
  { id: 'exec-1', task_history_id: 7, snippet_filename: 'diag/slow-query.sh' },
  { id: 'exec-2', task_history_id: 8, snippet_filename: 'diag/dmesg.sh' },
];

function sendLog(overrides: Record<string, unknown> = {}) {
  return {
    id: 'job-1',
    incident_id: 'inc-1',
    case_ref: 'CS0042',
    requested_by: 'alice',
    status: 'pending',
    started_at: null,
    finished_at: null,
    created_at: '2026-07-24T10:00:00Z',
    detail: {},
    ...overrides,
  };
}

function renderDialog(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

describe('SendDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('prefills the case reference from the incident', () => {
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0001"
        onClose={() => {}}
      />
    );

    const field = screen.getByLabelText(
      /Support case reference/i
    ) as HTMLInputElement;
    expect(field.value).toBe('CS0001');
  });

  it('lists every selected execution', () => {
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    expect(screen.getByText('diag/slow-query.sh')).toBeTruthy();
    expect(screen.getByText('diag/dmesg.sh')).toBeTruthy();
    expect(screen.getByText(/2 executions selected/i)).toBeTruthy();
  });

  it('keeps Send disabled until a case reference is entered', () => {
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    const send = screen.getByRole('button', {
      name: 'Send',
    }) as HTMLButtonElement;
    expect(send.disabled).toBe(true);

    fireEvent.change(screen.getByLabelText(/Support case reference/i), {
      target: { value: 'CS0042' },
    });

    expect(
      (screen.getByRole('button', { name: 'Send' }) as HTMLButtonElement)
        .disabled
    ).toBe(false);
  });

  it('disables Send when the selection is empty', () => {
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={[]}
        defaultCaseRef="CS0042"
        onClose={() => {}}
      />
    );

    expect(
      (screen.getByRole('button', { name: 'Send' }) as HTMLButtonElement)
        .disabled
    ).toBe(true);
    expect(
      screen.getByText(/None of the selected executions still exist/i)
    ).toBeTruthy();
  });

  it('posts the selection, then follows the job to success', async () => {
    mockedApi.post.mockResolvedValue({ data: sendLog() });
    mockedApi.get.mockResolvedValue({
      data: sendLog({
        status: 'success',
        finished_at: '2026-07-24T10:01:00Z',
        detail: { upload_reference: 'att-9' },
      }),
    });

    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0042"
        onClose={() => {}}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Send' }));

    await waitFor(() => {
      expect(screen.getByText(/Diagnostics sent/i)).toBeTruthy();
    });
    expect(mockedApi.post).toHaveBeenCalledWith(
      '/apps/atw/incidents/inc-1/send-jobs/',
      {
        case_ref: 'CS0042',
        execution_ids: ['exec-1', 'exec-2'],
      }
    );
    expect(screen.getByText(/att-9/)).toBeTruthy();
  });

  it('surfaces the recorded error and offers Send again on failure', async () => {
    mockedApi.post.mockResolvedValue({ data: sendLog() });
    mockedApi.get.mockResolvedValue({
      data: sendLog({
        status: 'failed',
        finished_at: '2026-07-24T10:01:00Z',
        detail: {
          error:
            'Bundle exceeded the configured 30 MiB limit while being built.',
        },
      }),
    });

    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0042"
        onClose={() => {}}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Send' }));

    await waitFor(() => {
      expect(screen.getByText(/30 MiB limit/)).toBeTruthy();
    });
    expect(screen.getByRole('button', { name: 'Send again' })).toBeTruthy();
  });

  it('shows progress while the job is still running', async () => {
    mockedApi.post.mockResolvedValue({ data: sendLog() });
    mockedApi.get.mockResolvedValue({ data: sendLog({ status: 'running' }) });

    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0042"
        onClose={() => {}}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Send' }));

    await waitFor(() => {
      expect(screen.getByText(/Bundling and uploading/i)).toBeTruthy();
    });
    expect(
      (screen.getByRole('button', { name: 'Send' }) as HTMLButtonElement)
        .disabled
    ).toBe(true);
  });

  it('refuses a second send between the POST landing and the first poll', async () => {
    mockedApi.post.mockResolvedValue({ data: sendLog() });
    mockedApi.get.mockReturnValue(new Promise(() => {}));

    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0042"
        onClose={() => {}}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Send' }));

    await waitFor(() => {
      expect(
        (screen.getByRole('button', { name: 'Send' }) as HTMLButtonElement)
          .disabled
      ).toBe(true);
    });
    fireEvent.click(screen.getByRole('button', { name: 'Send' }));

    expect(mockedApi.post).toHaveBeenCalledTimes(1);
  });

  it('surfaces a poll it could not complete and re-enables sending', async () => {
    mockedApi.post.mockResolvedValue({ data: sendLog() });
    mockedApi.get.mockRejectedValue(new Error('job vanished'));

    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0042"
        onClose={() => {}}
      />
    );
    fireEvent.click(screen.getByRole('button', { name: 'Send' }));

    await waitFor(() => {
      expect(screen.getByText(/Lost track of this send/i)).toBeTruthy();
    });
    expect(
      (screen.getByRole('button', { name: 'Send again' }) as HTMLButtonElement)
        .disabled
    ).toBe(false);
  });

  it('reports on close whether a send was started', async () => {
    mockedApi.post.mockResolvedValue({ data: sendLog() });
    mockedApi.get.mockReturnValue(new Promise(() => {}));
    const onClose = vi.fn();

    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0042"
        onClose={onClose}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenLastCalledWith(false);

    fireEvent.click(screen.getByRole('button', { name: 'Send' }));
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Close' })).toBeTruthy();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Close' }));

    expect(onClose).toHaveBeenLastCalledWith(true);
  });
});
