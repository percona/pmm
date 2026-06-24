import { render } from '@testing-library/react';
import { UpdateLog } from './UpdateLog';
import { wrapWithUpdatesProvider } from 'utils/testUtils';
import { UpdateStatus } from 'types/updates.types';
import { hardReloadPage } from 'utils/dom.utils';

const mockUseUpdateLog = vi.fn();

vi.mock('./UpdateLog.hooks', () => ({
  useUpdateLog: () => mockUseUpdateLog(),
}));

vi.mock('utils/dom.utils', async (importOriginal) => {
  const actual = await importOriginal<typeof import('utils/dom.utils')>();

  return {
    ...actual,
    hardReloadPage: vi.fn(),
  };
});

describe('UpdateLog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseUpdateLog.mockReturnValue({
      output: '',
      isDone: false,
    });
  });

  it('hard reloads the page when the update finishes', () => {
    const setStatus = vi.fn();

    mockUseUpdateLog.mockReturnValue({
      output: 'done',
      isDone: true,
    });

    render(
      wrapWithUpdatesProvider(
        <UpdateLog authToken="token" upgradeVersion="3.8.0" />,
        { setStatus }
      )
    );

    expect(setStatus).toHaveBeenCalledWith(UpdateStatus.Completed);
    expect(hardReloadPage).toHaveBeenCalledWith('3.8.0');
  });

  it('does not reload while the update is still running', () => {
    render(
      wrapWithUpdatesProvider(
        <UpdateLog authToken="token" upgradeVersion="3.8.0" />,
        { setStatus: vi.fn() }
      )
    );

    expect(hardReloadPage).not.toHaveBeenCalled();
  });
});
