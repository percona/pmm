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
  it('leaves the rest of the page standing when its chunk is gone', async () => {
    render(
      <div>
        <span data-testid="sibling">still here</span>
        <Icon name="pmm-rounded" />
      </div>
    );

    // without a boundary React unwinds to the root and the whole tree goes with it
    await waitFor(() =>
      expect(screen.getByTestId('sibling')).toBeInTheDocument()
    );
    expect(screen.getByTestId('sibling')).toBeInTheDocument();
  });
});
