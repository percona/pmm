import { render, screen } from '@testing-library/react';
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
    mocks.useHeader.mockClear();
  });

  it('should render', () => {
    mocks.useHeader.mockReturnValueOnce({
      visible: true,
      Component: () => <div>Header</div>,
    });

    render(<Header />);

    expect(screen.getByText('Header')).toBeInTheDocument();
  });

  it('should not render if not visible', () => {
    mocks.useHeader.mockReturnValueOnce({
      visible: false,
      Component: null,
    });

    render(<Header />);

    expect(screen.queryByText('Header')).not.toBeInTheDocument();
  });

  it('should not render if no component', () => {
    mocks.useHeader.mockReturnValueOnce({
      visible: true,
      Component: null,
    });

    render(<Header />);

    expect(screen.queryByText('Header')).not.toBeInTheDocument();
  });
});
