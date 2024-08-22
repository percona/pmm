import { render } from '@testing-library/react';
import { Changelog } from './Changelog';

const mocks = vi.hoisted(() => ({
  useChangelogs: vi.fn(),
}));

vi.mock('hooks/api/useUpdates', () => ({
  useChangelogs: mocks.useChangelogs,
}));

describe('Changelog', () => {
  beforeEach(() => {
    mocks.useChangelogs.mockClear();
  });

  it('is not visible when loading', () => {
    mocks.useChangelogs.mockReturnValueOnce({
      isLoading: true,
    });
    const { container } = render(<Changelog />);

    expect(container).toBeEmptyDOMElement();
  });

  it('is not visible if there are no release notes available', () => {
    mocks.useChangelogs.mockReturnValueOnce({
      data: { updates: [] },
    });
    const { container } = render(<Changelog />);

    expect(container).toBeEmptyDOMElement();
  });

  it("doesn't render divider after last changelog", () => {
    mocks.useChangelogs.mockReturnValueOnce({
      data: { updates: [{ version: '0.0.1', releaseNotesText: '# 0.0.1' }] },
    });
    const { container } = render(<Changelog />);

    expect(container.getElementsByTagName('hr')).toHaveLength(1);
  });

  it('renders divider between changelogs', () => {
    mocks.useChangelogs.mockReturnValueOnce({
      data: {
        updates: [
          { version: '0.0.1', releaseNotesText: '# 0.0.1' },
          { version: '0.0.2', releaseNotesText: '# 0.0.2' },
        ],
      },
    });
    const { container } = render(<Changelog />);

    expect(container.getElementsByTagName('hr')).toHaveLength(2);
  });
});
