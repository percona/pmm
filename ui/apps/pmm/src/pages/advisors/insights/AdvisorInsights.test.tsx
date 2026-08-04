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
  AdvisorTechnology,
  AdvisorInterval,
  Insight,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { format } from 'date-fns';
import { TIME_FORMAT } from 'lib/constants';

vi.mock('api/advisors');

const TEST_ADVISORS: Advisor[] = [
  {
    category: 'Configuration',
    subcategory: 'Version',
    checks: [
      {
        name: 'mysql_version_check',
        enabled: true,
        summary: 'MySQL version check',
        description: '',
        interval: AdvisorInterval.standard,
        technology: AdvisorTechnology.mysql,
        category: 'Configuration',
        subcategory: 'Version',
        userDefined: false,
      },
      {
        name: 'postgresql_super_role',
        enabled: false,
        summary: 'PostgreSQL super role',
        description: '',
        interval: AdvisorInterval.rare,
        technology: AdvisorTechnology.postgresql,
        category: 'Configuration',
        subcategory: 'Version',
        userDefined: false,
      },
    ],
  },
];

const TEST_ITEM: Insight = {
  id: 'result-1',
  checkName: 'mysql_version_check',
  subcategory: 'Version',
  category: 'Configuration',
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
  outcome: 'Installed version: 5.7.30',
  environment: 'prod',
  cluster: 'mysql-cluster',
  replicationSet: 'rs1',
  region: '',
  az: '',
};

const TEST_ITEM_UNREAD: Insight = {
  ...TEST_ITEM,
  id: 'result-2',
  checkName: 'postgresql_super_role',
  serviceId: 'service-2',
  serviceName: 'postgresql-prod',
  serviceType: 'postgresql',
  summary: 'PostgreSQL super role detected',
  description: 'A user has the SUPER role',
  severity: Severity.error,
  isRead: false,
  // recorded before run grouping existed
  runId: '',
};

// same advisors fixture, with one check turned off for a single service
const withDisabledService = (checkName: string, serviceId: string): Advisor[] =>
  TEST_ADVISORS.map((advisor) => ({
    ...advisor,
    checks: advisor.checks.map((check) =>
      check.name === checkName
        ? { ...check, disabledServiceIds: [serviceId] }
        : check
    ),
  }));

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
    vi.mocked(advisorsApi.listInsights).mockResolvedValue({
      totalItems: 250,
      totalPages: 3,
      results: [TEST_ITEM, TEST_ITEM_UNREAD],
    });
    vi.mocked(advisorsApi.markInsightsRead).mockResolvedValue();
    vi.mocked(advisorsApi.startAdvisorChecks).mockResolvedValue('run-123');
    vi.mocked(advisorsApi.changeAdvisorChecks).mockResolvedValue();
    vi.mocked(advisorsApi.listInsightsFilterValues).mockResolvedValue({
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

    // read rows get an opened envelope, unread ones a sealed envelope
    const readToggle = screen.getByTestId('insight-result-1-read-state');
    expect(readToggle).toHaveAttribute('aria-label', Messages.markAsUnread);
    expect(within(readToggle).getByTestId('DraftsOutlinedIcon')).toBeVisible();

    const unreadToggle = screen.getByTestId('insight-result-2-read-state');
    expect(unreadToggle).toHaveAttribute('aria-label', Messages.markAsRead);
    expect(
      within(unreadToggle).getByTestId('MarkunreadOutlinedIcon')
    ).toBeVisible();
  });

  it('requests the first page by default', async () => {
    renderComponent();

    await waitForRows();

    expect(advisorsApi.listInsights).toHaveBeenCalledWith(
      expect.objectContaining({ pageIndex: 0, pageSize: 100 })
    );
  });

  it('passes the runId deep link to the API', async () => {
    renderComponent('/advisors/insights?runId=run-42');

    await waitForRows();

    expect(advisorsApi.listInsights).toHaveBeenCalledWith(
      expect.objectContaining({ runId: 'run-42' })
    );
  });

  it('reads filters and pagination from the URL (deep link)', async () => {
    renderComponent(
      '/advisors/insights?category=version_configuration&service=mysql-prod&page=2&pageSize=50'
    );

    await waitForRows();

    expect(advisorsApi.listInsights).toHaveBeenCalledWith(
      expect.objectContaining({
        category: 'version_configuration',
        serviceName: 'mysql-prod',
        pageIndex: 1,
        pageSize: 50,
      })
    );
  });

  it('requests the next page on pagination change', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByRole('button', { name: /go to next page/i }));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
        expect.objectContaining({ pageIndex: 1 })
      )
    );
  });

  it('passes the service filter to the API and resets the page', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByRole('button', { name: /go to next page/i }));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
        expect.objectContaining({ pageIndex: 1 })
      )
    );

    await selectFilterOption('serviceName-filter', 'mysql-prod');

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
        expect.objectContaining({ serviceName: 'mysql-prod', pageIndex: 0 })
      )
    );
  });

  it('passes the node filter to the API', async () => {
    renderComponent();

    await waitForRows();

    await selectFilterOption('nodeName-filter', 'node-2');

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
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
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
        expect.objectContaining({ serviceName: 'mysql-prod' })
      )
    );

    fireEvent.click(screen.getByTestId('clear-filters'));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenLastCalledWith(
        expect.objectContaining({ serviceName: undefined, pageIndex: 0 })
      )
    );
    expect(screen.getByTestId('clear-filters')).toBeDisabled();
  });

  it('refetches results and filter data on refresh', async () => {
    renderComponent();

    await waitForRows();

    const historyCalls = vi.mocked(advisorsApi.listInsights).mock.calls.length;
    const advisorsCalls = vi.mocked(advisorsApi.listAdvisors).mock.calls.length;
    const filterValuesCalls = vi.mocked(advisorsApi.listInsightsFilterValues)
      .mock.calls.length;

    fireEvent.click(screen.getByTestId('refresh-insights'));

    await waitFor(() => {
      expect(vi.mocked(advisorsApi.listInsights).mock.calls.length).toBe(
        historyCalls + 1
      );
      expect(vi.mocked(advisorsApi.listAdvisors).mock.calls.length).toBe(
        advisorsCalls + 1
      );
      expect(
        vi.mocked(advisorsApi.listInsightsFilterValues).mock.calls.length
      ).toBe(filterValuesCalls + 1);
    });
  });

  it('makes every column sortable except summary and actions', async () => {
    renderComponent();

    await waitForRows();

    // 'hidden' skips the visibility computation, which crashes in jsdom:
    // nwsapi chokes on MRT's unescaped React useId in the pagination select
    const summaryHeader = screen.getByRole('columnheader', {
      name: Messages.columns.summary,
      hidden: true,
    });
    expect(summaryHeader.querySelector('.MuiTableSortLabel-root')).toBeNull();

    const actionsHeader = screen.getByRole('columnheader', {
      name: Messages.columns.actions,
      hidden: true,
    });
    expect(actionsHeader.querySelector('.MuiTableSortLabel-root')).toBeNull();

    for (const name of [
      Messages.columns.service,
      Messages.columns.category,
      Messages.columns.severity,
      Messages.columns.status,
      Messages.columns.checkedAt,
    ]) {
      // sortable headers embed the sort hint in the accessible name
      const header = screen.getByRole('columnheader', {
        name: new RegExp(name),
        hidden: true,
      });
      expect(header.querySelector('.MuiTableSortLabel-root')).not.toBeNull();
    }
  });

  it('marks an unread insight as read from the envelope icon', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-2-read-state'));

    await waitFor(() =>
      expect(advisorsApi.markInsightsRead).toHaveBeenCalledWith(
        { ids: ['result-2'], isRead: true },
        expect.anything()
      )
    );
    expect(
      await screen.findByText(Messages.success.markedRead)
    ).toBeInTheDocument();
  });

  it('marks a read insight as unread from the envelope icon', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-read-state'));

    await waitFor(() =>
      expect(advisorsApi.markInsightsRead).toHaveBeenCalledWith(
        { ids: ['result-1'], isRead: false },
        expect.anything()
      )
    );
  });

  it('disables the bulk mark-as-read button when no filters are active', async () => {
    renderComponent();

    await waitForRows();

    // guards against accidentally marking every record as read
    expect(screen.getByTestId('mark-filtered-read')).toBeDisabled();
  });

  it('marks all insights matching the filters as read', async () => {
    renderComponent('/advisors/insights?service=mysql-prod&read=false');

    await waitForRows();

    const button = screen.getByTestId('mark-filtered-read');
    expect(button).toBeEnabled();
    fireEvent.click(button);

    await waitFor(() =>
      expect(advisorsApi.markInsightsRead).toHaveBeenCalledWith(
        {
          filters: {
            serviceName: 'mysql-prod',
            isRead: false,
          },
          isRead: true,
        },
        expect.anything()
      )
    );
    expect(
      await screen.findByText(Messages.success.markedFilteredRead)
    ).toBeInTheDocument();
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
    // every open starts maximized, including from the row menu
    expect(
      within(pane).getByTestId('CloseFullscreenOutlinedIcon')
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
    expect(
      within(screen.getByTestId('details-field-triggered-by')).getByText('User')
    ).toBeInTheDocument();
    // populated topology fields and outcome
    expect(within(pane).getByText('prod')).toBeInTheDocument();
    expect(within(pane).getByText('mysql-cluster')).toBeInTheDocument();
    expect(within(pane).getByText('rs1')).toBeInTheDocument();
    expect(
      within(pane).getByText('Installed version: 5.7.30')
    ).toBeInTheDocument();
    // Region comes from labels, which this insight has none of
    expect(within(pane).getAllByText('—').length).toBeGreaterThanOrEqual(1);
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

  it('renders labels as chips verbatim and reads Region from its own field', async () => {
    vi.mocked(advisorsApi.listInsights).mockResolvedValue({
      totalItems: 1,
      totalPages: 1,
      results: [
        {
          ...TEST_ITEM,
          region: 'us-east-1',
          az: 'us-east-1f',
          labels: {
            az: 'us-east-1f',
            // deliberately disagrees with the region field, to prove which one
            // the Region column renders
            region: 'stale-label-region',
            agent_type: 'qan-mysql-slowlog-agent',
          },
        },
      ],
    });

    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-view-details'));

    const pane = await screen.findByTestId('insight-details-pane');

    const region = screen.getByTestId('details-field-region');
    expect(within(region).getByText('us-east-1')).toBeInTheDocument();
    expect(
      within(region).queryByText('stale-label-region')
    ).not.toBeInTheDocument();

    expect(
      within(pane).getByText('agent_type: qan-mysql-slowlog-agent')
    ).toBeInTheDocument();
    // AZ has no field of its own; it is only reachable as a label chip
    expect(within(pane).getByText('az: us-east-1f')).toBeInTheDocument();
    expect(screen.queryByTestId('details-field-az')).not.toBeInTheDocument();
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

  it('opens the details overlay maximized when deep-linked', async () => {
    renderComponent('/advisors/insights?insight=result-1');

    const pane = await screen.findByTestId('insight-details-pane');
    expect(
      within(pane).getByTestId('CloseFullscreenOutlinedIcon')
    ).toBeInTheDocument();
  });

  it('minimizes the details overlay only when the toggle is clicked', async () => {
    renderComponent('/advisors/insights?insight=result-1');

    const pane = await screen.findByTestId('insight-details-pane');
    fireEvent.click(screen.getByTestId('insight-details-maximize'));

    // the toggle drops it to the 60vh peek height
    expect(
      within(pane).getByTestId('OpenInFullOutlinedIcon')
    ).toBeInTheDocument();

    // the choice does not stick: the next open is maximized again
    fireEvent.click(screen.getByTestId('insight-details-close'));
    await waitForRows();
    fireEvent.dblClick(screen.getByTestId('insight-row-result-1'));

    const reopened = await screen.findByTestId('insight-details-pane');
    expect(
      within(reopened).getByTestId('CloseFullscreenOutlinedIcon')
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
      '  Run ID: run-1\n' +
      '  Check Name: mysql_version_check\n' +
      '  Category: Configuration\n' +
      '  Sub category: Version\n' +
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

  it('copies a shareable URL that keeps the current filters', async () => {
    renderComponent('/advisors/insights?service=mysql-prod&page=2');

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-copy-url'));

    const { origin, pathname } = window.location;
    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        `${origin}${pathname}?service=mysql-prod&page=2&insight=result-1`
      )
    );
    expect(
      await screen.findByText(Messages.success.urlCopied)
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

  it("dims rows whose check is disabled for that row's service", async () => {
    vi.mocked(advisorsApi.listAdvisors).mockResolvedValue(
      withDisabledService('mysql_version_check', 'service-1')
    );
    renderComponent();

    await waitForRows();

    await waitFor(() =>
      expect(screen.getByTestId('insight-row-result-1')).toHaveAttribute(
        'data-check-disabled',
        'true'
      )
    );
  });

  it("disables the check only for the row's service", async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    const item = await screen.findByTestId('action-disable-check');
    // the item stays disabled until the advisors list resolves
    await waitFor(() =>
      expect(item).not.toHaveAttribute('aria-disabled', 'true')
    );
    expect(item).toHaveTextContent(Messages.actions.disableCheck);

    const advisorsCalls = vi.mocked(advisorsApi.listAdvisors).mock.calls.length;
    fireEvent.click(item);

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [
          {
            name: 'mysql_version_check',
            serviceIds: ['service-1'],
            enable: false,
          },
        ],
        expect.anything()
      )
    );
    expect(
      await screen.findByText(
        Messages.success.checkDisabled('MySQL version check', 'mysql-prod')
      )
    ).toBeInTheDocument();
    // the advisors list refetches, so the menu reflects the new state
    await waitFor(() =>
      expect(
        vi.mocked(advisorsApi.listAdvisors).mock.calls.length
      ).toBeGreaterThan(advisorsCalls)
    );
  });

  it('offers re-enabling a check disabled for that service', async () => {
    vi.mocked(advisorsApi.listAdvisors).mockResolvedValue(
      withDisabledService('mysql_version_check', 'service-1')
    );
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    const item = await screen.findByTestId('action-disable-check');
    // the label flips only once the advisors list resolves
    await waitFor(() =>
      expect(item).toHaveTextContent(Messages.actions.enableCheck)
    );

    fireEvent.click(item);

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [
          {
            name: 'mysql_version_check',
            serviceIds: ['service-1'],
            enable: true,
          },
        ],
        expect.anything()
      )
    );
  });

  it('disables re-run and the per-service toggle for globally disabled checks', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-2-actions'));
    await screen.findByTestId('action-view-details');

    expect(screen.getByTestId('action-rerun-now')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
    // a check that is off everywhere is re-enabled from the Advisor checks page
    expect(screen.getByTestId('action-disable-check')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
    expect(screen.getByTestId('action-filter-by-run-id')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it("disables re-run when the check is off for the row's service", async () => {
    vi.mocked(advisorsApi.listAdvisors).mockResolvedValue(
      withDisabledService('mysql_version_check', 'service-1')
    );
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    await screen.findByTestId('action-view-details');

    expect(screen.getByTestId('action-rerun-now')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it('filters by run ID from the row menu, populates the input, then clears', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-filter-by-run-id'));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
        expect.objectContaining({ runId: 'run-1', pageIndex: 0 })
      )
    );

    // the run-id input reflects the applied filter
    const input = within(screen.getByTestId('run-id-filter')).getByRole(
      'textbox',
      { hidden: true }
    );
    expect(input).toHaveValue('run-1');

    fireEvent.click(screen.getByTestId('clear-filters'));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenLastCalledWith(
        expect.objectContaining({ runId: undefined })
      )
    );
    expect(input).toHaveValue('');
  });

  it('marks rows with documentation using a non-clickable indicator', async () => {
    renderComponent();

    await waitForRows();

    const indicators = screen.getAllByTestId('insight-read-more-indicator');
    expect(indicators).toHaveLength(2);
    // it replaced a "Read more" link: it must not be a click target itself,
    // the real link lives in the details pane
    expect(screen.queryByRole('link', { name: /read more/i })).toBeNull();
    expect(indicators[0].closest('a')).toBeNull();
  });

  it('omits the documentation indicator when the insight has no URL', async () => {
    vi.mocked(advisorsApi.listInsights).mockResolvedValue({
      totalItems: 1,
      totalPages: 1,
      results: [{ ...TEST_ITEM, readMoreUrl: '' }],
    });
    renderComponent();

    await waitForRows();

    expect(
      screen.queryByTestId('insight-read-more-indicator')
    ).not.toBeInTheDocument();
  });

  it('filters by check name from the row menu, then clears', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-filter-by-check-name'));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
        expect.objectContaining({
          checkName: 'mysql_version_check',
          pageIndex: 0,
        })
      )
    );

    fireEvent.click(screen.getByTestId('clear-filters'));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenLastCalledWith(
        expect.objectContaining({ checkName: undefined })
      )
    );
  });

  it('scopes bulk mark-as-read to the check-name filter', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    fireEvent.click(await screen.findByTestId('action-filter-by-check-name'));

    await waitFor(() =>
      expect(screen.getByTestId('mark-filtered-read')).toBeEnabled()
    );
    fireEvent.click(screen.getByTestId('mark-filtered-read'));

    // the filter must reach the API, or the bulk action would mark far more
    // insights read than the filtered list shows
    await waitFor(() =>
      expect(advisorsApi.markInsightsRead).toHaveBeenCalledWith(
        { filters: { checkName: 'mysql_version_check' }, isRead: true },
        expect.anything()
      )
    );
  });

  it('filters by a run ID typed into the input and committed with Enter', async () => {
    renderComponent();

    await waitForRows();

    const input = within(screen.getByTestId('run-id-filter')).getByRole(
      'textbox',
      { hidden: true }
    );
    fireEvent.change(input, { target: { value: 'run-77' } });
    fireEvent.keyDown(input, { key: 'Enter' });

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenCalledWith(
        expect.objectContaining({ runId: 'run-77', pageIndex: 0 })
      )
    );
  });

  it('re-runs the check from the row menu and links to the new run', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('insight-result-1-actions'));
    const rerun = await screen.findByTestId('action-rerun-now');
    // the item stays disabled until the advisors list resolves
    await waitFor(() =>
      expect(rerun).not.toHaveAttribute('aria-disabled', 'true')
    );
    fireEvent.click(rerun);

    await waitFor(() =>
      expect(advisorsApi.startAdvisorChecks).toHaveBeenCalledWith(
        { names: ['mysql_version_check'], serviceIds: ['service-1'] },
        expect.anything()
      )
    );
    expect(
      await screen.findByText(
        Messages.success.rerunStarted('MySQL version check', 'mysql-prod')
      )
    ).toBeInTheDocument();
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('run-123');

    fireEvent.click(screen.getByTestId('view-run-results'));

    await waitFor(() =>
      expect(advisorsApi.listInsights).toHaveBeenLastCalledWith(
        expect.objectContaining({ runId: 'run-123' })
      )
    );
  });
});
