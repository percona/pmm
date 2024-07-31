import { screen, render } from '@testing-library/react';
import { UpdatesContext, UpdatesContextProps } from 'contexts/updates';
import { Footer } from './Footer';
import { UpdateStatus } from 'types/updates.types';
import { Messages } from './Footer.messages';

const renderWithProvider = (value: Partial<UpdatesContextProps> = {}) =>
  render(
    <UpdatesContext.Provider
      value={{
        inProgress: false,
        isLoading: false,
        status: UpdateStatus.UpToDate,
        recheck: () => {},
        setStatus: () => {},
        versionInfo: {
          installed: {
            version: '3.0.0',
            fullVersion: '3.0.0',
            timestamp: '2024-07-23T00:00:00Z',
          },
          latest: {
            version: '3.0.0',
            tag: '',
            timestamp: null,
          },
          updateAvailable: false,
          latestNewsUrl: 'https://per.co.na/pmm/3.0.0',
          lastCheck: '2024-07-30T10:34:05.886739003Z',
        },
        ...value,
      }}
    >
      <Footer />
    </UpdatesContext.Provider>
  );

describe('Footer', () => {
  it("doesnt't show when version info is not available", () => {
    renderWithProvider({
      versionInfo: undefined,
    });
  });

  it('shows  correct checked date', () => {
    renderWithProvider();

    expect('Checked on: 2024/07/30');
  });

  it('shows in progress message', () => {
    renderWithProvider({
      inProgress: true,
    });

    expect(screen.getByText(Messages.inProgress));
  });
});
