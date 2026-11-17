import { fireEvent, render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { AdreAlertsPanel, type AlertItem } from './AdreAlertsPanel';

const { mutate, similarTo } = vi.hoisted(() => ({
  mutate: vi.fn(),
  similarTo: vi.fn(),
}));

vi.mock('hooks/api/useInvestigations', () => ({
  useCreateInvestigation: () => ({ mutate, isPending: false }),
  useSimilarToIncident: similarTo,
}));

const alerts: AlertItem[] = [
  {
    fingerprint: 'cafbab3831cea4fe',
    labels: {
      alertname: 'PMM agent down',
      node_name: 'ps_pmm_replication_8_4_2',
      service_name: 'pmm-managed',
      severity: 'critical',
    },
  },
  {
    fingerprint: 'dec81569f438ff67',
    labels: {
      alertname: 'PMM agent down',
      node_name: 'rs201._9419',
      severity: 'critical',
    },
  },
];

const renderPanel = () =>
  render(
    wrapWithQueryProvider(
      <TestWrapper>
        <AdreAlertsPanel alerts={alerts} />
      </TestWrapper>
    )
  );

describe('AdreAlertsPanel', () => {
  it('links an alert to the past investigation it looks like', () => {
    similarTo.mockImplementation((query: { node_name?: string }) => ({
      data:
        query.node_name === 'ps_pmm_replication_8_4_2'
          ? [
              {
                id: 'i1',
                title: 'Alert: PMM agent down',
                createdAt: '2026-10-16T13:00:00Z',
                reasons: ['Same alert', 'Same node'],
              },
              {
                id: 'i2',
                title: 'Alert: PMM agent down',
                createdAt: '2026-10-15T13:00:00Z',
                reasons: ['Same node'],
              },
            ]
          : [],
    }));
    renderPanel();

    expect(similarTo).toHaveBeenCalledWith(
      {
        alertname: 'PMM agent down',
        service_name: 'pmm-managed',
        node_name: 'ps_pmm_replication_8_4_2',
        cluster: undefined,
      },
      true
    );
    const seen = screen.getAllByText(/^Seen before/);
    expect(seen).toHaveLength(1);
    expect(seen[0]).toHaveTextContent(
      /^Seen before \(same alert, same node\): Alert: PMM agent down/
    );
    expect(seen[0]).toHaveTextContent('+1 more');
    expect(
      screen.getByRole('link', { name: 'Alert: PMM agent down' })
    ).toHaveAttribute('href', '/investigations/i1');
  });

  it('shows each alert as a card with its node, service and severity', () => {
    renderPanel();
    expect(screen.getByText('Firing alerts')).toBeInTheDocument();
    expect(screen.getByText('ps_pmm_replication_8_4_2')).toBeInTheDocument();
    expect(screen.getByText('pmm-managed')).toBeInTheDocument();
    expect(screen.getAllByText('critical')).toHaveLength(2);
    expect(screen.getByText('cafbab3831cea4fe')).toBeInTheDocument();
  });

  it('counts the selected alerts on the button', () => {
    renderPanel();
    expect(
      screen.getByRole('button', { name: 'Start investigation' })
    ).toBeDisabled();
    fireEvent.click(screen.getAllByRole('checkbox')[0]);
    fireEvent.click(
      screen.getByRole('button', { name: 'Start investigation (1 alert)' })
    );
    expect(mutate).toHaveBeenCalledWith(
      expect.objectContaining({ title: 'Alert: PMM agent down' }),
      expect.anything()
    );
  });
});
