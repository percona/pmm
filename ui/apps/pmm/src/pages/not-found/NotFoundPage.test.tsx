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
  it('shows the heading, explanation and requested address', () => {
    renderAt(['/feed?tab=1']);

    expect(screen.getByText(Messages.title)).toBeInTheDocument();
    expect(screen.getByText(Messages.heading)).toBeInTheDocument();
    expect(screen.getByText(Messages.description)).toBeInTheDocument();
    expect(screen.getByTestId('not-found-requested-path')).toHaveTextContent(
      '/pmm-ui/feed?tab=1'
    );
  });

  it('sets the document title', () => {
    renderAt(['/feed']);

    expect(document.title).toBe(`${Messages.title} - ${PMM_TITLE}`);
  });

  it('links to PMM Home and the quick links', () => {
    renderAt(['/feed']);

    expect(screen.getByTestId('not-found-home-button')).toHaveAttribute(
      'href',
      '/graph/d/pmm-home'
    );
    expect(screen.getByTestId('not-found-link-inventory')).toHaveAttribute(
      'href',
      '/graph/inventory'
    );
    expect(screen.getByTestId('not-found-link-settings')).toHaveAttribute(
      'href',
      '/settings'
    );
    expect(screen.getByTestId('not-found-link-help')).toHaveAttribute(
      'href',
      '/help'
    );
  });

  it('hides "Go back" when the user landed here directly', () => {
    renderAt(['/feed']);

    expect(screen.queryByTestId('not-found-back-button')).toBeNull();
  });

  it('shows "Go back" and returns to the previous page when there is history', () => {
    renderAt(['/help', '/feed']);

    fireEvent.click(screen.getByTestId('not-found-back-button'));

    expect(screen.getByText('Help page')).toBeInTheDocument();
  });
});
