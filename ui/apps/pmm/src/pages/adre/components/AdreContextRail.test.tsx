import { render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { AdreContextRail } from './AdreContextRail';

vi.mock('hooks/api/useInvestigations', () => ({
  useCreateInvestigation: () => ({ mutate: vi.fn(), isPending: false }),
}));

const renderRail = (props: Parameters<typeof AdreContextRail>[0]) =>
  render(
    wrapWithQueryProvider(
      <TestWrapper>
        <AdreContextRail {...props} />
      </TestWrapper>
    )
  );

describe('AdreContextRail', () => {
  it('shows how the latest answer was made', () => {
    renderRail({
      alerts: [],
      steps: [
        {
          id: '1',
          toolName: 'pmm_logs_patterns',
          description: 'pdpgsql_pmm_17_1, warnings and above',
          status: 'done',
        },
      ],
      working: false,
    });
    expect(screen.getByText('How this answer was made')).toBeInTheDocument();
    expect(screen.getByText('pmm_logs_patterns')).toBeInTheDocument();
    expect(
      screen.getByText('pdpgsql_pmm_17_1, warnings and above')
    ).toBeInTheDocument();
    expect(screen.queryByText('Firing alerts')).not.toBeInTheDocument();
  });

  it('says what it will show before the first answer', () => {
    renderRail({ alerts: [], steps: [], working: false });
    expect(
      screen.getByText('The steps of the latest answer show here.')
    ).toBeInTheDocument();
  });

  it('marks a running step while the answer is written', () => {
    renderRail({
      alerts: [],
      steps: [{ id: '1', toolName: 'pmm_clickhouse_query', status: 'running' }],
      working: true,
    });
    expect(screen.getByLabelText('Running')).toBeInTheDocument();
  });
});
