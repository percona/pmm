import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import AdvisorInsights from './AdvisorInsights';
import { Messages } from './AdvisorInsights.messages';
import * as advisorsApi from 'api/advisors';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import {
  AdvisorCheckResultStatus,
  AdvisorInterval,
  CheckResultHistoryItem,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';

vi.mock('api/advisors');

const TEST_ITEM: CheckResultHistoryItem = {
  id: 'result-1',
  checkName: 'mysql_version_check',
  advisorName: 'MySQL Version',
  category: 'version_configuration',
  interval: AdvisorInterval.standard,
  serviceId: 'service-1',
  serviceName: 'mysql-prod',
  serviceType: 'mysql',
  nodeId: 'node-1',
  nodeName: 'node-1',
  status: AdvisorCheckResultStatus.failed,
  summary: 'MySQL is outdated',
  description: 'Newer version of MySQL is available',
  readMoreUrl: 'https://percona.com',
  severity: Severity.warning,
  labels: {},
  checkedAt: '2026-07-05T10:00:00Z',
  isRead: true,
};

const TEST_ITEM_UNREAD: CheckResultHistoryItem = {
  ...TEST_ITEM,
  id: 'result-2',
  serviceName: 'postgresql-prod',
  serviceType: 'postgresql',
  summary: 'PostgreSQL super role detected',
  description: 'A user has the SUPER role',
  severity: Severity.error,
  isRead: false,
};

const renderComponent = () =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithUserProvider(wrapWithRouter(<AdvisorInsights />))
      )
    )
  );

const waitForRows = async () => {
  await waitFor(() =>
    expect(screen.getByText('MySQL is outdated')).toBeInTheDocument()
  );
};

describe('AdvisorInsights', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(advisorsApi.listAdvisors).mockResolvedValue([]);
    vi.mocked(advisorsApi.listCheckResultsHistory).mockResolvedValue({
      totalItems: 250,
      totalPages: 3,
      results: [TEST_ITEM, TEST_ITEM_UNREAD],
    });
    vi.mocked(advisorsApi.markCheckResultsRead).mockResolvedValue();
  });

  it('renders history items with read state', async () => {
    renderComponent();

    await waitForRows();

    expect(
      screen.getByText('PostgreSQL super role detected')
    ).toBeInTheDocument();
    expect(screen.getByTestId('insight-result-1-read-state')).toHaveTextContent(
      Messages.read
    );
    expect(screen.getByTestId('insight-result-2-read-state')).toHaveTextContent(
      Messages.unread
    );
  });

  it('requests the first page by default', async () => {
    renderComponent();

    await waitForRows();

    expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
      expect.objectContaining({ pageIndex: 0, pageSize: 100 })
    );
  });

  it('requests the next page on pagination change', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByRole('button', { name: /go to next page/i }));

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ pageIndex: 1 })
      )
    );
  });

  it('passes the service filter to the API and resets the page', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByRole('button', { name: /go to next page/i }));

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ pageIndex: 1 })
      )
    );

    fireEvent.change(screen.getByPlaceholderText('Filter by Service'), {
      target: { value: 'mysql-prod' },
    });

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ serviceName: 'mysql-prod', pageIndex: 0 })
      )
    );
  });

  it('marks an unread insight as read', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-2-toggle-read'));

    await waitFor(() =>
      expect(advisorsApi.markCheckResultsRead).toHaveBeenCalledWith(
        { ids: ['result-2'], isRead: true },
        expect.anything()
      )
    );
    expect(
      await screen.findByText(Messages.success.markedRead)
    ).toBeInTheDocument();
  });

  it('marks a read insight as unread', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-toggle-read'));

    await waitFor(() =>
      expect(advisorsApi.markCheckResultsRead).toHaveBeenCalledWith(
        { ids: ['result-1'], isRead: false },
        expect.anything()
      )
    );
  });
});
