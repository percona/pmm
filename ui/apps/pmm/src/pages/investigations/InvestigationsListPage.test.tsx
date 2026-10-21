import { fireEvent, render, screen, within } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import type { InvestigationListItem } from 'api/investigations';
import InvestigationsListPage from './InvestigationsListPage';

const { listSpy, deleteMutateAsync } = vi.hoisted(() => ({
  listSpy: vi.fn(),
  deleteMutateAsync: vi.fn(() => Promise.resolve()),
}));

const rows: InvestigationListItem[] = [
  {
    id: 'inv1',
    title: 'Alert: MySQL replication SQL thread stopped',
    status: 'completed',
    severity: 'critical',
    summary: 'A script writes to both source and replica.',
    createdBy: 'auto-investigate',
    createdAt: new Date(Date.now() - 2 * 3600_000).toISOString(),
    updatedAt: new Date().toISOString(),
    sourceType: 'alert',
    nodeName: 'ps-1',
    serviceName: 'ps-1-mysql',
  },
  {
    id: 'inv2',
    title: 'Chat: migration check',
    status: 'open',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    sourceType: 'chat',
  },
];

vi.mock('hooks/api/useInvestigations', () => ({
  useInvestigationsList: (params: unknown) => {
    listSpy(params);
    return { data: rows, isLoading: false, isError: false };
  },
  useCreateInvestigation: () => ({ mutate: vi.fn(), isPending: false }),
  useDeleteInvestigation: () => ({
    mutateAsync: deleteMutateAsync,
    isPending: false,
  }),
}));

vi.mock('./CreateInvestigationModal', () => ({
  CreateInvestigationModal: () => null,
}));

const renderPage = () =>
  render(
    wrapWithQueryProvider(
      <TestWrapper>
        <InvestigationsListPage />
      </TestWrapper>
    )
  );

describe('InvestigationsListPage', () => {
  beforeEach(() => vi.clearAllMocks());

  it('lists each investigation with its root cause, origin and status', () => {
    renderPage();

    const list = screen.getByRole('list', { name: 'Investigations' });
    const [alertRow, chatRow] = within(list).getAllByRole('listitem');

    expect(
      within(alertRow).getByRole('link', {
        name: 'Alert: MySQL replication SQL thread stopped',
      })
    ).toHaveAttribute('href', '/investigations/inv1');
    expect(alertRow).toHaveTextContent(
      'A script writes to both source and replica.'
    );
    expect(alertRow).toHaveTextContent('critical· Alert· ps-1· ps-1-mysql');
    expect(alertRow).toHaveTextContent('completed');
    expect(alertRow).toHaveTextContent('2 hours ago');

    expect(chatRow).toHaveTextContent('Assistant chat');
    expect(chatRow).toHaveTextContent('open');
    expect(screen.getByText('2 investigations')).toBeInTheDocument();
  });

  it('filters by trigger and sorts', () => {
    renderPage();
    expect(listSpy).toHaveBeenLastCalledWith({
      orderBy: 'created_at',
      order: 'desc',
      trigger: undefined,
    });

    const filter = screen.getByRole('group', {
      name: 'Filter investigations by trigger',
    });
    fireEvent.click(
      within(filter).getByRole('button', { name: 'Auto-investigated' })
    );
    expect(
      within(filter).getByRole('button', { name: 'Auto-investigated' })
    ).toHaveAttribute('aria-pressed', 'true');
    expect(listSpy).toHaveBeenLastCalledWith(
      expect.objectContaining({ trigger: 'auto' })
    );

    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: 'Sort investigations' })
    );
    fireEvent.click(screen.getByRole('option', { name: 'Title' }));
    expect(listSpy).toHaveBeenLastCalledWith({
      orderBy: 'title',
      order: 'asc',
      trigger: 'auto',
    });
  });

  it('deletes the selected investigations', async () => {
    renderPage();

    fireEvent.click(
      screen.getByRole('checkbox', { name: 'Select Chat: migration check' })
    );
    expect(screen.getByText('1 selected')).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole('button', { name: 'Delete selected (1)' })
    );
    expect(deleteMutateAsync).toHaveBeenCalledWith('inv2');
    expect(await screen.findByText('2 investigations')).toBeInTheDocument();
  });
});
