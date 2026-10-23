import { render, screen } from '@testing-library/react';
import { RouterProvider, createMemoryRouter } from 'react-router-dom';
import { advisorsRedirects } from './advisorsRedirects';

const renderAt = (path: string) =>
  render(
    <RouterProvider
      router={createMemoryRouter(
        [
          ...advisorsRedirects,
          { path: 'advisors', element: <div>Catalog</div> },
          { path: 'advisors/insights', element: <div>Insights</div> },
        ],
        { initialEntries: [path] }
      )}
    />
  );

describe('advisorsRedirects', () => {
  it.each([
    ['/graph/advisors', 'Catalog'],
    ['/graph/advisors/security', 'Catalog'],
    ['/graph/advisors/insights', 'Insights'],
    ['/graph/advisors/insights/service-id', 'Insights'],
    ['/graph/pmm-database-checks', 'Insights'],
    ['/graph/pmm-database-checks/failed-checks', 'Insights'],
    ['/graph/pmm-database-checks/all-checks', 'Catalog'],
  ])('redirects %s to the %s page', async (path, page) => {
    renderAt(path);

    expect(await screen.findByText(page)).toBeInTheDocument();
  });
});
