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
import { AdvisorCheckTriggeredBy, AdvisorRun } from 'types/advisors.types';
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
  startedAt: '2026-08-04T19:57:28Z',
  finishedAt: '2026-08-04T19:59:35Z',
  checksCount: 107,
  servicesCount: 3,
  findingsCount: 28,
  errorsCount: 1,
  severityCounts: [
    { severity: Severity.error, count: 4 },
    { severity: Severity.warning, count: 22 },
    { severity: Severity.info, count: 2 },
  ],
};

const RUNNING_RUN: AdvisorRun = {
  id: 'run-open',
  triggeredBy: AdvisorCheckTriggeredBy.scheduler,
  startedAt: '2026-08-04T20:05:00Z',
  finishedAt: null,
  checksCount: 0,
  servicesCount: 0,
  findingsCount: 0,
  errorsCount: 0,
  severityCounts: [],
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
    expect(screen.getByText('107')).toBeInTheDocument();
    expect(screen.getByText('28')).toBeInTheDocument();
    expect(screen.getByText('User')).toBeInTheDocument();
    // severity breakdown, in the order the API returned it
    expect(screen.getByText('4 Error')).toBeInTheDocument();
    expect(screen.getByText('22 Warning')).toBeInTheDocument();
    expect(screen.getByText('2 Info')).toBeInTheDocument();
  });

  it('shows a run with no completion as still running', async () => {
    renderComponent();

    await waitForRows();

    expect(screen.getByText(Messages.running)).toBeInTheDocument();
    expect(screen.getByTestId('run-in-progress')).toBeInTheDocument();
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

  it('polls once a minute while runs are in flight, then stops', async () => {
    // two concurrent runs: the interval is per query, so still one request
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 2,
      totalPages: 1,
      results: [RUNNING_RUN, { ...RUNNING_RUN, id: 'run-open-2' }],
    });

    vi.useFakeTimers();
    try {
      renderComponent();
      // let the initial fetch resolve while the clock is still frozen
      await act(() => vi.advanceTimersByTimeAsync(0));
      expect(advisorsApi.listRuns).toHaveBeenCalledTimes(1);

      await act(() => vi.advanceTimersByTimeAsync(60_000));
      expect(advisorsApi.listRuns).toHaveBeenCalledTimes(2);

      vi.mocked(advisorsApi.listRuns).mockResolvedValue({
        totalItems: 2,
        totalPages: 1,
        results: [FINISHED_RUN],
      });

      await act(() => vi.advanceTimersByTimeAsync(60_000));
      expect(advisorsApi.listRuns).toHaveBeenCalledTimes(3);

      // nothing is running anymore, so the polling stops
      await act(() => vi.advanceTimersByTimeAsync(180_000));
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
