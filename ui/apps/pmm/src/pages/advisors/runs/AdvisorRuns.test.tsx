import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import AdvisorRuns from './AdvisorRuns';
import { Messages } from './AdvisorRuns.messages';
import * as advisorsApi from 'api/advisors';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import {
  AdvisorCheckTriggeredBy,
  AdvisorInterval,
  type AdvisorRun,
  AdvisorRunStatus,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';

vi.mock('api/advisors');

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => ({
  ...(await vi.importActual('react-router-dom')),
  useNavigate: () => mockNavigate,
}));

const FINISHED_RUN: AdvisorRun = {
  id: 'run-finished',
  triggeredBy: AdvisorCheckTriggeredBy.user,
  status: AdvisorRunStatus.completed,
  startedAt: '2026-08-04T19:57:28Z',
  finishedAt: '2026-08-04T19:59:35Z',
  plannedChecksCount: 109,
  plannedServicesCount: 3,
  checksCount: 107,
  servicesCount: 3,
  findingsCount: 28,
  errorsCount: 1,
  severityCounts: [
    { severity: Severity.error, count: 4 },
    { severity: Severity.warning, count: 22 },
    { severity: Severity.info, count: 2 },
  ],
  checkNames: ['mysql_version'],
  serviceIds: ['service-1'],
  intervals: [],
};

const RUNNING_RUN: AdvisorRun = {
  id: 'run-open',
  triggeredBy: AdvisorCheckTriggeredBy.scheduler,
  status: AdvisorRunStatus.running,
  startedAt: '2026-08-04T20:05:00Z',
  finishedAt: null,
  plannedChecksCount: 40,
  plannedServicesCount: 2,
  checksCount: 12,
  servicesCount: 2,
  findingsCount: 0,
  errorsCount: 0,
  severityCounts: [],
  checkNames: [],
  serviceIds: [],
  intervals: [],
};

const renderComponent = (initialEntry = '/advisors/runs') =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithUserProvider(
          wrapWithRouter(<AdvisorRuns />, { initialEntries: [initialEntry] })
        )
      )
    )
  );

const waitForRows = async () =>
  waitFor(() => expect(screen.getByText('Scheduler')).toBeInTheDocument());

describe('AdvisorRuns', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 2,
      totalPages: 1,
      results: [RUNNING_RUN, FINISHED_RUN],
    });
  });

  it('requests the first page by default', async () => {
    renderComponent();

    await waitForRows();

    expect(advisorsApi.listRuns).toHaveBeenCalledWith(
      expect.objectContaining({ pageIndex: 0, pageSize: 50 })
    );
  });

  it('renders a finished run with its duration and totals', async () => {
    renderComponent();

    await waitForRows();

    // 19:57:28 -> 19:59:35 is 2m 07s
    expect(screen.getByText('2m 07s')).toBeInTheDocument();
    expect(screen.getByText('28')).toBeInTheDocument();
    expect(screen.getByText('User')).toBeInTheDocument();
    // severity breakdown, in the order the API returned it
    expect(screen.getByText('4 Error')).toBeInTheDocument();
    expect(screen.getByText('22 Warning')).toBeInTheDocument();
    expect(screen.getByText('2 Info')).toBeInTheDocument();
  });

  it('shows a running run as running', async () => {
    renderComponent();

    await waitForRows();

    expect(screen.getByText(Messages.running)).toBeInTheDocument();
    expect(screen.getByTestId('run-in-progress')).toBeInTheDocument();
  });

  it('shows a queued run as queued', async () => {
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 1,
      totalPages: 1,
      results: [{ ...RUNNING_RUN, status: AdvisorRunStatus.queued }],
    });
    renderComponent();

    await waitForRows();

    expect(screen.getByText(Messages.queued)).toBeInTheDocument();
    expect(screen.getByTestId('run-in-progress')).toBeInTheDocument();
    expect(screen.queryByText(Messages.running)).not.toBeInTheDocument();
  });

  it('shows interrupted and aborted runs instead of their duration', async () => {
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 2,
      totalPages: 1,
      results: [
        {
          ...FINISHED_RUN,
          id: 'run-interrupted',
          status: AdvisorRunStatus.interrupted,
        },
        {
          ...RUNNING_RUN,
          id: 'run-aborted',
          status: AdvisorRunStatus.aborted,
          finishedAt: RUNNING_RUN.startedAt,
        },
      ],
    });
    renderComponent();

    await waitForRows();

    expect(screen.getByTestId('run-interrupted')).toHaveTextContent(
      Messages.interrupted
    );
    expect(screen.getByTestId('run-aborted')).toHaveTextContent(
      Messages.aborted
    );
    // the interrupted run's duration would only cover its last saved insight
    expect(screen.queryByText('2m 07s')).not.toBeInTheDocument();
    expect(screen.queryByTestId('run-in-progress')).not.toBeInTheDocument();
  });

  it('passes the trigger filter to the API and resets the page', async () => {
    renderComponent('/advisors/runs?page=3');

    await waitForRows();

    fireEvent.mouseDown(
      within(screen.getByTestId('triggeredBy-filter')).getByRole('combobox')
    );
    // 'hidden' skips the visibility computation, which crashes in jsdom
    const listbox = await screen.findByRole('listbox', { hidden: true });
    fireEvent.click(within(listbox).getByText('Scheduler'));

    await waitFor(() =>
      expect(advisorsApi.listRuns).toHaveBeenCalledWith(
        expect.objectContaining({
          triggeredBy: AdvisorCheckTriggeredBy.scheduler,
          pageIndex: 0,
        })
      )
    );
  });

  it('reads the trigger filter from the URL', async () => {
    renderComponent(
      '/advisors/runs?triggeredBy=ADVISOR_CHECK_TRIGGERED_BY_USER'
    );

    await waitForRows();

    expect(advisorsApi.listRuns).toHaveBeenCalledWith(
      expect.objectContaining({
        triggeredBy: AdvisorCheckTriggeredBy.user,
      })
    );
  });

  it('keeps Clear filters on screen, disabled until a filter is applied', async () => {
    renderComponent();

    await waitForRows();

    expect(screen.getByTestId('clear-run-filters')).toBeDisabled();
  });

  it('enables Clear filters once a filter is applied, and clears it', async () => {
    renderComponent(
      '/advisors/runs?triggeredBy=ADVISOR_CHECK_TRIGGERED_BY_USER'
    );

    await waitForRows();

    const clear = screen.getByTestId('clear-run-filters');
    expect(clear).toBeEnabled();

    fireEvent.click(clear);

    await waitFor(() =>
      expect(advisorsApi.listRuns).toHaveBeenCalledWith(
        expect.objectContaining({ triggeredBy: undefined })
      )
    );
    expect(screen.getByTestId('clear-run-filters')).toBeDisabled();
  });

  it("deep-links from the row menu to the run's insights", async () => {
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('run-run-finished-actions'));
    fireEvent.click(await screen.findByTestId('action-view-run-insights'));

    expect(mockNavigate).toHaveBeenCalledWith(
      '/advisors/insights?runId=run-finished'
    );
  });

  it('shows what each run covered out of what it planned', async () => {
    renderComponent();

    await waitForRows();

    const [runningChecks, finishedChecks] = screen.getAllByTestId('run-checks');
    const [runningServices, finishedServices] =
      screen.getAllByTestId('run-services');
    expect(runningChecks).toHaveTextContent('12/40');
    expect(runningServices).toHaveTextContent('2/2');
    expect(finishedChecks).toHaveTextContent('107/109');
    expect(finishedServices).toHaveTextContent('3/3');
    // only a finished run that fell short is highlighted
    expect(finishedChecks.querySelector('[data-short]')).not.toBeNull();
    expect(finishedServices.querySelector('[data-short]')).toBeNull();
    expect(runningChecks.querySelector('[data-short]')).toBeNull();
  });

  it('shows a dash for a run without a plan, and 0/0 when there was nothing to check', async () => {
    const noPlan = {
      plannedChecksCount: 0,
      plannedServicesCount: 0,
      checksCount: 0,
      servicesCount: 0,
    };
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 3,
      totalPages: 1,
      results: [
        {
          ...RUNNING_RUN,
          ...noPlan,
          id: 'run-queued',
          status: AdvisorRunStatus.queued,
        },
        {
          ...RUNNING_RUN,
          ...noPlan,
          id: 'run-aborted',
          status: AdvisorRunStatus.aborted,
        },
        { ...FINISHED_RUN, ...noPlan, id: 'run-empty' },
      ],
    });
    renderComponent();

    // two of the runs are scheduled, so waitForRows' single match won't do
    await waitFor(() =>
      expect(screen.getAllByTestId('run-checks')).toHaveLength(3)
    );

    const [queued, aborted, empty] = screen.getAllByTestId('run-checks');
    expect(queued).toHaveTextContent('—');
    expect(aborted).toHaveTextContent('—');
    expect(empty).toHaveTextContent('0/0');
  });

  it('labels a run narrowed to interval groups with them', async () => {
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 2,
      totalPages: 1,
      results: [
        {
          ...RUNNING_RUN,
          intervals: [AdvisorInterval.frequent, AdvisorInterval.standard],
        },
        FINISHED_RUN,
      ],
    });
    renderComponent();

    expect(
      await screen.findByText('Scheduler · Frequent, Standard')
    ).toBeInTheDocument();
  });

  it('runs a run again with the same scope', async () => {
    vi.mocked(advisorsApi.startAdvisorChecks).mockResolvedValue('run-new');
    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('run-run-finished-actions'));
    fireEvent.click(await screen.findByTestId('action-run-again'));

    await waitFor(() =>
      expect(advisorsApi.startAdvisorChecks).toHaveBeenCalledWith(
        {
          names: ['mysql_version'],
          serviceIds: ['service-1'],
          intervals: [],
        },
        expect.anything()
      )
    );
    expect(
      await screen.findByText(Messages.success.checksStarted)
    ).toBeInTheDocument();
  });

  it('polls every 10 seconds while a run is in flight, then stops', async () => {
    // a queued run counts as in flight too
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 1,
      totalPages: 1,
      results: [{ ...RUNNING_RUN, status: AdvisorRunStatus.queued }],
    });

    vi.useFakeTimers();
    try {
      renderComponent();
      // let the initial fetch resolve while the clock is still frozen
      await act(() => vi.advanceTimersByTimeAsync(0));
      expect(advisorsApi.listRuns).toHaveBeenCalledTimes(1);

      await act(() => vi.advanceTimersByTimeAsync(10_000));
      expect(advisorsApi.listRuns).toHaveBeenCalledTimes(2);

      vi.mocked(advisorsApi.listRuns).mockResolvedValue({
        totalItems: 2,
        totalPages: 1,
        results: [FINISHED_RUN],
      });

      await act(() => vi.advanceTimersByTimeAsync(10_000));
      expect(advisorsApi.listRuns).toHaveBeenCalledTimes(3);

      // nothing is running anymore, so the polling stops
      await act(() => vi.advanceTimersByTimeAsync(60_000));
      expect(advisorsApi.listRuns).toHaveBeenCalledTimes(3);
    } finally {
      vi.useRealTimers();
    }
  });

  it('copies the run ID from the row menu', async () => {
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn().mockResolvedValue(undefined) },
    });

    renderComponent();

    await waitForRows();

    fireEvent.click(screen.getByTestId('run-run-finished-actions'));
    fireEvent.click(await screen.findByTestId('action-copy-run-id'));

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('run-finished');
    expect(
      await screen.findByText(Messages.success.runIdCopied)
    ).toBeInTheDocument();
  });
});
