import { fireEvent, render, screen } from '@testing-library/react';
import { wrapWithVersion } from 'utils/testUtils';
import { Messages } from './ReloadPrompt.messages';
import ReloadPrompt from './ReloadPrompt';

const renderPrompt = (isOutdated = true, serverVersion = '3.10.0') => {
  const reload = vi.fn();

  render(
    wrapWithVersion(<ReloadPrompt />, { isOutdated, serverVersion, reload })
  );

  return { reload };
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
});
