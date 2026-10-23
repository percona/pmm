import { fireEvent, render, screen } from '@testing-library/react';
import { ReactNode } from 'react';
import { Dump, DumpStatus } from 'types/dump.types';
import { DumpPage } from './DumpPage';
import { TestWrapper } from 'utils/testWrapper';
import { TEST_USER_VIEWER } from 'utils/testStubs';

const mocks = vi.hoisted(() => ({
  useDumps: vi.fn(),
}));

vi.mock('hooks/api/useDump', () => ({
  useDumps: mocks.useDumps,
  useDeleteDumps: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDumpLogs: () => ({ data: undefined, isLoading: false, isError: false }),
  useUploadDumps: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock('@percona/peak-ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@percona/peak-ui')>()),
  Table: ({
    data,
    noDataMessage,
    onRowSelectionChange,
    renderTopToolbarCustomActions,
  }: {
    data: Dump[];
    noDataMessage: string;
    onRowSelectionChange: (selection: Record<string, boolean>) => void;
    renderTopToolbarCustomActions: () => ReactNode;
  }) => (
    <div data-testid="pmm-dump-table">
      {renderTopToolbarCustomActions()}
      {data.length || noDataMessage}
      <button
        onClick={() =>
          onRowSelectionChange(
            Object.fromEntries(data.map(({ dumpId }) => [dumpId, true]))
          )
        }
      >
        select all
      </button>
    </div>
  ),
}));

const createDump = (dumpId: string): Dump => ({
  dumpId,
  status: DumpStatus.Success,
  serviceNames: [],
  startTime: '2026-10-22T00:00:00Z',
  endTime: '2026-10-22T01:00:00Z',
  createdAt: '2026-10-22T01:00:00Z',
  encrypted: false,
});

describe('DumpPage', () => {
  beforeEach(() => {
    mocks.useDumps.mockReturnValue({
      data: { dumps: [] },
      isLoading: false,
      isError: false,
    });
  });

  it('shows the empty inventory and create action to admins', () => {
    render(
      <TestWrapper>
        <DumpPage />
      </TestWrapper>
    );

    expect(screen.getByTestId('pmm-dump-table')).toHaveTextContent(
      'No dumps available'
    );
    expect(screen.getByTestId('create-dataset')).toHaveAttribute(
      'href',
      '/pmm-dump/new'
    );
    expect(mocks.useDumps).toHaveBeenCalledWith({ enabled: true });
  });

  it('shows the selection count with bulk actions', () => {
    mocks.useDumps.mockReturnValue({
      data: { dumps: [createDump('dump-1'), createDump('dump-2')] },
      isLoading: false,
      isError: false,
    });
    render(
      <TestWrapper>
        <DumpPage />
      </TestWrapper>
    );

    fireEvent.click(screen.getByRole('button', { name: 'select all' }));

    expect(screen.getByTestId('dump-selected-count')).toHaveTextContent(
      '2 rows selected'
    );
    expect(screen.getByTestId('dump-delete-selected')).toHaveTextContent(
      'Delete 2 items'
    );
  });

  it('shows the unauthorized state to non-admin users', () => {
    render(
      <TestWrapper userContext={{ user: TEST_USER_VIEWER, isLoading: false }}>
        <DumpPage />
      </TestWrapper>
    );

    expect(screen.getByTestId('unauthorized')).toBeInTheDocument();
    expect(screen.queryByTestId('pmm-dump-page')).not.toBeInTheDocument();
    expect(mocks.useDumps).toHaveBeenCalledWith({ enabled: false });
  });
});
