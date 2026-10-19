import { fireEvent, render, screen, within } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import type { Investigation } from 'api/investigations';
import InvestigationDetailPage from './InvestigationDetailPage';

const { state, runMutate, chatMutate, mutation } = vi.hoisted(() => ({
  state: { inv: undefined as unknown },
  runMutate: vi.fn(),
  chatMutate: vi.fn(),
  mutation:
    (mutate = vi.fn()) =>
    () => ({ mutate, isPending: false }),
}));

vi.mock('hooks/api/useInvestigations', () => ({
  INVESTIGATIONS_KEYS: {
    detail: (id: string) => ['investigations', id],
    messagesPrefix: (id: string) => ['investigations', id, 'messages'],
    usage: (id: string) => ['investigations', id, 'usage'],
  },
  useInvestigation: () => ({
    data: state.inv,
    isLoading: false,
    isError: false,
  }),
  useInvestigationComments: () => ({ data: [] }),
  useInvestigationMessages: () => ({
    data: [
      {
        id: 'm2',
        role: 'assistant',
        content: 'The replica is **writable**.',
        createdAt: '2026-10-16T13:05:00Z',
      },
      {
        id: 'm1',
        role: 'user',
        content: 'Why did replication stop?',
        createdAt: '2026-10-16T13:04:00Z',
      },
    ],
  }),
  useInvestigationTimeline: () => ({
    data: [
      {
        id: 't1',
        eventTime: '2026-10-16T13:01:00Z',
        title: 'SQL thread stops',
        description: 'Duplicate key error 1062',
      },
    ],
  }),
  useSimilarInvestigations: () => ({ data: [] }),
  usePostInvestigationComment: mutation(),
  usePostInvestigationChat: mutation(chatMutate),
  usePostInvestigationRun: mutation(runMutate),
  usePatchInvestigation: mutation(),
  usePatchInvestigationBlock: mutation(),
  useDeleteInvestigationBlock: mutation(),
  useCreateServiceNowTicket: mutation(),
}));

vi.mock('hooks/api/useAdre', () => ({
  useAdreSettings: () => ({ data: { servicenowConfigured: false } }),
}));

vi.mock('hooks/api/useAdreUsage', () => ({
  useInvestigationUsage: () => ({ data: undefined, isLoading: false }),
}));

const investigation = (patch: Partial<Investigation>): Investigation =>
  ({
    id: 'inv1',
    title: 'Alert: MySQL replication SQL thread stopped',
    status: 'completed',
    severity: 'critical',
    createdAt: '2026-10-16T13:00:00Z',
    updatedAt: '2026-10-16T13:10:00Z',
    createdBy: 'auto-investigate',
    timeFrom: '2026-10-16T12:30:00Z',
    timeTo: '2026-10-16T13:30:00Z',
    summary: 'Replication stopped on a duplicate key.',
    summaryDetailed: '',
    rootCauseSummary: 'A script writes to both source and replica.',
    resolutionSummary: 'Set read_only on the replica.',
    sourceType: 'alert',
    sourceRef: '',
    confidence: 'high',
    confidenceScore: 90,
    confidenceRationale: 'Three sources agree.',
    evidence: [
      {
        id: 'e1',
        kind: 'log',
        claim: 'Error 1062 on the replica',
        excerpt: 'Duplicate entry',
        verification: 'verified',
      },
    ],
    blocks: [
      {
        id: 'b1',
        investigationId: 'inv1',
        type: 'remediation_steps',
        title: 'Next steps',
        position: 0,
        dataJson: {
          steps: ['Set super_read_only', 'Point the client at the source'],
        },
        createdAt: '',
        updatedAt: '',
      },
    ],
    ...patch,
  }) as Investigation;

const renderPage = () =>
  render(
    wrapWithQueryProvider(
      <TestWrapper routerProps={{ initialEntries: ['/investigations/inv1'] }}>
        <Routes>
          <Route
            path="/investigations/:id"
            element={<InvestigationDetailPage />}
          />
        </Routes>
      </TestWrapper>
    )
  );

describe('InvestigationDetailPage', () => {
  beforeEach(() => vi.clearAllMocks());

  it('leads with the summary, root cause, resolution and how sure it is', () => {
    state.inv = investigation({});
    renderPage();

    const lead = screen.getByRole('region', { name: 'Summary' });
    expect(lead).toHaveTextContent('Replication stopped on a duplicate key.');
    expect(lead).toHaveTextContent('Root cause');
    expect(lead).toHaveTextContent(
      'A script writes to both source and replica.'
    );
    expect(lead).toHaveTextContent('Set read_only on the replica.');
    expect(lead).toHaveTextContent('High confidence · 90');

    fireEvent.click(within(lead).getByRole('button', { name: 'Evidence (1)' }));
    expect(lead).toHaveTextContent('Error 1062 on the replica');
  });

  it('lays out the timeline, report and chat like the Assistant', () => {
    state.inv = investigation({});
    renderPage();

    const timeline = screen.getByRole('region', { name: 'Timeline' });
    expect(timeline).toHaveTextContent('2026-10-16 13:01:00');
    expect(timeline).toHaveTextContent('SQL thread stops');

    const report = screen.getByRole('region', { name: 'Report' });
    expect(
      within(report).getByRole('heading', { name: 'Next steps' })
    ).toBeInTheDocument();
    expect(
      within(report).getByRole('button', { name: 'Delete block' })
    ).toBeInTheDocument();

    const chat = screen.getByRole('region', { name: 'Chat' });
    expect(within(chat).getByText('writable').tagName).toBe('STRONG');
    fireEvent.change(
      within(chat).getByLabelText('Ask about this investigation'),
      {
        target: { value: 'And now?' },
      }
    );
    fireEvent.keyDown(
      within(chat).getByLabelText('Ask about this investigation'),
      {
        key: 'Enter',
      }
    );
    expect(chatMutate).toHaveBeenCalledWith('And now?', expect.anything());
  });

  it('runs an open investigation and explains why a finished one cannot', () => {
    state.inv = investigation({});
    const { unmount } = renderPage();
    expect(
      screen.getByRole('button', { name: 'Run investigation' })
    ).toBeDisabled();
    unmount();

    state.inv = investigation({
      status: 'open',
      summary: 'Check the replica',
      rootCauseSummary: '',
      resolutionSummary: '',
      confidenceScore: 0,
      evidence: [],
      blocks: [],
    });
    renderPage();
    expect(screen.getByText('Check the replica')).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Summary' })).toHaveTextContent(
      'No report yet'
    );
    fireEvent.click(screen.getByRole('button', { name: 'Run investigation' }));
    expect(runMutate).toHaveBeenCalled();
  });
});
