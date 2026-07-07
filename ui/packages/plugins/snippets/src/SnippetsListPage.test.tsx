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
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '@sep/api';
import { SnippetsListPage } from './SnippetsListPage';
import {
  useSnippets,
  useApproveSnippet,
  useRemoveSnippetApproval,
  useBatchApproveSnippets,
  useSnippetsCapabilities,
  useRefreshSnippets,
} from './hooks';

vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock('./hooks', () => ({
  useSnippets: vi.fn(),
  useApproveSnippet: vi.fn(),
  useRemoveSnippetApproval: vi.fn(),
  useBatchApproveSnippets: vi.fn(),
  useSnippetsCapabilities: vi.fn(),
  useRefreshSnippets: vi.fn(),
}));

const mockUseSnippets = vi.mocked(useSnippets);
const mockUseApproveSnippet = vi.mocked(useApproveSnippet);
const mockUseRemoveSnippetApproval = vi.mocked(useRemoveSnippetApproval);
const mockUseBatchApproveSnippets = vi.mocked(useBatchApproveSnippets);
const mockUseSnippetsCapabilities = vi.mocked(useSnippetsCapabilities);
const mockUseRefreshSnippets = vi.mocked(useRefreshSnippets);

const unapprovedSnippet = {
  filename: 'check.sh',
  title: 'Check',
  description: 'A check script',
  service_type: 'mysql',
  size: 100,
  md5_digest: 'abc123',
  is_approved: false,
  approved_at: null,
  updated_by: null,
  reason: 'New snippet',
  requires_sudo: false,
  sudo_optional: false,
  sudo_default: false,
  interpreter: 'bash',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: null,
};

const approvedSnippet = {
  ...unapprovedSnippet,
  filename: 'approved.sh',
  is_approved: true,
};

describe('SnippetsListPage — ApproveButton', () => {
  const approveMutate = vi.fn();
  const removeMutate = vi.fn();
  const batchMutate = vi.fn();
  const refreshMutate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseApproveSnippet.mockReturnValue({
      mutate: approveMutate,
      isPending: false,
    } as unknown as ReturnType<typeof useApproveSnippet>);
    mockUseRemoveSnippetApproval.mockReturnValue({
      mutate: removeMutate,
      isPending: false,
    } as unknown as ReturnType<typeof useRemoveSnippetApproval>);
    mockUseBatchApproveSnippets.mockReturnValue({
      mutate: batchMutate,
      isPending: false,
    } as unknown as ReturnType<typeof useBatchApproveSnippets>);
    // Default: manual sync disabled, no pending refresh.
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: false },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: refreshMutate,
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);
  });

  describe('per-row approval requires confirmation', () => {
    beforeEach(() => {
      mockUseSnippets.mockReturnValue({
        data: [unapprovedSnippet],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useSnippets>);
    });

    it('disables approval until the snippet has been downloaded', () => {
      render(<SnippetsListPage isAdmin />);

      expect(screen.getByRole('button', { name: /approve/i })).toBeDisabled();
    });

    it('download action enables the approval confirmation flow', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(screen.getByRole('link', { name: /download/i }));
      fireEvent.click(screen.getByRole('button', { name: /approve/i }));

      expect(screen.getByRole('dialog')).toBeInTheDocument();
      expect(approveMutate).not.toHaveBeenCalled();
    });

    it('dialog warns about approving without inspection', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(screen.getByRole('link', { name: /download/i }));
      fireEvent.click(screen.getByRole('button', { name: /approve/i }));

      const dialog = screen.getByRole('dialog');
      expect(dialog).toHaveTextContent(/inspect/i);
    });

    it('dialog includes the snippet filename', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(screen.getByRole('link', { name: /download/i }));
      fireEvent.click(screen.getByRole('button', { name: /approve/i }));

      expect(screen.getByRole('dialog')).toHaveTextContent('check.sh');
    });

    it('Cancel closes the dialog without approving', async () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(screen.getByRole('link', { name: /download/i }));
      fireEvent.click(screen.getByRole('button', { name: /approve/i }));
      expect(screen.getByRole('dialog')).toBeInTheDocument();

      fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

      await waitFor(() => {
        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
      });
      expect(approveMutate).not.toHaveBeenCalled();
    });

    it('Confirm in dialog calls approve mutation', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(screen.getByRole('link', { name: /download/i }));
      fireEvent.click(screen.getByRole('button', { name: /approve/i }));
      fireEvent.click(screen.getByRole('button', { name: /^approve$/i }));

      expect(approveMutate).toHaveBeenCalledOnce();
    });
  });

  describe('Remove approval button needs no confirmation', () => {
    beforeEach(() => {
      mockUseSnippets.mockReturnValue({
        data: [approvedSnippet],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useSnippets>);
    });

    it('fires remove mutation immediately without a dialog', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(screen.getByRole('button', { name: /remove/i }));

      expect(removeMutate).toHaveBeenCalledOnce();
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
  });

  describe('batch approval requires confirmation', () => {
    beforeEach(() => {
      mockUseSnippets.mockReturnValue({
        data: [unapprovedSnippet, approvedSnippet],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useSnippets>);
    });

    it('opens a confirmation dialog instead of batch approving immediately', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(
        screen.getByRole('checkbox', { name: /select check\.sh/i })
      );
      fireEvent.click(screen.getByRole('button', { name: /batch approve/i }));

      expect(screen.getByRole('dialog')).toHaveTextContent(
        /WITHOUT being downloaded first/i
      );
      expect(batchMutate).not.toHaveBeenCalled();
    });

    it('confirming the batch dialog calls the batch mutation', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(
        screen.getByRole('checkbox', { name: /select check\.sh/i })
      );
      fireEvent.click(screen.getByRole('button', { name: /batch approve/i }));
      fireEvent.click(
        screen.getByRole('button', { name: /approve selected/i })
      );

      expect(batchMutate).toHaveBeenCalledWith(
        { filenames: ['check.sh'] },
        expect.any(Object)
      );
    });

    it('select all ignores already-approved snippets', () => {
      render(<SnippetsListPage isAdmin />);

      fireEvent.click(
        screen.getByRole('checkbox', { name: /select all snippets/i })
      );

      expect(
        screen.getByRole('checkbox', { name: /select check\.sh/i })
      ).toBeChecked();
      expect(
        screen.getByRole('checkbox', { name: /select approved\.sh/i })
      ).not.toBeChecked();
      expect(
        screen.getByRole('checkbox', { name: /select approved\.sh/i })
      ).toBeDisabled();
      expect(
        screen.getByRole('button', { name: /batch approve \(1\)/i })
      ).toBeInTheDocument();
    });
  });
});

describe('SnippetsListPage — RefreshButton', () => {
  const refreshMutate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseSnippets.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippets>);
    mockUseApproveSnippet.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useApproveSnippet>);
    mockUseRemoveSnippetApproval.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useRemoveSnippetApproval>);
    mockUseBatchApproveSnippets.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useBatchApproveSnippets>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: refreshMutate,
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);
  });

  it('hides refresh button when manual_sync_enabled is false (admin)', () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: false },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);

    render(<SnippetsListPage isAdmin />);

    expect(
      screen.queryByRole('button', { name: /refresh snippets/i })
    ).not.toBeInTheDocument();
  });

  it('hides refresh button when user is not admin (sync enabled)', () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);

    render(<SnippetsListPage isAdmin={false} />);

    expect(
      screen.queryByRole('button', { name: /refresh snippets/i })
    ).not.toBeInTheDocument();
  });

  it('shows refresh button when isAdmin and manual_sync_enabled', () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);

    render(<SnippetsListPage isAdmin />);

    expect(
      screen.getByRole('button', { name: /refresh snippets/i })
    ).toBeInTheDocument();
  });

  it('click opens confirmation dialog with correct text', () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);

    render(<SnippetsListPage isAdmin />);
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('dialog')).toHaveTextContent(
      /Are you sure you want to refresh the saved snippets now\?/i
    );
    expect(refreshMutate).not.toHaveBeenCalled();
  });

  it('Cancel closes dialog without calling mutation', async () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);

    render(<SnippetsListPage isAdmin />);
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(refreshMutate).not.toHaveBeenCalled();
  });

  it('Confirm calls refresh mutation', () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);

    render(<SnippetsListPage isAdmin />);
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /^refresh$/i }));

    expect(refreshMutate).toHaveBeenCalledOnce();
  });

  it('disables refresh button while mutation is pending', () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: refreshMutate,
      isPending: true,
    } as unknown as ReturnType<typeof useRefreshSnippets>);

    render(<SnippetsListPage isAdmin />);

    expect(
      screen.getByRole('button', { name: /refresh snippets/i })
    ).toBeDisabled();
  });

  it('shows generic error alert on non-HTTP failure', async () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: vi.fn((_, callbacks) => {
        callbacks?.onError?.(new Error('disk walk failed'));
      }),
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);

    render(<SnippetsListPage isAdmin />);
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /^refresh$/i }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/refresh failed/i);
    });
  });

  it('surfaces backend detail for HTTP error responses', async () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    const httpErr = new ApiError({
      kind: 'http',
      status: 403,
      message: 'Manual snippet sync is disabled in this deployment.',
    });
    mockUseRefreshSnippets.mockReturnValue({
      mutate: vi.fn((_, callbacks) => {
        callbacks?.onError?.(httpErr);
      }),
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);

    render(<SnippetsListPage isAdmin />);
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /^refresh$/i }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        /Manual snippet sync is disabled/i
      );
    });
  });

  it('shows formatted timestamp on success', async () => {
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: vi.fn((_, callbacks) => {
        callbacks?.onSuccess?.({ refreshed_at: '2026-05-07T12:34:56Z' });
      }),
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);

    render(<SnippetsListPage isAdmin />);
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /^refresh$/i }));

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(
        /Snippets refreshed at/i
      );
    });
  });

  it('clears downloaded set after successful refresh', async () => {
    mockUseSnippets.mockReturnValue({
      data: [unapprovedSnippet],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippets>);
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: vi.fn((_, callbacks) => {
        callbacks?.onSuccess?.({ refreshed_at: '2026-05-07T12:34:56Z' });
      }),
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);

    render(<SnippetsListPage isAdmin />);

    // Download enables the Approve button.
    fireEvent.click(screen.getByRole('link', { name: /download/i }));
    expect(screen.getByRole('button', { name: /approve/i })).not.toBeDisabled();

    // Trigger refresh.
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /^refresh$/i }));

    // Approve should be disabled again — downloaded set cleared.
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /approve/i })).toBeDisabled();
    });
  });

  it('clears selected set after successful refresh', async () => {
    mockUseSnippets.mockReturnValue({
      data: [unapprovedSnippet],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippets>);
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: vi.fn((_, callbacks) => {
        callbacks?.onSuccess?.({ refreshed_at: '2026-05-07T12:34:56Z' });
      }),
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);

    render(<SnippetsListPage isAdmin />);

    // Select the snippet.
    fireEvent.click(
      screen.getByRole('checkbox', { name: /select check\.sh/i })
    );
    expect(
      screen.getByRole('button', { name: /batch approve \(1\)/i })
    ).toBeInTheDocument();

    // Trigger refresh.
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /^refresh$/i }));

    // Checkbox unchecked — selected set cleared.
    await waitFor(() => {
      expect(
        screen.getByRole('checkbox', { name: /select check\.sh/i })
      ).not.toBeChecked();
    });
  });

  it('resets the service-type filter after a successful refresh', async () => {
    mockUseSnippets.mockReturnValue({
      data: [
        { ...unapprovedSnippet, filename: 'mysql.sh', service_type: 'mysql' },
      ],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippets>);
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: true },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: vi.fn((_, callbacks) => {
        callbacks?.onSuccess?.({ refreshed_at: '2026-05-07T12:34:56Z' });
      }),
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);

    render(<SnippetsListPage isAdmin />);

    // Narrow to the mysql service type.
    fireEvent.mouseDown(screen.getByLabelText('Filter by service type'));
    fireEvent.click(screen.getByRole('option', { name: 'mysql' }));
    expect(screen.getByLabelText('Filter by service type')).toHaveTextContent(
      'mysql'
    );

    // Refresh resets the filter back to "All services".
    fireEvent.click(screen.getByRole('button', { name: /refresh snippets/i }));
    fireEvent.click(screen.getByRole('button', { name: /^refresh$/i }));

    await waitFor(() => {
      expect(screen.getByLabelText('Filter by service type')).toHaveTextContent(
        'All services'
      );
    });
  });
});

describe('SnippetsListPage — filters', () => {
  const mysqlUnapproved = {
    ...unapprovedSnippet,
    filename: 'mysql-log.sh',
    title: 'MySQL log rotate',
    description: 'rotates logs',
    service_type: 'mysql',
    is_approved: false,
  };
  const mongoApproved = {
    ...unapprovedSnippet,
    filename: 'mongo-status.sh',
    title: 'Mongo status',
    description: 'status check',
    service_type: 'mongodb',
    is_approved: true,
  };
  const mongoUnapproved = {
    ...unapprovedSnippet,
    filename: 'mongo-slow.sh',
    title: 'Mongo slow query',
    description: 'slow query log helper',
    service_type: 'mongodb',
    is_approved: false,
  };
  const uncategorized = {
    ...unapprovedSnippet,
    filename: 'misc.sh',
    title: 'Miscellaneous',
    description: 'no service type here',
    service_type: null,
    is_approved: false,
  };

  const allSnippets = [
    mysqlUnapproved,
    mongoApproved,
    mongoUnapproved,
    uncategorized,
  ];

  function bodyFilenames(): string[] {
    const rows = screen.getAllByRole('row').slice(1); // drop header row
    // Non-admin rows have no checkbox column, so the first cell is Filename.
    return rows.map(
      (row) => within(row).getAllByRole('cell')[0].textContent ?? ''
    );
  }

  function selectOption(controlLabel: string, optionName: string) {
    fireEvent.mouseDown(screen.getByLabelText(controlLabel));
    fireEvent.click(screen.getByRole('option', { name: optionName }));
  }

  beforeEach(() => {
    vi.clearAllMocks();
    mockUseSnippets.mockReturnValue({
      data: allSnippets,
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippets>);
    mockUseApproveSnippet.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useApproveSnippet>);
    mockUseRemoveSnippetApproval.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useRemoveSnippetApproval>);
    mockUseBatchApproveSnippets.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useBatchApproveSnippets>);
    mockUseSnippetsCapabilities.mockReturnValue({
      data: { manual_sync_enabled: false },
      isLoading: false,
    } as unknown as ReturnType<typeof useSnippetsCapabilities>);
    mockUseRefreshSnippets.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useRefreshSnippets>);
  });

  it('free-text search matches filename, title, and description (case-insensitive)', () => {
    render(<SnippetsListPage />);

    fireEvent.change(screen.getByLabelText('Search snippets'), {
      target: { value: 'SLOW' },
    });

    expect(bodyFilenames()).toEqual(['mongo-slow.sh']);
  });

  it('approval filter narrows to not-approved snippets', () => {
    render(<SnippetsListPage />);

    selectOption('Filter by approval status', 'Not approved');

    expect(bodyFilenames()).toEqual([
      'mysql-log.sh',
      'mongo-slow.sh',
      'misc.sh',
    ]);
  });

  it('service-type filter narrows to a single service', () => {
    render(<SnippetsListPage />);

    selectOption('Filter by service type', 'mongodb');

    expect(bodyFilenames()).toEqual(['mongo-status.sh', 'mongo-slow.sh']);
  });

  it('Uncategorized option shows only snippets without a service type', () => {
    render(<SnippetsListPage />);

    selectOption('Filter by service type', 'Uncategorized');

    expect(bodyFilenames()).toEqual(['misc.sh']);
  });

  it('combines filters with AND semantics', () => {
    render(<SnippetsListPage />);

    selectOption('Filter by approval status', 'Not approved');
    selectOption('Filter by service type', 'mongodb');
    fireEvent.change(screen.getByLabelText('Search snippets'), {
      target: { value: 'log' },
    });

    expect(bodyFilenames()).toEqual(['mongo-slow.sh']);
  });

  it('shows an empty state when no snippet matches the active filters', () => {
    render(<SnippetsListPage />);

    fireEvent.change(screen.getByLabelText('Search snippets'), {
      target: { value: 'no-such-snippet' },
    });

    expect(screen.queryByRole('table')).not.toBeInTheDocument();
    expect(
      screen.getByText(/no snippets match the current filters/i)
    ).toBeInTheDocument();
  });

  it('keeps a snippet whose service_type equals the "all" sentinel selectable', () => {
    const literalAll = {
      ...unapprovedSnippet,
      filename: 'literal-all.sh',
      title: 'Literal all',
      description: '',
      service_type: 'all',
      is_approved: false,
    };
    mockUseSnippets.mockReturnValue({
      data: [literalAll, mongoUnapproved],
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useSnippets>);

    render(<SnippetsListPage />);

    // The literal "all" service type gets its own option, distinct from the
    // "All services" (no-filter) entry, and selecting it narrows to that row.
    selectOption('Filter by service type', 'all');

    expect(bodyFilenames()).toEqual(['literal-all.sh']);
  });

  it('drops selections for rows hidden by a filter from the batch payload', () => {
    const batchMutate = vi.fn();
    mockUseBatchApproveSnippets.mockReturnValue({
      mutate: batchMutate,
      isPending: false,
    } as unknown as ReturnType<typeof useBatchApproveSnippets>);

    render(<SnippetsListPage isAdmin />);

    // Select an unapproved mysql snippet.
    fireEvent.click(
      screen.getByRole('checkbox', { name: /select mysql-log\.sh/i })
    );
    expect(
      screen.getByRole('button', { name: /batch approve \(1\)/i })
    ).toBeInTheDocument();

    // Filter to mongodb — mysql-log.sh is now hidden, so the batch action
    // (and its payload) must no longer include it.
    selectOption('Filter by service type', 'mongodb');

    expect(
      screen.queryByRole('button', { name: /batch approve/i })
    ).not.toBeInTheDocument();
    expect(batchMutate).not.toHaveBeenCalled();
  });
});
