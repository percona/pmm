import { fireEvent, render, screen } from '@testing-library/react';
import { DumpStatus as Status } from 'types/dump.types';
import { TEST_DUMP } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { DumpStatus } from './DumpStatus';

const renderStatus = (
  status: Status,
  { onShowLogs = vi.fn(), createdAt = '2026-10-27T06:36:00Z' } = {}
) =>
  render(
    <TestWrapper>
      <DumpStatus
        dump={{ ...TEST_DUMP, status, createdAt }}
        onShowLogs={onShowLogs}
      />
    </TestWrapper>
  );

describe('DumpStatus', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-27T06:39:00Z'));
  });

  afterEach(() => vi.useRealTimers());

  it('shows Ready for a successful dataset', () => {
    renderStatus(Status.Success);

    expect(screen.getByText('Ready')).toBeVisible();
    expect(screen.queryByText(/started/)).not.toBeInTheDocument();
  });

  it('shows when a creating dataset started', () => {
    renderStatus(Status.InProgress);

    expect(screen.getByText('Creating…')).toBeVisible();
    expect(screen.getByText('started 3 minutes ago')).toBeVisible();
  });

  it('leaves out the start time when it cannot be read', () => {
    renderStatus(Status.InProgress, { createdAt: '' });

    expect(screen.getByText('Creating…')).toBeVisible();
    expect(screen.queryByText(/started/)).not.toBeInTheDocument();
  });

  it('opens the logs from a failed dataset', () => {
    const onShowLogs = vi.fn();
    renderStatus(Status.Error, { onShowLogs });

    expect(screen.getByText('Failed')).toBeVisible();
    fireEvent.click(
      screen.getByRole('button', { name: 'See what went wrong' })
    );

    expect(onShowLogs).toHaveBeenCalledTimes(1);
  });

  it('shows Unknown for an unspecified status', () => {
    renderStatus(Status.Unspecified);

    expect(screen.getByText('Unknown')).toBeVisible();
  });
});
