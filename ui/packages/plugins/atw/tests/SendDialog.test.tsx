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

import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { SendDialog } from '../src/SendDialog';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
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

const CASE_MATCHES = [
  { reference: 'CS0001', title: 'Slow queries on the primary' },
  { reference: 'CS0002', title: 'Replica lag after failover' },
];

interface MockApiOptions {
  /** What `atw_config` reports for `case_search_available`. */
  caseSearchAvailable?: boolean;
  /** What the case-search route answers with. */
  search?: { available: boolean; matches: typeof CASE_MATCHES };
  /** Answer `atw_config` without the field, as a server predating it would. */
  omitAvailabilityField?: boolean;
}

/** Route each mocked GET by path, so config and case search answer separately. */
function mockApis({
  caseSearchAvailable = true,
  search = { available: true, matches: CASE_MATCHES },
  omitAvailabilityField = false,
}: MockApiOptions = {}): void {
  mockedApi.get.mockImplementation((url: string) => {
    if (url.startsWith('/apps/atw/case-search/')) {
      return Promise.resolve({ data: search });
    }
    if (url.startsWith('/apps/atw/config/')) {
      return Promise.resolve({
        data: omitAvailabilityField
          ? { send_disabled_reasons: [] }
          : {
              send_disabled_reasons: [],
              case_search_available: caseSearchAvailable,
            },
      });
    }
    return Promise.resolve({ data: sendLog() });
  });
}

/** The term of every case-search request issued so far, in order. */
function caseSearchTerms(): string[] {
  // Indexed rather than destructured with a tuple annotation: PMM typechecks
  // these tests where SEP does not, and mock.calls is not a fixed-length tuple.
  return mockedApi.get.mock.calls
    .filter((call) => String(call[0]).startsWith('/apps/atw/case-search/'))
    .map((call) => (call[1] as { params: { term: string } }).params.term);
}

function caseRefField(): HTMLInputElement {
  return screen.getByLabelText(/Support case reference/i) as HTMLInputElement;
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

  it('states that the bundle carries both output files and logs', () => {
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    expect(
      screen.getByText(/output files and captured logs are bundled/i)
    ).toBeTruthy();
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

describe('SendDialog case search', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('offers the cases the provider matched for the typed term', async () => {
    mockApis();
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS00');

    expect(
      await screen.findByRole('option', { name: /CS0001/ }, { timeout: 3000 })
    ).toBeTruthy();
    expect(screen.getByRole('option', { name: /CS0002/ })).toBeTruthy();
  });

  it('shows the case title beneath the reference', async () => {
    mockApis();
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS00');

    const option = await screen.findByRole(
      'option',
      { name: /CS0001/ },
      { timeout: 3000 }
    );
    expect(option).toHaveTextContent('Slow queries on the primary');

    const reference = within(option).getByText('CS0001');
    const title = within(option).getByText('Slow queries on the primary');
    expect(title.parentElement).toBe(reference.parentElement);

    const stack = getComputedStyle(reference.parentElement as HTMLElement);
    expect(stack.display).toBe('flex');
    expect(stack.flexDirection).toBe('column');
  });

  it('puts the picked case reference in the field', async () => {
    mockApis();
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS00');
    await userEvent.click(
      await screen.findByRole('option', { name: /CS0002/ }, { timeout: 3000 })
    );

    expect(caseRefField().value).toBe('CS0002');
  });

  it('sends a reference the search never matched, exactly as typed', async () => {
    mockApis({ search: { available: true, matches: [] } });
    mockedApi.post.mockResolvedValue({ data: sendLog({ case_ref: 'CS9999' }) });
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS9999');
    await userEvent.click(screen.getByRole('button', { name: 'Send' }));

    await waitFor(() => expect(mockedApi.post).toHaveBeenCalled());
    expect(mockedApi.post.mock.calls[0][1].case_ref).toBe('CS9999');
  });

  it('sends the prefilled reference without the search ever answering', async () => {
    // Every GET hangs, so nothing the search could return is in hand.
    mockedApi.get.mockImplementation(() => new Promise(() => {}));
    mockedApi.post.mockResolvedValue({ data: sendLog() });
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        defaultCaseRef="CS0042"
        onClose={() => {}}
      />
    );

    const send = screen.getByRole('button', { name: 'Send' });
    expect(send).not.toBeDisabled();
    await userEvent.click(send);

    await waitFor(() => expect(mockedApi.post).toHaveBeenCalled());
    expect(mockedApi.post.mock.calls[0][1].case_ref).toBe('CS0042');
  });

  it('offers no options and no empty state when the search is unavailable', async () => {
    mockApis({ search: { available: false, matches: [] } });
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS00');
    await waitFor(() => expect(caseSearchTerms()).toHaveLength(1), {
      timeout: 3000,
    });

    expect(screen.queryAllByRole('option')).toHaveLength(0);
    expect(screen.queryByText(/No matching case/i)).toBeNull();
    expect(caseRefField()).not.toBeDisabled();
  });

  it('distinguishes a search that matched nothing from one that could not run', async () => {
    mockApis({ search: { available: true, matches: [] } });
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS00');

    expect(
      await screen.findByText(/No matching case/i, {}, { timeout: 3000 })
    ).toBeTruthy();
  });

  it('issues one search for a term typed in one burst', async () => {
    mockApis();
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS001234');

    await waitFor(() => expect(caseSearchTerms()).toHaveLength(1), {
      timeout: 3000,
    });
    expect(caseSearchTerms()[0]).toBe('CS001234');
  });

  it('issues no search at all where the deployment declares none', async () => {
    mockApis({ caseSearchAvailable: false });
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS00');
    await new Promise((resolve) => setTimeout(resolve, 500));

    expect(caseSearchTerms()).toHaveLength(0);
  });

  it('treats a config response without the field as no search available', async () => {
    // A server predating this field omits it; the field must stay a plain input
    // rather than issuing searches the deployment cannot serve.
    mockApis({ omitAvailabilityField: true });
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    await userEvent.type(caseRefField(), 'CS00');
    await new Promise((resolve) => setTimeout(resolve, 500));

    expect(caseSearchTerms()).toHaveLength(0);
    expect(caseRefField().value).toBe('CS00');
  });

  it('drops the offered options the instant the term is typed on', async () => {
    mockApis();
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    const field = caseRefField();
    await userEvent.type(field, 'CS00');
    await screen.findByRole('option', { name: /CS0001/ }, { timeout: 3000 });

    // Synchronous, so the assertion lands inside the debounce window while the
    // query key — and therefore the data in hand — is still the previous term's.
    fireEvent.change(field, { target: { value: 'CS001' } });

    expect(screen.queryAllByRole('option')).toHaveLength(0);
  });

  it('drops the offered options the instant the field is cleared', async () => {
    mockApis();
    renderDialog(
      <SendDialog
        open
        incidentId="inc-1"
        executions={EXECUTIONS}
        onClose={() => {}}
      />
    );

    const field = caseRefField();
    await userEvent.type(field, 'CS00');
    await screen.findByRole('option', { name: /CS0001/ }, { timeout: 3000 });

    fireEvent.change(field, { target: { value: '' } });

    expect(screen.queryAllByRole('option')).toHaveLength(0);
  });
});
