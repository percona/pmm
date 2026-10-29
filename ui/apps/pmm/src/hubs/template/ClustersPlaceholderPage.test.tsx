import { screen, within } from '@testing-library/react';
import { hasPeakHub, renderHub } from '../testUtils';
import { ClustersPlaceholderPage } from './ClustersPlaceholderPage';

describe.skipIf(!hasPeakHub)('ClustersPlaceholderPage', () => {
  it('says the page is in design and lists the clusters the link asked for', () => {
    renderHub(<ClustersPlaceholderPage />, {
      path: '/mongodb/clusters?clusters=cart-eu,users-eu,search-eu',
    });

    expect(
      screen.getByRole('heading', {
        level: 1,
        name: 'Clusters page (in design)',
      })
    ).toBeInTheDocument();
    expect(
      screen.getByText('The link asked for 3 clusters:')
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole('list'))
        .getAllByRole('listitem')
        .map((item) => item.textContent)
    ).toEqual(['cart-eu', 'users-eu', 'search-eu']);
  });

  it('says so when the link names no cluster', () => {
    renderHub(<ClustersPlaceholderPage />, { path: '/mongodb/clusters' });

    expect(
      screen.getByText('The link asked for no cluster in particular.')
    ).toBeInTheDocument();
    expect(screen.queryByRole('list')).toBeNull();
  });
});
