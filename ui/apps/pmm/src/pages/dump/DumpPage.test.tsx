import { render, screen } from '@testing-library/react';
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

vi.mock('@percona/percona-ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@percona/percona-ui')>()),
  Table: ({
    data,
    noDataMessage,
  }: {
    data: unknown[];
    noDataMessage: string;
  }) => <div data-testid="pmm-dump-table">{data.length || noDataMessage}</div>,
}));

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
