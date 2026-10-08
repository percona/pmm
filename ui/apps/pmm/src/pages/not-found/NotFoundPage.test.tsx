import { fireEvent, render, screen } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { TestWrapper } from 'utils/testWrapper';
import { PMM_TITLE } from 'lib/constants';
import { NotFoundPage } from './NotFoundPage';
import { Messages } from './NotFoundPage.messages';

const renderAt = (initialEntries: string[]) =>
  render(
    <TestWrapper routerProps={{ initialEntries }}>
      <Routes>
        <Route path="/graph/d/pmm-home" element={<div>Home dashboard</div>} />
        <Route path="/help" element={<div>Help page</div>} />
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </TestWrapper>
  );

describe('NotFoundPage', () => {
  beforeEach(() => {
    window.history.replaceState(null, '');
  });

  it('shows the heading and explanation', () => {
    renderAt(['/feed?tab=1']);

    expect(
      screen.getByRole('heading', { level: 1, name: Messages.title })
    ).toBeInTheDocument();
    expect(screen.getAllByText(Messages.title)).toHaveLength(1);
    expect(screen.getByText(Messages.description)).toBeInTheDocument();
  });

  it('does not show the version footer', () => {
    renderAt(['/feed']);

    expect(screen.queryByTestId('pmm-footer')).toBeNull();
    expect(screen.queryByRole('separator')).toBeNull();
  });

  it('sets the document title', () => {
    renderAt(['/feed']);

    expect(document.title).toBe(`${Messages.title} - ${PMM_TITLE}`);
  });

  it('links to PMM Home', () => {
    renderAt(['/feed']);

    expect(screen.getByTestId('not-found-home-button')).toHaveAttribute(
      'href',
      '/graph/d/pmm-home'
    );
  });

  it('hides "Go back" when the user landed here directly', () => {
    renderAt(['/feed']);

    expect(screen.queryByTestId('not-found-back-button')).toBeNull();
  });

  it('hides "Go back" when a redirect replaced the first entry', () => {
    window.history.replaceState({ idx: 0 }, '');
    renderAt(['/help', '/feed']);

    expect(screen.queryByTestId('not-found-back-button')).toBeNull();
  });

  it('shows "Go back" and returns to the previous page when there is history', () => {
    window.history.replaceState({ idx: 1 }, '');
    renderAt(['/help', '/feed']);

    fireEvent.click(screen.getByTestId('not-found-back-button'));

    expect(screen.getByText('Help page')).toBeInTheDocument();
  });
});
