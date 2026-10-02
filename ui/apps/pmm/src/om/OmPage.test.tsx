import { render, screen } from '@testing-library/react';
import { FC, PropsWithChildren } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { OmPage } from './OmPage';
import { Messages } from './OmPage.messages';

const { useReadonlySettings } = vi.hoisted(() => ({
  useReadonlySettings: vi.fn(),
}));

vi.mock('hooks/api/useSettings', () => ({ useReadonlySettings }));
vi.mock('contexts/user', () => ({
  useUser: () => ({ user: { isPMMAdmin: true } }),
}));
vi.mock('components/page', () => ({
  Page: (({ children }) => <>{children}</>) as FC<PropsWithChildren>,
}));

describe('OmPage', () => {
  beforeEach(() => {
    useReadonlySettings.mockReturnValue({
      data: { omEnabled: true },
      isLoading: false,
    });
  });

  afterEach(() => {
    localStorage.clear();
  });

  it('shows a developer preview banner that cannot be dismissed', () => {
    render(<OmPage>content</OmPage>);

    const banner = screen.getByTestId('om-developer-preview');
    expect(banner).toHaveTextContent(Messages.developerPreviewBody);
    expect(banner).toHaveTextContent('not recommended for production');
    expect(screen.queryByRole('button', { name: /close/i })).toBeNull();
  });

  it('keeps the banner for a browser that dismissed the old one', () => {
    localStorage.setItem('pmm-ui.om.technicalPreviewDismissed', 'true');

    render(<OmPage>content</OmPage>);

    expect(screen.getByTestId('om-developer-preview')).toBeInTheDocument();
  });

  it('shows the switched-off notice instead when Operations for MongoDB is off', () => {
    useReadonlySettings.mockReturnValue({
      data: { omEnabled: false },
      isLoading: false,
    });

    render(<OmPage>content</OmPage>);

    expect(screen.getByTestId('om-switched-off')).toBeInTheDocument();
    expect(screen.queryByTestId('om-developer-preview')).toBeNull();
  });
});
