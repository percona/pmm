import { screen, within } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { testDoor } from '../hub.testFixtures';
import { HubSection } from './HubSection';

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

const renderSection = () =>
  renderHub(
    <HubSection
      id="clusters"
      title="Clusters"
      count={3}
      doors={[
        testDoor('Fleet in Operations', '/operations?tab=clusters', {
          preview: 'developer',
        }),
      ]}
    >
      body
    </HubSection>
  );

describe('HubSection', () => {
  beforeEach(() => {
    if (!window.matchMedia) {
      window.matchMedia = (() => ({})) as unknown as typeof window.matchMedia;
    }
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('opens its door as a text button with the Preview chip', () => {
    matchAllMedia(false);
    renderSection();

    const door = screen.getByRole('link', {
      name: 'Fleet in Operations Developer preview',
    });

    expect(door).toHaveTextContent('Fleet in Operations');
    expect(within(door).getByTestId('preview-chip')).toBeInTheDocument();
  });

  it('keeps the Preview chip visible beside the narrow icon door', () => {
    matchAllMedia(true);
    renderSection();

    const door = screen.getByRole('link', {
      name: 'Fleet in Operations Developer preview',
    });
    // The header row: the title with its count, then the door.
    const header = screen.getByRole('heading', { name: 'Clusters' })
      .parentElement?.parentElement as HTMLElement;

    expect(door).toHaveTextContent('');
    expect(within(header).getByTestId('preview-chip')).toBeInTheDocument();
    expect(
      within(header).getAllByRole('link', { name: /Fleet in Operations/ })
    ).toHaveLength(1);
  });

  it('keeps several doors as named buttons on a phone, where arrows could not tell them apart', () => {
    matchAllMedia(true);
    renderHub(
      <HubSection
        id="dashboards"
        title="Dashboards"
        doors={[
          testDoor('All MongoDB dashboards 41', '/graph/dashboards'),
          testDoor('All apps', '/prototype/all-apps'),
        ]}
      >
        body
      </HubSection>
    );

    expect(
      screen.getByRole('link', { name: 'All MongoDB dashboards 41' })
    ).toHaveTextContent('All MongoDB dashboards 41');
    expect(screen.getByRole('link', { name: 'All apps' })).toHaveAttribute(
      'href',
      '/prototype/all-apps'
    );
  });
});
