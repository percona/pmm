import { render, screen } from '@testing-library/react';
import { wrapWithQueryProvider } from 'utils/testUtils';
import Header from './Header';

const mocks = vi.hoisted(() => ({
  useHeader: vi.fn(),
}));

vi.mock('hooks/useHeader', async (importOriginal) => ({
  ...(await importOriginal()),
  useHeader: mocks.useHeader,
}));

describe('Header', () => {
  beforeEach(() => {
    mocks.useHeader.mockReset();
  });

  it('renders the app header of the app the page belongs to', () => {
    mocks.useHeader.mockReturnValueOnce({
      visible: true,
      page: {
        app: { id: 'app-mysql-backups', text: 'MySQL backups' },
        starrable: true,
      },
      inset: 'page',
    });

    render(wrapWithQueryProvider(<Header />));

    expect(
      screen.getByRole('heading', { level: 1, name: 'MySQL backups' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Add to Starred' })
    ).toBeInTheDocument();
  });

  it('leaves the star off an app with a place of its own', () => {
    mocks.useHeader.mockReturnValueOnce({
      visible: true,
      page: { app: { id: 'qan', text: 'Query analytics' }, starrable: false },
      inset: 'frame',
    });

    render(wrapWithQueryProvider(<Header />));

    expect(
      screen.queryByRole('button', { name: 'Add to Starred' })
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Copy link' })
    ).toBeInTheDocument();
  });

  it('renders nothing outside an app', () => {
    mocks.useHeader.mockReturnValueOnce({
      visible: false,
      page: undefined,
      inset: 'page',
    });

    const { container } = render(wrapWithQueryProvider(<Header />));

    expect(container).toBeEmptyDOMElement();
  });
});
