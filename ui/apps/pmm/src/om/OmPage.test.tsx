import { render, screen } from '@testing-library/react';
import type { FC, PropsWithChildren } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuth } from '@pmm-extensions/api';
import type { OrgRole, User } from 'types/user.types';
import {
  TEST_USER_ADMIN,
  TEST_USER_ANONYMOUS,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { OmPage } from './OmPage';
import { Messages } from './OmPage.messages';

const { useReadonlySettings, useUser } = vi.hoisted(() => ({
  useReadonlySettings: vi.fn(),
  useUser: vi.fn(),
}));

vi.mock('hooks/api/useSettings', () => ({ useReadonlySettings }));
vi.mock('contexts/user', () => ({ useUser }));
vi.mock('components/page', () => ({
  Page: (({ children, roles }) => (
    <div data-testid="page" data-roles={roles?.join(',') ?? 'any'}>
      {children}
    </div>
  )) as FC<PropsWithChildren<{ roles?: OrgRole[] }>>,
}));

const CanMutate = () => <>{String(useAuth().canMutate)}</>;

describe('OmPage', () => {
  beforeEach(() => {
    useUser.mockReturnValue({ user: TEST_USER_ADMIN });
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

  it.each<[string, User, string]>([
    ['an admin may act', TEST_USER_ADMIN, 'true'],
    ['an editor may only look', TEST_USER_EDITOR, 'false'],
    ['a viewer may only look', TEST_USER_VIEWER, 'false'],
  ])('tells the plugin whether %s', (_, user, canMutate) => {
    useUser.mockReturnValue({ user });

    render(
      <OmPage>
        <CanMutate />
      </OmPage>
    );

    expect(screen.getByTestId('page')).toHaveTextContent(canMutate);
  });

  it.each<[string, User, string]>([
    ['admits a signed-in viewer', TEST_USER_VIEWER, 'any'],
    ['turns an anonymous visitor away', TEST_USER_ANONYMOUS, ''],
  ])('%s', (_, user, roles) => {
    useUser.mockReturnValue({ user });

    render(<OmPage>content</OmPage>);

    expect(screen.getByTestId('page')).toHaveAttribute('data-roles', roles);
  });
});
