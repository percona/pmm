import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { uploadDumps } from 'api/dump';
import type { Dump } from 'types/dump.types';
import { TEST_DUMP } from 'utils/testStubs';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { SendToSupport } from './SendToSupport';

vi.mock('api/dump');

const renderDialog = (dumps: Dump[]) =>
  render(
    <TestWrapper>
      {wrapWithQueryProvider(
        wrapWithSnackbarProvider(
          <SendToSupport open dumps={dumps} onClose={vi.fn()} />
        )
      )}
    </TestWrapper>
  );

describe('SendToSupport', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(uploadDumps).mockResolvedValue({});
  });

  it('titles the dialog by count', () => {
    const { unmount } = renderDialog([TEST_DUMP]);
    expect(
      screen.getByText('Send this dataset to Percona Support')
    ).toBeVisible();
    unmount();

    renderDialog([TEST_DUMP, { ...TEST_DUMP, dumpId: 'dump-2' }]);
    expect(
      screen.getByText('Send 2 datasets to Percona Support')
    ).toBeVisible();
  });

  it('lists the datasets being sent', () => {
    renderDialog([TEST_DUMP]);

    expect(
      screen.getByText('Oct 26, 18:39 – Oct 27, 06:39 · mysql-1, pg-1')
    ).toBeVisible();
  });

  it('warns about encryption only for encrypted datasets', () => {
    const { unmount } = renderDialog([TEST_DUMP]);
    expect(screen.queryByText(/are encrypted/)).not.toBeInTheDocument();
    unmount();

    renderDialog([{ ...TEST_DUMP, encrypted: true }]);
    expect(screen.getByText(/are encrypted/)).toBeVisible();
  });

  it('uploads the dataset ids with the SFTP details', async () => {
    renderDialog([TEST_DUMP]);

    fireEvent.change(screen.getByTestId('dump-support-address'), {
      target: { value: 'sftp.example.com:22' },
    });
    fireEvent.change(screen.getByTestId('dump-support-user'), {
      target: { value: 'foo' },
    });
    fireEvent.change(screen.getByTestId('dump-support-password'), {
      target: { value: 'secret' },
    });
    fireEvent.change(screen.getByTestId('dump-support-directory'), {
      target: { value: '/upload/' },
    });
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Send' })).toBeEnabled()
    );
    fireEvent.click(screen.getByRole('button', { name: 'Send' }));

    await waitFor(() =>
      expect(vi.mocked(uploadDumps).mock.calls[0][0]).toEqual({
        dumpIds: ['dump-1'],
        sftpParameters: {
          address: 'sftp.example.com:22',
          user: 'foo',
          password: 'secret',
          directory: '/upload/',
        },
      })
    );
  });
});
