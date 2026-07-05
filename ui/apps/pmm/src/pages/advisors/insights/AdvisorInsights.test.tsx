import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
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
  AdvisorCheckTriggeredBy,
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
  runId: 'run-1',
  triggeredBy: AdvisorCheckTriggeredBy.user,
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

const renderComponent = (initialEntry = '/advisors/insights') =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithUserProvider(
          wrapWithRouter(<AdvisorInsights />, {
            initialEntries: [initialEntry],
          })
        )
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
    vi.mocked(advisorsApi.listCheckResultsFilterValues).mockResolvedValue({
      serviceNames: ['mysql-prod', 'postgresql-prod'],
      nodeNames: ['node-1', 'node-2'],
    });
  });

  const selectFilterOption = async (filterTestId: string, option: string) => {
    fireEvent.mouseDown(
      within(screen.getByTestId(filterTestId)).getByRole('combobox')
    );

    // 'hidden' skips the visibility computation, which crashes in jsdom:
    // nwsapi expands ':scope' with the listbox's unescaped React useId
    const listbox = await screen.findByRole('listbox', { hidden: true });
    fireEvent.click(within(listbox).getByText(option));
  };

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

  it('passes the runId deep link to the API', async () => {
    renderComponent('/advisors/insights?runId=run-42');

    await waitForRows();

    expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
      expect.objectContaining({ runId: 'run-42' })
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

    await selectFilterOption('serviceName-filter', 'mysql-prod');

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ serviceName: 'mysql-prod', pageIndex: 0 })
      )
    );
  });

  it('passes the node filter to the API', async () => {
    renderComponent();

    await waitForRows();

    await selectFilterOption('nodeName-filter', 'node-2');

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ nodeName: 'node-2', pageIndex: 0 })
      )
    );
  });

  it('clears all filters and resets to the first page', async () => {
    renderComponent();

    await waitForRows();

    expect(screen.getByTestId('clear-filters')).toBeDisabled();

    await selectFilterOption('serviceName-filter', 'mysql-prod');

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ serviceName: 'mysql-prod' })
      )
    );

    fireEvent.click(screen.getByTestId('clear-filters'));

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenLastCalledWith(
        expect.objectContaining({ serviceName: undefined, pageIndex: 0 })
      )
    );
    expect(screen.getByTestId('clear-filters')).toBeDisabled();
  });

  it('refetches results and filter data on refresh', async () => {
    renderComponent();

    await waitForRows();

    const historyCalls = vi.mocked(advisorsApi.listCheckResultsHistory).mock
      .calls.length;
    const advisorsCalls = vi.mocked(advisorsApi.listAdvisors).mock.calls
      .length;
    const filterValuesCalls = vi.mocked(
      advisorsApi.listCheckResultsFilterValues
    ).mock.calls.length;

    fireEvent.click(screen.getByTestId('refresh-insights'));

    await waitFor(() => {
      expect(
        vi.mocked(advisorsApi.listCheckResultsHistory).mock.calls.length
      ).toBe(historyCalls + 1);
      expect(vi.mocked(advisorsApi.listAdvisors).mock.calls.length).toBe(
        advisorsCalls + 1
      );
      expect(
        vi.mocked(advisorsApi.listCheckResultsFilterValues).mock.calls.length
      ).toBe(filterValuesCalls + 1);
    });
  });

  it('makes every column sortable except summary', async () => {
    renderComponent();

    await waitForRows();

    // 'hidden' skips the visibility computation, which crashes in jsdom:
    // nwsapi chokes on MRT's unescaped React useId in the pagination select
    const summaryHeader = screen.getByRole('columnheader', {
      name: Messages.columns.summary,
      hidden: true,
    });
    expect(summaryHeader.querySelector('.MuiTableSortLabel-root')).toBeNull();

    for (const name of [
      Messages.columns.service,
      Messages.columns.category,
      Messages.columns.severity,
      Messages.columns.status,
      Messages.columns.checkedAt,
      Messages.columns.read,
    ]) {
      // sortable headers embed the sort hint in the accessible name
      const header = screen.getByRole('columnheader', {
        name: new RegExp(name),
        hidden: true,
      });
      expect(
        header.querySelector('.MuiTableSortLabel-root')
      ).not.toBeNull();
    }
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
