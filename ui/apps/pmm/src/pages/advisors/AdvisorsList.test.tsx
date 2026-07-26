import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import AdvisorsList from './AdvisorsList';
import { Messages } from './AdvisorsList.messages';
import * as advisorsApi from 'api/advisors';
import * as servicesApi from 'api/services';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import {
  Advisor,
  AdvisorCheck,
  AdvisorFamily,
  AdvisorInterval,
} from 'types/advisors.types';

vi.mock('api/advisors');
vi.mock('api/services');

const TEST_ADVISORS: Advisor[] = [
  {
    category: 'Configuration',
    subcategory: 'Version',
    checks: [
      {
        name: 'mysql_version_check',
        enabled: true,
        summary: 'MySQL version check',
        description: 'Warns if MySQL version is EOL',
        interval: AdvisorInterval.standard,
        family: AdvisorFamily.mysql,
        category: 'Configuration',
        subcategory: 'Version',
        userDefined: false,
      },
    ],
  },
  {
    category: 'Security',
    subcategory: 'Authentication',
    checks: [
      {
        name: 'postgresql_super_role',
        enabled: true,
        summary: 'PostgreSQL super role',
        description: 'Detects users with the SUPER role',
        interval: AdvisorInterval.rare,
        family: AdvisorFamily.postgresql,
        category: 'Security',
        subcategory: 'Authentication',
        userDefined: true,
      },
      {
        name: 'postgresql_disabled_check',
        enabled: false,
        summary: 'PostgreSQL disabled check',
        description: 'A disabled check',
        interval: AdvisorInterval.standard,
        family: AdvisorFamily.postgresql,
        category: 'Security',
        subcategory: 'Authentication',
        userDefined: false,
      },
    ],
  },
];

// the full definition returned by getAdvisorCheck (queries + script populated)
const FULL_CHECK: AdvisorCheck = {
  ...TEST_ADVISORS[0].checks[0],
  queries: [{ type: 'MYSQL_SHOW', query: 'version' }],
  script: 'print("hi")',
};

const renderComponent = (initialEntry = '/advisors') =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithUserProvider(
          wrapWithRouter(<AdvisorsList />, { initialEntries: [initialEntry] })
        )
      )
    )
  );

const waitForRows = async () => {
  await waitFor(() =>
    expect(screen.getByText('MySQL version check')).toBeInTheDocument()
  );
};

const selectFilterOption = async (filterTestId: string, option: string) => {
  fireEvent.mouseDown(
    within(screen.getByTestId(filterTestId)).getByRole('combobox')
  );

  // 'hidden' skips the visibility computation, which crashes in jsdom:
  // nwsapi expands ':scope' with the listbox's unescaped React useId
  const listbox = await screen.findByRole('listbox', { hidden: true });
  fireEvent.click(within(listbox).getByText(option));
};

describe('AdvisorsList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(advisorsApi.listAdvisors).mockResolvedValue(TEST_ADVISORS);
    vi.mocked(advisorsApi.startAdvisorChecks).mockResolvedValue('run-123');
    vi.mocked(advisorsApi.changeAdvisorChecks).mockResolvedValue();
    vi.mocked(advisorsApi.getAdvisorCheck).mockResolvedValue(FULL_CHECK);
    vi.mocked(advisorsApi.listAdvisorCheckTestTargets).mockResolvedValue([]);
    vi.mocked(advisorsApi.testAdvisorCheck).mockResolvedValue({ results: [] });
    vi.mocked(advisorsApi.deleteAdvisorCheck).mockResolvedValue();
    vi.mocked(servicesApi.listServices).mockResolvedValue({ mysql: [] });
  });

  it('opens the check details overlay on row double-click and shows the code', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.dblClick(screen.getByTestId('advisor-row-mysql_version_check'));

    const pane = await screen.findByTestId('check-details-pane');
    // the check's fields are shown
    expect(
      within(screen.getByTestId('check-details-field-check-name')).getByText(
        'mysql_version_check'
      )
    ).toBeInTheDocument();
    // double-click opens the pane maximized
    expect(
      within(pane).getByTestId('CloseFullscreenOutlinedIcon')
    ).toBeInTheDocument();

    // the definition is fetched lazily; the script shows in the read-only
    // editor (the testid sits on the editor container, the value on its
    // textarea)
    const editor = await screen.findByTestId('check-code');
    expect(advisorsApi.getAdvisorCheck).toHaveBeenCalledWith(
      'mysql_version_check'
    );
    expect(within(editor).getByRole('textbox')).toHaveValue('print("hi")');

    // Copy button copies the code
    fireEvent.click(within(pane).getByTestId('check-code-copy'));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('print("hi")');
  });

  it('tests the check against a picked service from the details pane', async () => {
    // the backend decides eligibility; the picker offers its list verbatim
    vi.mocked(advisorsApi.listAdvisorCheckTestTargets).mockResolvedValue([
      { serviceId: 'svc-1', serviceName: 'mysql-svc-1' },
    ]);

    renderComponent();

    await waitForRows();

    fireEvent.dblClick(screen.getByTestId('advisor-row-mysql_version_check'));

    const pane = await screen.findByTestId('check-details-pane');
    // wait for the full definition (the Test payload) to load
    await screen.findByTestId('check-code');

    // pick the target service in the footer toolbar
    const picker = within(pane).getByTestId('advisor-check-form-test-service');
    // ArrowDown opens the MUI Autocomplete popup reliably in jsdom
    fireEvent.keyDown(within(picker).getByRole('combobox'), {
      key: 'ArrowDown',
    });
    const listbox = await screen.findByRole('listbox', { hidden: true });
    fireEvent.click(within(listbox).getByText('mysql-svc-1'));

    fireEvent.click(within(pane).getByTestId('advisor-check-form-test'));

    // the saved definition is dry-run as-is against the picked service
    await waitFor(() =>
      expect(advisorsApi.testAdvisorCheck).toHaveBeenCalledWith(
        {
          check: {
            name: FULL_CHECK.name,
            summary: FULL_CHECK.summary,
            description: FULL_CHECK.description,
            category: FULL_CHECK.category,
            subcategory: FULL_CHECK.subcategory,
            family: FULL_CHECK.family,
            interval: FULL_CHECK.interval,
            queries: FULL_CHECK.queries,
            script: FULL_CHECK.script,
          },
          serviceId: 'svc-1',
        },
        expect.anything()
      )
    );
    expect(
      await within(pane).findByTestId('advisor-check-form-test-results')
    ).toBeInTheDocument();
  });

  it('renders all checks', async () => {
    renderComponent();

    await waitForRows();

    expect(screen.getByText('PostgreSQL super role')).toBeInTheDocument();
    expect(screen.getByText('PostgreSQL disabled check')).toBeInTheDocument();
    expect(screen.getByTestId('add-advisor')).toBeInTheDocument();
  });

  it('disables run selected when nothing is filtered', async () => {
    renderComponent();

    await waitForRows();

    expect(screen.getByTestId('run-selected-checks')).toBeDisabled();
  });

  it('filters checks with global search', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.change(screen.getByPlaceholderText(/search/i), {
      target: { value: 'super role' },
    });

    // the matched text gets split by search-highlight marks, so assert via testids
    await waitFor(() =>
      expect(
        screen.queryByTestId('check-mysql_version_check-run')
      ).not.toBeInTheDocument()
    );
    expect(
      screen.getByTestId('check-postgresql_super_role-run')
    ).toBeInTheDocument();
  });

  it('filters checks by the vendor dropdown', async () => {
    renderComponent();

    await waitForRows();

    await selectFilterOption('vendor-filter', 'PostgreSQL');

    await waitFor(() =>
      expect(
        screen.queryByTestId('check-mysql_version_check-run')
      ).not.toBeInTheDocument()
    );
    expect(
      screen.getByTestId('check-postgresql_super_role-run')
    ).toBeInTheDocument();
  });

  it('reads the vendor filter from the URL (deep link)', async () => {
    renderComponent('/advisors?vendor=PostgreSQL');

    await waitFor(() =>
      expect(screen.getByText('PostgreSQL super role')).toBeInTheDocument()
    );

    expect(
      screen.queryByTestId('check-mysql_version_check-run')
    ).not.toBeInTheDocument();
  });

  it('clears an active filter via the clear-filters button', async () => {
    renderComponent();

    await waitForRows();

    expect(screen.getByTestId('clear-filters')).toBeDisabled();

    await selectFilterOption('vendor-filter', 'PostgreSQL');

    await waitFor(() =>
      expect(
        screen.queryByTestId('check-mysql_version_check-run')
      ).not.toBeInTheDocument()
    );
    expect(screen.getByTestId('clear-filters')).not.toBeDisabled();

    fireEvent.click(screen.getByTestId('clear-filters'));

    await waitFor(() =>
      expect(
        screen.getByTestId('check-mysql_version_check-run')
      ).toBeInTheDocument()
    );
    expect(screen.getByTestId('clear-filters')).toBeDisabled();
  });

  it('runs all checks', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('run-all-checks'));

    await waitFor(() =>
      expect(advisorsApi.startAdvisorChecks).toHaveBeenCalledWith(
        [],
        expect.anything()
      )
    );
    expect(
      screen.getByText(Messages.success.checksStarted)
    ).toBeInTheDocument();
    expect(screen.getByTestId('view-run-results')).toBeInTheDocument();
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('run-123');
  });

  it('runs checks matching the global search', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.change(screen.getByPlaceholderText(/search/i), {
      target: { value: 'super role' },
    });

    await waitFor(() =>
      expect(
        screen.queryByTestId('check-mysql_version_check-run')
      ).not.toBeInTheDocument()
    );

    fireEvent.click(screen.getByTestId('run-selected-checks'));

    await waitFor(() =>
      expect(advisorsApi.startAdvisorChecks).toHaveBeenCalledWith(
        ['postgresql_super_role'],
        expect.anything()
      )
    );
  });

  it('disables run selected when only disabled checks are visible', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.change(screen.getByPlaceholderText(/search/i), {
      target: { value: 'A disabled check' },
    });

    await waitFor(() =>
      expect(
        screen.queryByTestId('check-mysql_version_check-run')
      ).not.toBeInTheDocument()
    );

    expect(screen.getByTestId('run-selected-checks')).toBeDisabled();
  });

  it('runs an individual check', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('check-mysql_version_check-run'));

    await waitFor(() =>
      expect(advisorsApi.startAdvisorChecks).toHaveBeenCalledWith(
        ['mysql_version_check'],
        expect.anything()
      )
    );
  });

  it('disables the run action for disabled checks', async () => {
    renderComponent();

    await waitForRows();

    expect(
      screen.getByTestId('check-postgresql_disabled_check-run')
    ).toBeDisabled();
  });

  it('toggles a check', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(
      within(
        screen.getByTestId('check-mysql_version_check-status-switch')
      ).getByRole('switch')
    );

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [{ name: 'mysql_version_check', enable: false }],
        expect.anything()
      )
    );
  });

  it('changes the check interval', async () => {
    renderComponent();

    await waitForRows();

    const select = within(
      screen.getByTestId('check-mysql_version_check-interval-select')
    ).getByRole('combobox');
    fireEvent.mouseDown(select);

    // 'hidden' skips the visibility computation, which crashes in jsdom:
    // nwsapi expands ':scope' with the listbox's unescaped React useId
    const listbox = await screen.findByRole('listbox', { hidden: true });
    fireEvent.click(within(listbox).getByText('Rare'));

    await waitFor(() =>
      expect(advisorsApi.changeAdvisorChecks).toHaveBeenCalledWith(
        [{ name: 'mysql_version_check', interval: AdvisorInterval.rare }],
        expect.anything()
      )
    );
  });

  it('opens the disable-for-services drawer from the row action', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('check-mysql_version_check-services'));

    const drawer = await screen.findByTestId('disable-services-drawer');
    expect(
      within(drawer).getByText('MySQL version check', { exact: false })
    ).toBeInTheDocument();
  });

  it('shows edit/delete only for user-defined checks', async () => {
    renderComponent();

    await waitForRows();

    expect(
      screen.getByTestId('check-postgresql_super_role-edit')
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId('check-mysql_version_check-edit')
    ).not.toBeInTheDocument();
  });

  it('deletes a user-defined check after confirmation', async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('check-postgresql_super_role-delete'));

    const confirm = await screen.findByTestId('delete-advisor-confirm');
    fireEvent.click(confirm);

    await waitFor(() =>
      expect(advisorsApi.deleteAdvisorCheck).toHaveBeenCalledWith(
        'postgresql_super_role',
        expect.anything()
      )
    );
  });
});
