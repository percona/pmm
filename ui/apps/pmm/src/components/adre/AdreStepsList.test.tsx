import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AdreStepsList } from './AdreStepsList';

vi.mock('hooks/api/useServices', () => ({
  useManagedServices: () => ({ data: undefined }),
}));

describe('AdreStepsList', () => {
  it('shows each step’s query on demand and links to its data', () => {
    render(
      <MemoryRouter>
        <AdreStepsList
          working={false}
          steps={[
            {
              id: '1',
              toolName: 'execute_prometheus_range_query',
              status: 'done',
              params: {
                query: 'up',
                start: '-3600',
                description: 'Targets up',
              },
            },
            {
              id: '2',
              toolName: 'pmm_logs_patterns',
              status: 'done',
              failed: true,
              error: 'ClickHouse timed out',
            },
          ]}
        />
      </MemoryRouter>
    );

    expect(screen.getByText('Targets up')).toBeInTheDocument();
    expect(screen.getByLabelText('Failed')).toBeInTheDocument();
    expect(screen.getByText('ClickHouse timed out')).toBeInTheDocument();
    expect(screen.queryByText('up')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Show query' }));
    expect(screen.getByText('up')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Hide query' })).toHaveAttribute(
      'aria-expanded',
      'true'
    );

    const explore = screen.getByRole('link', { name: 'Open in Explore' });
    expect(explore.getAttribute('href')).toMatch(/^\/graph\/explore\?/);
    expect(explore).toHaveAttribute('target', '_blank');
  });
});
