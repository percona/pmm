import { render } from '@testing-library/react';
import { ChangeLog } from './ChangeLog';

const mocks = vi.hoisted(() => ({
  useChangeLogs: vi.fn(),
}));

vi.mock('hooks/api/useUpdates', () => ({
  useChangeLogs: mocks.useChangeLogs,
}));

describe('ChangeLog', () => {
  beforeEach(() => {
    mocks.useChangeLogs.mockClear();
  });

  it('is not visible when loading', () => {
    mocks.useChangeLogs.mockReturnValueOnce({
      isLoading: true,
    });
    const { container } = render(<ChangeLog />);

    expect(container).toBeEmptyDOMElement();
  });

  it('is not visible if there are no release notes available', () => {
    mocks.useChangeLogs.mockReturnValueOnce({
      data: { updates: [] },
    });
    const { container } = render(<ChangeLog />);

    expect(container).toBeEmptyDOMElement();
  });

  it("doesn't render divider after last change log", () => {
    mocks.useChangeLogs.mockReturnValueOnce({
      data: { updates: [{ version: '0.0.1', releaseNotesText: '# 0.0.1' }] },
    });
    const { container } = render(<ChangeLog />);

    expect(container.getElementsByTagName('hr')).toHaveLength(1);
  });

  it('renders divider between change logs', () => {
    mocks.useChangeLogs.mockReturnValueOnce({
      data: {
        updates: [
          { version: '0.0.1', releaseNotesText: '# 0.0.1' },
          { version: '0.0.2', releaseNotesText: '# 0.0.2' },
        ],
      },
    });
    const { container } = render(<ChangeLog />);

    expect(container.getElementsByTagName('hr')).toHaveLength(2);
  });
});
