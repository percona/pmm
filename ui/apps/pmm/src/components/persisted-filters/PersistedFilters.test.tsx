import { fireEvent, render, screen } from '@testing-library/react';
import {
  MemoryRouter,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from 'react-router-dom';
import PersistedFilters from './PersistedFilters';

const STORAGE_KEY = 'pmm-ui.test.filters';
const OTHER_STORAGE_KEY = 'pmm-ui.test.other.filters';
const PARAMS = ['status', 'runId', 'pageSize'];

// every query string the page rendered with, in order
const renderedSearches: string[] = [];

const Page = () => {
  const { search } = useLocation();
  const navigate = useNavigate();
  renderedSearches.push(search);

  return (
    <>
      <span data-testid="search">{search}</span>
      <button
        onClick={() =>
          navigate({ search: '?status=failed&page=2' }, { replace: true })
        }
      >
        filter
      </button>
      <button onClick={() => navigate({ search: '' }, { replace: true })}>
        clear
      </button>
      <button onClick={() => navigate('/page')}>page</button>
      <button onClick={() => navigate('/other')}>other</button>
    </>
  );
};

const renderAt = (entry: string) =>
  render(
    <MemoryRouter initialEntries={[entry]}>
      <Routes>
        <Route
          path="/page"
          element={
            <PersistedFilters storageKey={STORAGE_KEY} params={PARAMS}>
              <Page />
            </PersistedFilters>
          }
        />
        <Route
          path="/other"
          element={
            <PersistedFilters storageKey={OTHER_STORAGE_KEY} params={PARAMS}>
              <Page />
            </PersistedFilters>
          }
        />
      </Routes>
    </MemoryRouter>
  );

describe('PersistedFilters', () => {
  beforeEach(() => {
    localStorage.removeItem(STORAGE_KEY);
    localStorage.removeItem(OTHER_STORAGE_KEY);
    renderedSearches.length = 0;
  });

  it('saves only the listed params', () => {
    renderAt('/page?status=failed&runId=r1&page=2&pageSize=25&insight=i1');

    expect(localStorage.getItem(STORAGE_KEY)).toBe(
      'status=failed&runId=r1&pageSize=25'
    );
  });

  it('saves filter changes', () => {
    renderAt('/page');

    fireEvent.click(screen.getByRole('button', { name: 'filter' }));

    expect(localStorage.getItem(STORAGE_KEY)).toBe('status=failed');
  });

  it('restores saved filters on a bare URL before rendering the page', async () => {
    localStorage.setItem(STORAGE_KEY, 'status=failed&runId=r1');

    renderAt('/page');

    expect(await screen.findByTestId('search')).toHaveTextContent(
      '?status=failed&runId=r1'
    );
    expect(renderedSearches).not.toContain('');
  });

  it('leaves a URL with a query string as is', () => {
    localStorage.setItem(STORAGE_KEY, 'status=failed');

    renderAt('/page?page=3');

    expect(screen.getByTestId('search')).toHaveTextContent('?page=3');
    expect(localStorage.getItem(STORAGE_KEY)).toBe('');
  });

  it('does not restore filters again once they are cleared', async () => {
    localStorage.setItem(STORAGE_KEY, 'status=failed');
    renderAt('/page');
    await screen.findByTestId('search');

    fireEvent.click(screen.getByRole('button', { name: 'clear' }));

    expect(screen.getByTestId('search')).toBeEmptyDOMElement();
    expect(localStorage.getItem(STORAGE_KEY)).toBe('');
  });

  it('keeps the filters of each page when moving between pages', async () => {
    localStorage.setItem(OTHER_STORAGE_KEY, 'status=ok');
    renderAt('/page?status=failed');

    fireEvent.click(screen.getByRole('button', { name: 'other' }));
    expect(await screen.findByTestId('search')).toHaveTextContent('?status=ok');

    fireEvent.click(screen.getByRole('button', { name: 'page' }));
    expect(await screen.findByTestId('search')).toHaveTextContent(
      '?status=failed'
    );
  });

  it('restores saved filters when the page links to itself', async () => {
    renderAt('/page?status=failed');

    fireEvent.click(screen.getByRole('button', { name: 'page' }));

    expect(await screen.findByTestId('search')).toHaveTextContent(
      '?status=failed'
    );
    expect(localStorage.getItem(STORAGE_KEY)).toBe('status=failed');
  });
});
