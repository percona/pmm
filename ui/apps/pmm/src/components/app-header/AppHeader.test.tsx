import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { NavigationContext } from 'contexts/navigation';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { AppHeader, type AppHeaderProps } from './AppHeader';
import { Messages } from './AppHeader.messages';

const { mockCreateShortUrl } = vi.hoisted(() => ({
  mockCreateShortUrl: vi.fn(),
}));

vi.mock('api/short-urls', () => ({
  createShortUrl: mockCreateShortUrl,
}));

const renderHeader = (
  props: Partial<AppHeaderProps> = {},
  { starredApps = [] as string[], toggleStarredApp = vi.fn() } = {}
) => {
  const ui = (apps: string[]) =>
    wrapWithSnackbarProvider(
      wrapWithQueryProvider(
        <NavigationContext.Provider
          value={{
            navTree: [],
            navOpen: true,
            setNavOpen: vi.fn(),
            starredApps: apps,
            toggleStarredApp,
          }}
        >
          <AppHeader title="MySQL backups" inset="page" {...props} />
        </NavigationContext.Provider>
      )
    );
  const view = render(ui(starredApps));

  return {
    ...view,
    toggleStarredApp,
    rerenderWith: (apps: string[]) => view.rerender(ui(apps)),
  };
};

const atPath = (pathname: string) =>
  Object.defineProperty(window, 'location', {
    value: {
      ...window.location,
      pathname,
      search: '',
      hash: '',
      href: `http://localhost${pathname}`,
    },
    writable: true,
    configurable: true,
  });

describe('AppHeader', () => {
  const original = window.location;

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: original,
      writable: true,
      configurable: true,
    });
    mockCreateShortUrl.mockReset();
  });

  it('names the app as the page h1, with a link to share and no star', () => {
    renderHeader();

    expect(
      screen.getByRole('heading', { level: 1, name: 'MySQL backups' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: Messages.share })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: Messages.star })
    ).not.toBeInTheDocument();
  });

  it('stars an app under Apps, then offers to take it out again', () => {
    const { toggleStarredApp, rerenderWith } = renderHeader({
      starId: 'app-mysql-backups',
    });

    fireEvent.click(screen.getByRole('button', { name: Messages.star }));

    expect(toggleStarredApp).toHaveBeenCalledWith('app-mysql-backups');
    expect(screen.getByText(Messages.starred)).toBeInTheDocument();

    rerenderWith(['app-mysql-backups']);
    fireEvent.click(screen.getByRole('button', { name: Messages.unstar }));

    expect(screen.getByText(Messages.unstarred)).toBeInTheDocument();
  });

  it('copies the address of a PMM page as it is', async () => {
    atPath('/pmm-ui/extensions/mysql-backups');
    renderHeader();

    fireEvent.click(screen.getByRole('button', { name: Messages.share }));

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'http://localhost/pmm-ui/extensions/mysql-backups'
      )
    );
    expect(mockCreateShortUrl).not.toHaveBeenCalled();
    expect(await screen.findByText(Messages.copied)).toBeInTheDocument();
  });

  it('copies a Grafana short URL on a Grafana page, which keeps its state', async () => {
    mockCreateShortUrl.mockResolvedValue({
      uid: '1',
      url: 'https://pmm.example/goto/1',
    });
    atPath('/pmm-ui/graph/d/pmm-qan/pmm-query-analytics');
    renderHeader({ title: 'Query analytics' });

    fireEvent.click(screen.getByRole('button', { name: Messages.share }));

    await waitFor(() =>
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
        'https://pmm.example/goto/1'
      )
    );
    expect(await screen.findByText(Messages.copied)).toBeInTheDocument();
  });

  it('says when the link could not be copied', async () => {
    mockCreateShortUrl.mockRejectedValue(new Error('no short URL'));
    atPath('/pmm-ui/graph/backup/inventory');
    renderHeader();

    fireEvent.click(screen.getByRole('button', { name: Messages.share }));

    expect(await screen.findByText(Messages.copyFailed)).toBeInTheDocument();
  });
});
