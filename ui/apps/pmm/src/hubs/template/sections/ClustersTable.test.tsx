import { fireEvent, screen, within } from '@testing-library/react';
import { closestTo, renderHub } from '../../testUtils';
import {
  TEST_CLUSTERS,
  TEST_EXPANDED,
  TEST_TREE_COLUMNS,
} from '../hub.testFixtures';
import { ClustersTable } from './ClustersTable';

const renderTable = () =>
  renderHub(
    <ClustersTable
      tableName="hub-test-clusters"
      clusters={TEST_CLUSTERS}
      treeColumns={TEST_TREE_COLUMNS}
      expanded={TEST_EXPANDED}
      label="MongoDB clusters"
    />
  );

// Answers every media query; MUI's useMediaQuery reads it.
const matchAllMedia = (matches: boolean) =>
  vi.spyOn(window, 'matchMedia').mockImplementation(
    (query: string) =>
      ({
        matches,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      }) as MediaQueryList
  );

describe('ClustersTable', () => {
  beforeEach(() => {
    if (!window.matchMedia) {
      window.matchMedia = (() => ({})) as unknown as typeof window.matchMedia;
    }
    matchAllMedia(false);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('is a table named for the technology', () => {
    renderTable();

    expect(
      screen.getByRole('table', { name: 'MongoDB clusters' })
    ).toBeInTheDocument();
  });

  it('makes no cell a tab stop, so Tab goes from link to button', () => {
    renderTable();

    const table = screen.getByRole('table', { name: 'MongoDB clusters' });

    expect(table.querySelectorAll('td[tabindex], th[tabindex]')).toHaveLength(
      0
    );
  });

  it('names the expand column and offers no hidden expand-all', () => {
    renderTable();

    const [expandHeader] = screen.getAllByRole('columnheader');

    expect(expandHeader).toHaveTextContent('Expand');
    expect(within(expandHeader).queryByRole('button')).toBeNull();

    const before = screen.getAllByRole('row').length;

    fireEvent.keyDown(expandHeader, { key: 'Enter' });

    expect(screen.getAllByRole('row')).toHaveLength(before);
  });

  it('opens the problem path on load and keeps healthy branches folded', () => {
    renderTable();

    expect(
      screen.getByRole('link', { name: 'billing-m1 in billing' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', {
        name: 'orders-shard1-m3 in orders › orders-shard1',
      })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /^orders-mongos in / })
    ).not.toBeInTheDocument();
  });

  it('names each expand button after its row and toggles it', () => {
    renderTable();

    const collapse = screen.getByRole('button', { name: 'Collapse billing' });

    expect(collapse.tagName).toBe('BUTTON');

    fireEvent.click(collapse);

    expect(
      screen.queryByRole('link', { name: 'billing-m1 in billing' })
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Expand billing' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Expand Routers (mongos)' })
    ).toBeInTheDocument();
  });

  it('says whether each branch is expanded, and nothing on a leaf', () => {
    renderTable();

    const billing = screen.getByRole('button', { name: 'Collapse billing' });

    expect(billing).toHaveAttribute('aria-expanded', 'true');
    expect(
      screen.getByRole('button', { name: 'Expand Routers (mongos)' })
    ).toHaveAttribute('aria-expanded', 'false');

    fireEvent.click(billing);

    expect(
      screen.getByRole('button', { name: 'Expand billing' })
    ).toHaveAttribute('aria-expanded', 'false');
    expect(
      closestTo(
        screen.getByRole('link', { name: 'reports' }),
        'tr'
      ).querySelector('button')
    ).not.toHaveAttribute('aria-expanded');
  });

  it('hides the expand button of a row with nothing below it', () => {
    renderTable();

    expect(
      screen.queryByRole('button', { name: /billing-m1$/ })
    ).not.toBeInTheDocument();
  });

  it('opens a name link without toggling its row', () => {
    renderTable();

    fireEvent.click(screen.getByRole('link', { name: 'billing' }));

    expect(screen.getByTestId('hub-location')).toHaveTextContent(
      '/graph/d/summary?var-name=billing'
    );
    expect(
      screen.getByRole('link', { name: 'billing-m1 in billing' })
    ).toBeInTheDocument();
  });

  it('toggles a row from a click on its plain area', () => {
    renderTable();

    fireEvent.click(screen.getByText('Sharded cluster'));

    expect(
      screen.queryByRole('link', {
        name: 'orders-shard1-m3 in orders › orders-shard1',
      })
    ).not.toBeInTheDocument();
  });

  it('shows health words, members with no data, and flagged measures', () => {
    renderTable();

    const reports = closestTo(
      screen.getByRole('link', { name: 'reports' }),
      'tr'
    );

    expect(within(reports).getByText('No data')).toBeInTheDocument();
    expect(reports).toHaveTextContent('0/1 · 1 no data');
    // A link beside plain text is underlined, not told apart by hover alone.
    expect(within(reports).getByRole('link', { name: '0/1' })).toHaveClass(
      'MuiLink-underlineAlways'
    );

    const orders = closestTo(
      screen.getByRole('link', { name: 'orders' }),
      'tr'
    );

    expect(
      within(orders).getByRole('link', {
        name: '40s over the warning threshold for orders-shard1',
      })
    ).toBeInTheDocument();
  });

  it('shows a member’s health only when it is not up', () => {
    renderTable();

    const down = closestTo(
      screen.getByRole('link', { name: 'billing-m1 in billing' }),
      'tr'
    );
    const up = closestTo(
      screen.getByRole('link', { name: 'billing-m3 in billing' }),
      'tr'
    );

    expect(within(down).getByTestId('hub-health')).toHaveTextContent('Down');
    expect(within(up).queryByTestId('hub-health')).not.toBeInTheDocument();
  });

  it('lists the row’s doors in its menu, with the Preview chip on Operations', () => {
    renderTable();

    const billing = closestTo(
      screen.getByRole('link', { name: 'billing' }),
      'tr'
    );

    fireEvent.click(
      within(billing).getByRole('button', { name: 'Row Actions' })
    );

    const menu = screen.getByRole('menu');

    expect(
      within(menu).getByRole('menuitem', { name: 'Open summary' })
    ).toHaveAttribute('href', '/graph/d/replset?var-rs=billing');
    expect(
      within(menu).getByRole('menuitem', {
        name: 'Manage in Operations Developer preview',
      })
    ).toHaveAttribute('href', '/operations?cluster=om-billing');
  });

  it('folds type, environment and members into a caption below md', () => {
    matchAllMedia(true);
    renderTable();

    expect(
      screen.queryByRole('columnheader', { name: 'Type' })
    ).not.toBeInTheDocument();
    expect(screen.getByText('Replica set · prod · 1/3')).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: /Replication lag/ })
    ).toBeInTheDocument();
  });

  describe('by the width it has, not the viewport’s', () => {
    const observeWidth = (width: number) => {
      vi.stubGlobal(
        'ResizeObserver',
        class {
          constructor(
            private readonly callback: (
              entries: Array<{ contentRect: { width: number } }>
            ) => void
          ) {}

          observe() {
            this.callback([{ contentRect: { width } }]);
          }

          disconnect() {}
        }
      );
    };

    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it('folds in a frame narrower than the full tree, on a wide viewport', () => {
      matchAllMedia(false);
      observeWidth(910);
      renderTable();

      expect(
        screen.queryByRole('columnheader', { name: 'Type' })
      ).not.toBeInTheDocument();
      expect(screen.getByText('Replica set · prod · 1/3')).toBeInTheDocument();
    });

    it('shows every column in a frame wide enough, on a narrow viewport', () => {
      matchAllMedia(true);
      observeWidth(1300);
      renderTable();

      expect(
        screen.getByRole('columnheader', { name: 'Type' })
      ).toBeInTheDocument();
    });
  });

  it('describes each measure column in a tooltip its header can focus', async () => {
    renderTable();

    const header = screen.getByText('Replication lag');

    expect(header).toHaveAttribute('tabindex', '0');

    fireEvent.mouseOver(header);

    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      'How far each secondary is behind its primary.'
    );
  });
});
