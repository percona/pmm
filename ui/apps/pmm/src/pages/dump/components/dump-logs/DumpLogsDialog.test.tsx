import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TEST_DUMP } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { DumpLogsDialog } from './DumpLogsDialog';

const mocks = vi.hoisted(() => ({
  useDumpLogs: vi.fn(),
}));

vi.mock('hooks/api/useDump', () => ({
  useDumpLogs: mocks.useDumpLogs,
}));

const renderDialog = () =>
  render(
    <TestWrapper>
      <DumpLogsDialog dump={TEST_DUMP} open onClose={vi.fn()} />
    </TestWrapper>
  );

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

    renderDialog();

    expect(await screen.findByText(/first line\s+second line/)).toBeVisible();
    fireEvent.click(screen.getByTestId('ContentCopyOutlinedIcon'));

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'first line\nsecond line'
      )
    );
  });

  it('titles the dialog without the dataset id', () => {
    mocks.useDumpLogs.mockReturnValue({
      data: { logs: [], end: true },
      isError: false,
    });

    renderDialog();

    expect(screen.getByText('Dataset logs')).toBeVisible();
    expect(
      screen.getByText('Oct 26, 18:39 – Oct 27, 06:39 · mysql-1, pg-1')
    ).toBeVisible();
    expect(screen.queryByText(new RegExp(TEST_DUMP.dumpId))).toBeNull();
  });

  it('says when the export is still running', () => {
    mocks.useDumpLogs.mockReturnValue({
      data: { logs: [{ chunkId: 1, data: 'exporting' }], end: false },
      isError: false,
    });

    renderDialog();

    expect(
      screen.getByText("Export running. New lines appear as they're written.")
    ).toBeVisible();
  });

  it('keeps showing progress while an active export has no logs', () => {
    mocks.useDumpLogs.mockReturnValue({
      data: { logs: [], end: false },
      isError: false,
    });

    renderDialog();

    expect(screen.getByLabelText('Loading logs')).toBeVisible();
    expect(screen.queryByText('No logs available.')).not.toBeInTheDocument();
  });

  it('says when a finished export left no logs', () => {
    mocks.useDumpLogs.mockReturnValue({
      data: { logs: [], end: true },
      isError: false,
    });

    renderDialog();

    expect(screen.getByText('No logs available.')).toBeVisible();
    expect(screen.getByText('Export finished.')).toBeVisible();
    expect(screen.queryByLabelText('Loading logs')).not.toBeInTheDocument();
  });
});
