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
  Advisor,
  AdvisorCheckResultStatus,
  AdvisorCheckTriggeredBy,
  AdvisorFamily,
  AdvisorInterval,
  CheckResultHistoryItem,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { format } from 'date-fns';
import { TIME_FORMAT } from 'lib/constants';

vi.mock('api/advisors');

const TEST_ADVISORS: Advisor[] = [
  {
    name: 'test_advisor',
    summary: 'Test advisor',
    description: '',
    comment: '',
    category: 'configuration',
    checks: [
      {
        name: 'mysql_version_check',
        enabled: true,
        summary: 'MySQL version check',
        description: '',
        interval: AdvisorInterval.standard,
        family: AdvisorFamily.mysql,
      },
      {
        name: 'postgresql_super_role',
        enabled: false,
        summary: 'PostgreSQL super role',
        description: '',
        interval: AdvisorInterval.rare,
        family: AdvisorFamily.postgresql,
      },
    ],
  },
];

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
  batchId: 'batch-1',
  triggeredBy: AdvisorCheckTriggeredBy.user,
  outcome: 'Installed version: 5.7.30',
  environment: 'prod',
  cluster: 'mysql-cluster',
  replicationSet: 'rs1',
};

const TEST_ITEM_UNREAD: CheckResultHistoryItem = {
  ...TEST_ITEM,
  id: 'result-2',
  checkName: 'postgresql_super_role',
  serviceName: 'postgresql-prod',
  serviceType: 'postgresql',
  summary: 'PostgreSQL super role detected',
  description: 'A user has the SUPER role',
  severity: Severity.error,
  isRead: false,
  // recorded before batch grouping existed
  batchId: '',
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
    vi.mocked(advisorsApi.listAdvisors).mockResolvedValue(TEST_ADVISORS);
    vi.mocked(advisorsApi.listCheckResultsHistory).mockResolvedValue({
      totalItems: 250,
      totalPages: 3,
      results: [TEST_ITEM, TEST_ITEM_UNREAD],
    });
    vi.mocked(advisorsApi.markCheckResultsRead).mockResolvedValue();
    vi.mocked(advisorsApi.startAdvisorChecks).mockResolvedValue('batch-123');
    vi.mocked(advisorsApi.changeAdvisorChecks).mockResolvedValue();
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

  it('passes the batchId deep link to the API', async () => {
    renderComponent('/advisors/insights?batchId=batch-42');

    await waitForRows();

    expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
      expect.objectContaining({ batchId: 'batch-42' })
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
    const advisorsCalls = vi.mocked(advisorsApi.listAdvisors).mock.calls.length;
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
      expect(header.querySelector('.MuiTableSortLabel-root')).not.toBeNull();
    }
  });

  it('marks an unread insight as read via the row menu', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-2-actions'));
    fireEvent.click(await screen.findByTestId('action-toggle-read'));

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

  it('marks a read insight as unread via the row menu', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-toggle-read'));

    await waitFor(() =>
      expect(advisorsApi.markCheckResultsRead).toHaveBeenCalledWith(
        { ids: ['result-1'], isRead: false },
        expect.anything()
      )
    );
  });

  it('opens and closes the details pane from the row menu', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-view-details'));

    const pane = await screen.findByTestId('insight-details-pane');
    expect(pane).toBeInTheDocument();
    expect(screen.getByText(Messages.details.title)).toBeInTheDocument();
    expect(screen.getByTestId('insight-details-maximize')).toBeInTheDocument();
    // opened from the menu, the pane starts at the default (non-maximized) height
    expect(
      within(pane).getByTestId('OpenInFullOutlinedIcon')
    ).toBeInTheDocument();

    // content from the selected insight
    expect(within(pane).getByText('MySQL is outdated')).toBeInTheDocument();
    expect(
      within(pane).getByText('Newer version of MySQL is available')
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId('details-field-check-name')).getByText(
        'mysql_version_check'
      )
    ).toBeInTheDocument();
    // the underlying check is enabled in the advisors fixture
    expect(
      within(screen.getByTestId('details-field-advisor-status')).getByText(
        Messages.details.enabled
      )
    ).toBeInTheDocument();
    // populated topology fields and outcome
    expect(within(pane).getByText('prod')).toBeInTheDocument();
    expect(within(pane).getByText('mysql-cluster')).toBeInTheDocument();
    expect(within(pane).getByText('rs1')).toBeInTheDocument();
    expect(
      within(pane).getByText('Installed version: 5.7.30')
    ).toBeInTheDocument();
    // not-yet-populated fields (Region, AZ) render an em-dash
    expect(within(pane).getAllByText('—').length).toBeGreaterThanOrEqual(2);
    // the history record ID is shown as Check ID
    expect(
      within(screen.getByTestId('details-field-check-id')).getByText('result-1')
    ).toBeInTheDocument();
    // the labels section title is always shown, even without labels
    expect(within(pane).getByText(Messages.details.labels)).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('insight-details-close'));

    await waitFor(() =>
      expect(
        screen.queryByTestId('insight-details-pane')
      ).not.toBeInTheDocument()
    );
  });

  it('opens the details overlay maximized on row double-click', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.dblClick(screen.getByTestId('insight-row-result-1'));

    const pane = await screen.findByTestId('insight-details-pane');
    expect(within(pane).getByText('MySQL is outdated')).toBeInTheDocument();
    // a maximized pane shows the "collapse" toggle icon
    expect(
      within(pane).getByTestId('CloseFullscreenOutlinedIcon')
    ).toBeInTheDocument();
  });

  it('copies the insight as a narrative text', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-copy-as-text'));

    const checkedAt = format(new Date(TEST_ITEM.checkedAt), TIME_FORMAT);
    const expected =
      `The Advisor Check "MySQL is outdated" completed at ${checkedAt} ` +
      'with status "Failed".\n' +
      '\n' +
      'Check Details:\n' +
      '  ID: result-1\n' +
      '  Batch ID: batch-1\n' +
      '  Check Name: mysql_version_check\n' +
      '  Advisor: MySQL Version\n' +
      '  Category: Version_configuration\n' +
      '  Service Name: mysql-prod\n' +
      '  Service Type: mysql\n' +
      '  Node Name: node-1\n' +
      '  Environment: prod\n' +
      '  Cluster: mysql-cluster\n' +
      '  Replication Set: rs1\n' +
      '  Interval: Standard\n' +
      '  Triggered By: User\n' +
      '  Read: Read\n' +
      '  Summary: MySQL is outdated\n' +
      '  Description: Newer version of MySQL is available\n' +
      '  Outcome: Installed version: 5.7.30\n' +
      '  Severity: Warning\n' +
      '  Read More: https://percona.com';

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expected)
    );
    expect(
      await screen.findByText(Messages.success.copied)
    ).toBeInTheDocument();
  });

  it('dims rows whose checks are disabled', async () => {
    renderComponent();

    await waitForRows();

    // postgresql_super_role is disabled in the advisors fixture
    await waitFor(() =>
      expect(screen.getByTestId('insight-row-result-2')).toHaveAttribute(
        'data-check-disabled',
        'true'
      )
    );
    expect(screen.getByTestId('insight-row-result-1')).not.toHaveAttribute(
      'data-check-disabled'
    );
  });

  it('disables an enabled check from the row menu', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    const item = await screen.findByTestId('action-disable-check');
    expect(item).toHaveTextContent(Messages.actions.disableCheck);

    const advisorsCalls = vi.mocked(advisorsApi.listAdvisors).mock.calls.length;
    fireEvent.click(item);

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [{ name: 'mysql_version_check', enable: false }],
        expect.anything()
      )
    );
    expect(
      await screen.findByText(
        Messages.success.checkDisabled('MySQL version check')
      )
    ).toBeInTheDocument();
    // the advisors list refetches, so the menu reflects the new state
    await waitFor(() =>
      expect(
        vi.mocked(advisorsApi.listAdvisors).mock.calls.length
      ).toBeGreaterThan(advisorsCalls)
    );
  });

  it('offers enabling for a disabled check', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-2-actions'));
    const item = await screen.findByTestId('action-disable-check');
    expect(item).toHaveTextContent(Messages.actions.enableCheck);

    fireEvent.click(item);

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [{ name: 'postgresql_super_role', enable: true }],
        expect.anything()
      )
    );
  });

  it('disables re-run for disabled checks and batch filter for rows without batch ID', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-2-actions'));
    await screen.findByTestId('action-view-details');

    expect(screen.getByTestId('action-rerun-now')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
    expect(screen.getByTestId('action-filter-by-batch-id')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it('filters by batch ID from the row menu, populates the input, then clears', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-filter-by-batch-id'));

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ batchId: 'batch-1', pageIndex: 0 })
      )
    );

    // the batch-id input reflects the applied filter
    const input = within(screen.getByTestId('batch-id-filter')).getByRole(
      'textbox',
      { hidden: true }
    );
    expect(input).toHaveValue('batch-1');

    fireEvent.click(screen.getByTestId('clear-filters'));

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenLastCalledWith(
        expect.objectContaining({ batchId: undefined })
      )
    );
    expect(input).toHaveValue('');
  });

  it('filters by a batch ID typed into the input and committed with Enter', async () => {
    renderComponent();

    await waitForRows();

    const input = within(screen.getByTestId('batch-id-filter')).getByRole(
      'textbox',
      { hidden: true }
    );
    fireEvent.change(input, { target: { value: 'batch-77' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenCalledWith(
        expect.objectContaining({ batchId: 'batch-77', pageIndex: 0 })
      )
    );
  });

  it('re-runs the check from the row menu and links to the new batch', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-rerun-now'));

    await waitFor(() =>
      expect(advisorsApi.startAdvisorChecks).toHaveBeenCalledWith(
        ['mysql_version_check'],
        expect.anything()
      )
    );
    expect(
      await screen.findByText(
        Messages.success.rerunStarted('MySQL version check')
      )
    ).toBeInTheDocument();

    fireEvent.click(screen.getByTestId('view-run-results'));

    await waitFor(() =>
      expect(advisorsApi.listCheckResultsHistory).toHaveBeenLastCalledWith(
        expect.objectContaining({ batchId: 'batch-123' })
      )
    );
  });
});
