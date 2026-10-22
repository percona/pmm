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

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SnackbarProvider } from 'notistack';
import { CollectPane } from '../src/CollectPane';
import type { AtwSnippetSummary } from '../src/types';

vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAuth: () => ({ isAdmin: true, canMutate: true }),
}));

import { apiClient } from '@pmm-extensions/api';
const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

const SLOW_LOG: AtwSnippetSummary = {
  name: 'ops/slow-log.sh',
  title: 'Slow log',
  description: 'Collects the slow query log.',
  sudo: 'never',
};

const DISK_USAGE: AtwSnippetSummary = {
  name: 'ops/disk-usage.sh',
  title: 'Disk usage',
  description: 'Reports disk usage.',
  sudo: 'never',
};

const EXECUTOR_HOST_FIELD = {
  type: 'host',
  name: 'executor_host',
  label: 'Execution Host',
  required: true,
  default: 'sep-mysql',
};

const HOSTS = [{ id: 'sep-mysql', name: 'sep-mysql', address: '172.28.9.40' }];

/** Per-snippet fields keyed by filename, as the merged-schema route reports them. */
type SnippetFields = Record<string, unknown[]>;

function mockApis(fields: SnippetFields) {
  mockedApi.get.mockImplementation((url: string) => {
    if (url === '/sep/hosts/') {
      return Promise.resolve({ data: HOSTS });
    }
    if (url.includes('/execution-schema/')) {
      return Promise.resolve({
        data: {
          shared: [EXECUTOR_HOST_FIELD],
          per_snippet: Object.entries(fields).map(
            ([snippet_filename, snippetFields]) => ({
              snippet_filename,
              fields: snippetFields,
            })
          ),
        },
      });
    }
    return Promise.resolve({ data: [] });
  });
}

/** Open the pane with `snippets` already selected, as "Run again" does. */
function renderWithSelection(
  snippets: AtwSnippetSummary[],
  fields: SnippetFields,
  onDispatched?: () => void
) {
  mockApis(fields);
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <SnackbarProvider>
        <CollectPane
          incidentId="inc-1"
          onDispatched={onDispatched}
          rerunRequest={{
            nonce: 1,
            snippetFilename: snippets[0].name,
            remembered: { snippets, values: {} },
          }}
        />
      </SnackbarProvider>
    </QueryClientProvider>
  );
}

/** A batch response launching one task per requested snippet. */
function launched(...filenames: string[]) {
  return {
    data: {
      items: filenames.map((snippet_filename, index) => ({
        snippet_filename,
        task_history_id: 100 + index,
        error: null,
      })),
    },
  };
}

const MINUTES_FIELD = {
  type: 'integer',
  name: 'minutes',
  label: 'Minutes',
  default: 30,
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe('CollectPane script sections', () => {
  it('opens a script section collapsed, naming the values it holds', async () => {
    renderWithSelection([SLOW_LOG], { 'ops/slow-log.sh': [MINUTES_FIELD] });

    const header = await screen.findByRole('button', {
      name: /Slow log/,
      expanded: false,
    });
    expect(within(header).getByText('Minutes: 30')).toBeVisible();
    // A collapsed shell mounts nothing, which is the point: two selected
    // scripts used to open two screens' worth of fields.
    expect(
      screen.queryByTestId('text-input-overrides.snip0.minutes')
    ).toBeNull();
  });

  it('opens a section the reader has to fill in', async () => {
    renderWithSelection([SLOW_LOG], {
      'ops/slow-log.sh': [
        { type: 'string', name: 'pattern', label: 'Pattern', required: true },
      ],
    });

    // Collapsed, the required field would neither validate nor be visible.
    expect(
      await screen.findByTestId('text-input-overrides.snip0.pattern')
    ).toBeVisible();
  });

  it('opens a section holding a one-of group', async () => {
    // The group's own slot is what unregisters the branches the reader did not
    // pick, and a collapsed shell never mounts it — every branch's seeded
    // default would ship at once.
    renderWithSelection([SLOW_LOG], {
      'ops/slow-log.sh': [
        {
          type: 'one_of',
          name: 'source',
          label: 'Source',
          discriminator: 'source_mode',
          branches: [
            {
              value: 'file',
              label: 'File',
              fields: [{ type: 'string', name: 'path', label: 'Path' }],
            },
            {
              value: 'table',
              label: 'Table',
              fields: [{ type: 'string', name: 'table', label: 'Table' }],
            },
          ],
        },
      ],
    });

    expect(
      await screen.findByTestId('one-of-overrides.snip0.source')
    ).toBeVisible();
  });

  it('opens a section holding a gated field', async () => {
    // A gate that would make the field required is evaluated by the field's
    // own slot, so it decides nothing while the section is closed.
    renderWithSelection([SLOW_LOG], {
      'ops/slow-log.sh': [
        {
          type: 'string',
          name: 'pattern',
          label: 'Pattern',
          default: 'SELECT',
          requires: [{ when: { truthy: 'verbose' } }],
        },
      ],
    });

    expect(
      await screen.findByTestId('text-input-overrides.snip0.pattern')
    ).toBeVisible();
  });

  it('keeps a required field with a default out of the way', async () => {
    renderWithSelection([SLOW_LOG], {
      'ops/slow-log.sh': [
        {
          type: 'string',
          name: 'pattern',
          label: 'Pattern',
          required: true,
          default: 'SELECT',
        },
      ],
    });

    const header = await screen.findByRole('button', {
      name: /Slow log/,
      expanded: false,
    });
    expect(within(header).getByText('Pattern: SELECT')).toBeVisible();
  });
});

describe('CollectPane run feedback', () => {
  it('reports the run where the button was pressed, and collapses the form', async () => {
    const onDispatched = vi.fn();
    let resolvePost: (value: unknown) => void = () => {};
    mockedApi.post.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolvePost = resolve;
        })
    );
    renderWithSelection(
      [SLOW_LOG, DISK_USAGE],
      {
        'ops/slow-log.sh': [MINUTES_FIELD],
        'ops/disk-usage.sh': [MINUTES_FIELD],
      },
      onDispatched
    );

    const run = await screen.findByRole('button', { name: 'Execute batch' });
    await userEvent.click(run);

    // The click itself has to change something: the button says what it is
    // doing rather than leaving the reader to look at the other column.
    expect(
      await screen.findByRole('button', { name: /Starting/ })
    ).toBeDisabled();

    resolvePost(launched('ops/slow-log.sh', 'ops/disk-usage.sh'));

    const confirmation = await screen.findByTestId('atw-collect-dispatched');
    expect(confirmation).toHaveTextContent('Started 2 scripts');
    expect(screen.queryByRole('button', { name: 'Execute batch' })).toBeNull();
    expect(onDispatched).toHaveBeenCalledWith(
      [SLOW_LOG, DISK_USAGE],
      expect.any(Object),
      expect.any(Object),
      'collect'
    );
  });

  it('counts one script as one', async () => {
    mockedApi.post.mockResolvedValue(launched('ops/slow-log.sh'));
    renderWithSelection([SLOW_LOG], { 'ops/slow-log.sh': [MINUTES_FIELD] });

    await userEvent.click(
      await screen.findByRole('button', { name: 'Execute batch' })
    );

    expect(
      await screen.findByTestId('atw-collect-dispatched')
    ).toHaveTextContent('Started 1 script.');
  });

  it('brings the form back filled in when the reader wants another run', async () => {
    mockedApi.post.mockResolvedValue(launched('ops/slow-log.sh'));
    renderWithSelection([SLOW_LOG], { 'ops/slow-log.sh': [MINUTES_FIELD] });

    await userEvent.click(
      await screen.findByRole('button', {
        name: /Slow log/,
        expanded: false,
      })
    );
    const minutes = await screen.findByTestId(
      'text-input-overrides.snip0.minutes'
    );
    await userEvent.clear(minutes);
    await userEvent.type(minutes, '90');

    await userEvent.click(
      screen.getByRole('button', { name: 'Execute batch' })
    );
    await screen.findByTestId('atw-collect-dispatched');

    await userEvent.click(
      screen.getByRole('button', { name: /Change parameters and run again/ })
    );

    // The collapse unmounts the form, so a value that did not round-trip
    // through `defaultValues` would come back as the schema's 30.
    const header = await screen.findByRole('button', {
      name: /Slow log/,
      expanded: false,
    });
    expect(within(header).getByText('Minutes: 90')).toBeVisible();
  });

  it('does not pin a late result onto a selection that moved on', async () => {
    let resolvePost: (value: unknown) => void = () => {};
    mockedApi.post.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolvePost = resolve;
        })
    );
    mockApis({
      'ops/slow-log.sh': [MINUTES_FIELD],
      'ops/disk-usage.sh': [MINUTES_FIELD],
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const paneFor = (nonce: number, snippet: AtwSnippetSummary) => (
      <QueryClientProvider client={queryClient}>
        <SnackbarProvider>
          <CollectPane
            incidentId="inc-1"
            rerunRequest={{
              nonce,
              snippetFilename: snippet.name,
              remembered: { snippets: [snippet], values: {} },
            }}
          />
        </SnackbarProvider>
      </QueryClientProvider>
    );
    const { rerender } = render(paneFor(1, SLOW_LOG));

    await userEvent.click(
      await screen.findByRole('button', { name: 'Execute batch' })
    );

    // The reader reopens the form for another script while the first batch is
    // still in flight — the Results pane's "Edit parameters and run again".
    rerender(paneFor(2, DISK_USAGE));
    await screen.findByRole('button', { name: /Disk usage/, expanded: false });

    resolvePost(launched('ops/slow-log.sh'));

    // The batch that finished is not the one on screen, so its confirmation
    // would hide a form the reader is in the middle of filling.
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Execute batch' })
      ).toBeEnabled()
    );
    expect(screen.queryByTestId('atw-collect-dispatched')).toBeNull();
  });

  it('keeps the form open when the batch launched nothing', async () => {
    mockedApi.post.mockResolvedValue({
      data: {
        items: [
          {
            snippet_filename: 'ops/slow-log.sh',
            task_history_id: null,
            error: 'minutes: must be positive',
          },
        ],
      },
    });
    renderWithSelection([SLOW_LOG], { 'ops/slow-log.sh': [MINUTES_FIELD] });

    await userEvent.click(
      await screen.findByRole('button', { name: 'Execute batch' })
    );

    await screen.findByText(/must be positive/);
    expect(screen.queryByTestId('atw-collect-dispatched')).toBeNull();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Execute batch' })
      ).toBeEnabled()
    );
  });
});
