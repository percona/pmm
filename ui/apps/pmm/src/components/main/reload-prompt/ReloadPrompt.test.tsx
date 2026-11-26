import { fireEvent, render, screen } from '@testing-library/react';
import { wrapWithVersion } from 'utils/testUtils';
import { Messages } from './ReloadPrompt.messages';
import ReloadPrompt from './ReloadPrompt';

const renderPrompt = (
  isOutdated = true,
  serverVersion = '3.10.0',
  serverBuild = 'abc123'
) => {
  const reload = vi.fn();

  const { rerender } = render(
    wrapWithVersion(<ReloadPrompt />, {
      isOutdated,
      serverVersion,
      serverBuild,
      reload,
    })
  );

  /** The server the tab polls turns out to be running a different build again. */
  const serveBuild = (nextBuild: string, nextVersion = serverVersion) =>
    rerender(
      wrapWithVersion(<ReloadPrompt />, {
        isOutdated,
        serverVersion: nextVersion,
        serverBuild: nextBuild,
        reload,
      })
    );

  return { reload, serveBuild };
};

describe('ReloadPrompt', () => {
  it('stays hidden while the page matches the server', () => {
    renderPrompt(false);

    expect(screen.queryByTestId('reload-prompt')).not.toBeInTheDocument();
  });

  it('names the version the server was updated to', () => {
    renderPrompt();

    expect(screen.getByTestId('reload-prompt-title')).toHaveTextContent(
      'PMM Server was updated to 3.10.0'
    );
  });

  it('falls back to a generic title when the version is unknown', () => {
    renderPrompt(true, '');

    expect(screen.getByTestId('reload-prompt-title')).toHaveTextContent(
      Messages.title()
    );
  });

  it('reloads on request', () => {
    const { reload } = renderPrompt();

    fireEvent.click(screen.getByTestId('reload-prompt-reload-button'));

    expect(reload).toHaveBeenCalledTimes(1);
  });

  it('hides when dismissed', () => {
    const { reload } = renderPrompt();

    fireEvent.click(screen.getByTestId('reload-prompt-dismiss-button'));

    expect(screen.queryByTestId('reload-prompt')).not.toBeInTheDocument();
    expect(reload).not.toHaveBeenCalled();
  });

  it('hides when closed', () => {
    renderPrompt();

    fireEvent.click(screen.getByTestId('reload-prompt-close-button'));

    expect(screen.queryByTestId('reload-prompt')).not.toBeInTheDocument();
  });

  it('stays hidden while the server keeps running the dismissed build', () => {
    const { serveBuild } = renderPrompt();
    fireEvent.click(screen.getByTestId('reload-prompt-dismiss-button'));

    serveBuild('abc123');

    expect(screen.queryByTestId('reload-prompt')).not.toBeInTheDocument();
  });

  it('asks again once the server runs a newer build', () => {
    const { serveBuild } = renderPrompt();
    fireEvent.click(screen.getByTestId('reload-prompt-dismiss-button'));

    serveBuild('def456', '3.11.0');

    expect(screen.getByTestId('reload-prompt')).toBeInTheDocument();
    expect(screen.getByTestId('reload-prompt-title')).toHaveTextContent(
      'PMM Server was updated to 3.11.0'
    );
  });

  it('asks again after a rebuild under an unchanged version', () => {
    const { serveBuild } = renderPrompt();
    fireEvent.click(screen.getByTestId('reload-prompt-close-button'));

    // same version tag, different commit: what a release candidate rebuild does
    serveBuild('def456');

    expect(screen.getByTestId('reload-prompt')).toBeInTheDocument();
  });
});
