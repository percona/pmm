import { render, screen, waitFor } from '@testing-library/react';
import { Icon } from './index';

vi.mock('./Icon.constants', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./Icon.constants')>();

  return {
    ...actual,
    DYNAMIC_ICON_IMPORT_MAP: {
      ...actual.DYNAMIC_ICON_IMPORT_MAP,
      'pmm-rounded': () => Promise.reject(new Error('chunk is gone')),
    },
  };
});

describe('Icon', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('leaves the rest of the page standing when its chunk is gone', async () => {
    // React reports the error it caught through console.error, which is the only
    // signal that the import has actually failed by now rather than still pending
    const reported = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <div>
        <span data-testid="sibling">still here</span>
        <Icon name="pmm-rounded" />
      </div>
    );

    await waitFor(() => expect(reported).toHaveBeenCalled());

    // with nothing to catch it React unwinds to the root and the sibling goes too
    expect(screen.getByTestId('sibling')).toBeInTheDocument();
  });
});
