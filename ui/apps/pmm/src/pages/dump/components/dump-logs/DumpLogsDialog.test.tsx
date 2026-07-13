import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { DumpLogsDialog } from './DumpLogsDialog';

const mocks = vi.hoisted(() => ({
  useDumpLogs: vi.fn(),
}));

vi.mock('hooks/api/useDump', () => ({
  useDumpLogs: mocks.useDumpLogs,
}));

vi.mock('notistack', () => ({
  enqueueSnackbar: vi.fn(),
}));

describe('DumpLogsDialog', () => {
  it('renders newline-separated chunks and copies them', async () => {
    mocks.useDumpLogs.mockReturnValue({
      data: {
        logs: [
          { chunkId: 1, data: 'first line' },
          { chunkId: 2, data: 'second line' },
        ],
        end: true,
      },
      isError: false,
    });

    render(<DumpLogsDialog dumpId="dump-1" onClose={vi.fn()} />);

    expect(await screen.findByText(/first line\s+second line/)).toBeVisible();
    fireEvent.click(screen.getByRole('button', { name: 'Copy logs' }));

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'first line\nsecond line'
      )
    );
  });

  it('keeps showing progress while an active dump has no logs', () => {
    mocks.useDumpLogs.mockReturnValue({
      data: { logs: [], end: false },
      isError: false,
    });

    render(<DumpLogsDialog dumpId="dump-1" onClose={vi.fn()} />);

    expect(screen.getByLabelText('Loading logs')).toBeVisible();
    expect(screen.queryByText('No logs available.')).not.toBeInTheDocument();
  });
});
