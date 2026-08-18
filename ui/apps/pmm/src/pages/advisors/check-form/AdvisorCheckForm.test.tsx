import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AxiosError, AxiosResponse } from 'axios';
import * as advisorsApi from 'api/advisors';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import {
  AdvisorCheck,
  AdvisorTechnology,
  AdvisorInterval,
  TestAdvisorCheckResult,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { Messages } from '../check-test/CheckTest.messages';
import { AdvisorCheckForm } from './AdvisorCheckForm';

vi.mock('api/advisors');

const SOURCE_CHECK: AdvisorCheck = {
  name: 'mysql_version_check',
  enabled: true,
  summary: 'MySQL version check',
  description: 'Warns if MySQL version is EOL',
  category: 'Configuration',
  technology: AdvisorTechnology.mysql,
  interval: AdvisorInterval.standard,
  userDefined: false,
  queries: [{ type: 'MYSQL_SHOW', query: 'version' }],
  script: 'def check_context(docs, context):\n    return []',
};

const TEST_RESULT: TestAdvisorCheckResult = {
  summary: 'MySQL is outdated',
  checkName: 'custom_mysql_version_check',
  description: 'Upgrade MySQL',
  readMoreUrl: '',
  severity: Severity.warning,
  labels: {},
  serviceName: 'mysql-svc-1',
  serviceId: 'svc-1',
};

const renderForm = (onClose = vi.fn()) =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        <AdvisorCheckForm
          open
          mode="clone"
          checkName={SOURCE_CHECK.name}
          onClose={onClose}
        />
      )
    )
  );

const pickTestService = async (serviceName: string) => {
  const picker = await screen.findByTestId('advisor-check-form-test-service');
  const input = within(picker).getByRole('combobox');
  // ArrowDown opens the MUI Autocomplete popup reliably in jsdom
  fireEvent.keyDown(input, { key: 'ArrowDown' });
  const listbox = await screen.findByRole('listbox', { hidden: true });
  fireEvent.click(within(listbox).getByText(serviceName));
};

const waitForPrefill = async () => {
  await waitFor(() =>
    expect(screen.getByTestId('check-name')).toHaveValue(
      `custom_${SOURCE_CHECK.name}`
    )
  );
};

describe('AdvisorCheckForm test run', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(advisorsApi.getAdvisorCheck).mockResolvedValue(SOURCE_CHECK);
    vi.mocked(advisorsApi.listAdvisorCheckTestTargets).mockResolvedValue([
      { serviceId: 'svc-1', serviceName: 'mysql-svc-1' },
      { serviceId: 'svc-2', serviceName: 'mysql-svc-2' },
    ]);
    vi.mocked(advisorsApi.testAdvisorCheck).mockResolvedValue({
      results: [TEST_RESULT],
      scriptOutput: 'version = 8.2.6',
    });
  });

  it('keeps the test button disabled until a service is picked', async () => {
    renderForm();
    await waitForPrefill();

    expect(screen.getByTestId('advisor-check-form-test')).toBeDisabled();

    await pickTestService('mysql-svc-1');

    expect(screen.getByTestId('advisor-check-form-test')).toBeEnabled();
  });

  it('tests the current definition against the picked service and shows the results', async () => {
    renderForm();
    await waitForPrefill();

    await pickTestService('mysql-svc-1');
    fireEvent.click(screen.getByTestId('advisor-check-form-test'));

    await waitFor(() =>
      expect(advisorsApi.testAdvisorCheck).toHaveBeenCalledWith(
        {
          check: {
            name: `custom_${SOURCE_CHECK.name}`,
            summary: SOURCE_CHECK.summary,
            description: SOURCE_CHECK.description,
            category: SOURCE_CHECK.category,
            technology: SOURCE_CHECK.technology,
            interval: SOURCE_CHECK.interval,
            queries: [{ type: 'MYSQL_SHOW', query: 'version' }],
            script: SOURCE_CHECK.script,
          },
          serviceId: 'svc-1',
        },
        expect.anything()
      )
    );

    const results = await screen.findByTestId(
      'advisor-check-form-test-results'
    );
    // the run status and the findings count are separate facts
    expect(
      within(results).getByTestId('advisor-check-form-test-status')
    ).toHaveTextContent(Messages.testSuccess);
    expect(
      within(results).getByText(`· ${Messages.testFindings(1)}`)
    ).toBeInTheDocument();
    expect(
      screen.getByTestId('advisor-check-form-test-output')
    ).toBeInTheDocument();
    // the script's print() output is shown in its own section
    expect(
      screen.getByTestId('advisor-check-form-test-script-output')
    ).toHaveTextContent('version = 8.2.6');
  });

  it('omits the script output section when the script printed nothing', async () => {
    vi.mocked(advisorsApi.testAdvisorCheck).mockResolvedValue({
      results: [TEST_RESULT],
    });

    renderForm();
    await waitForPrefill();

    await pickTestService('mysql-svc-1');
    fireEvent.click(screen.getByTestId('advisor-check-form-test'));
    await screen.findByTestId('advisor-check-form-test-results');

    expect(
      screen.queryByTestId('advisor-check-form-test-script-output')
    ).not.toBeInTheDocument();
  });

  it('shows the backend error inside the results panel when the test fails', async () => {
    const error = new AxiosError('Request failed');
    error.response = {
      data: { message: 'invalid advisor check: unknown query type' },
    } as AxiosResponse;
    vi.mocked(advisorsApi.testAdvisorCheck).mockRejectedValue(error);

    renderForm();
    await waitForPrefill();

    await pickTestService('mysql-svc-2');
    fireEvent.click(screen.getByTestId('advisor-check-form-test'));

    expect(
      await screen.findByTestId('advisor-check-form-test-error')
    ).toHaveTextContent('invalid advisor check: unknown query type');
    expect(
      screen.getByTestId('advisor-check-form-test-status')
    ).toHaveTextContent(Messages.testFailure);
  });

  it('clears the results panel with its close button', async () => {
    renderForm();
    await waitForPrefill();

    await pickTestService('mysql-svc-1');
    fireEvent.click(screen.getByTestId('advisor-check-form-test'));
    await screen.findByTestId('advisor-check-form-test-results');

    fireEvent.click(
      screen.getByTestId('advisor-check-form-test-results-close')
    );

    expect(
      screen.queryByTestId('advisor-check-form-test-results')
    ).not.toBeInTheDocument();
  });
});
