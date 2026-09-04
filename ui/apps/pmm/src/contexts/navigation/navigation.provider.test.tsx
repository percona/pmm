import { renderHook } from '@testing-library/react';
import { ReactElement } from 'react';
import { MemoryRouterProps } from 'react-router-dom';
import { NavItem } from 'types/navigation.types';
import { User } from 'types/user.types';
import { SEP_ATW_PATH, SEP_MYSQL_BACKUPS_PATH } from 'lib/constants';
import { findActiveNavItem } from 'utils/navigation.utils';
import {
  TEST_USER_ADMIN,
  TEST_USER_ANONYMOUS,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings, wrapWithUpdatesProvider } from 'utils/testUtils';
import { NavigationProvider } from './navigation.provider';
import { useNavigation } from './navigation.hooks';

vi.mock('hooks/api/useServices', () => ({
  useServiceTypes: () => ({ data: { serviceTypes: [] } }),
}));

vi.mock('hooks/api/useAdvisors', () => ({
  useAdvisors: () => ({ data: [] }),
}));

vi.mock('hooks/api/useFolders', () => ({
  useFolders: () => ({ data: [] }),
}));

vi.mock('hooks/api/useHA', () => ({
  useHaInfo: () => ({ data: { enabled: false, nodes: [] } }),
}));

vi.mock('hooks/theme', () => ({
  useColorMode: () => ({ colorMode: 'light', toggleColorMode: () => {} }),
}));

const renderNavTree = (
  user: User = TEST_USER_ADMIN,
  routerProps?: MemoryRouterProps,
  settings?: { sepEnabled?: boolean; backupManagementEnabled?: boolean }
) => {
  const { result } = renderHook(() => useNavigation(), {
    wrapper: ({ children }) => (
      <TestWrapper
        userContext={{ isLoading: false, user }}
        routerProps={routerProps}
      >
        {wrapWithSettings(
          wrapWithUpdatesProvider(
            <NavigationProvider>{children}</NavigationProvider>
          ) as ReactElement,
          {
            settings: {
              backupManagementEnabled: true,
              sepEnabled: true,
              ...settings,
            },
          }
        )}
      </TestWrapper>
    ),
  });

  return result.current.navTree;
};

const findById = (items: NavItem[], id: string): NavItem | undefined =>
  items.find((item) => item.id === id);

describe('NavigationProvider', () => {
  describe('Management section', () => {
    it.each([
      ['admin', TEST_USER_ADMIN],
      ['editor', TEST_USER_EDITOR],
      ['viewer', TEST_USER_VIEWER],
    ])('is a single collapsible group for %s', (_role, user) => {
      const navTree = renderNavTree(user);
      const management = findById(navTree, 'management');

      expect(management).toBeDefined();
      expect(management?.text).toBe('Management');
      expect(management?.url).toBeUndefined();
      expect(management?.children?.map((child) => child.id)).toEqual([
        'sep-mysql-backups',
        'sep-atw',
      ]);
    });

    it('does not leave the SEP apps as top-level entries', () => {
      const ids = renderNavTree().map((item) => item.id);

      expect(ids).not.toContain('sep-atw');
      expect(ids).not.toContain('sep-mysql-backups');
    });

    it('preserves each child url, matches and icon', () => {
      const management = findById(renderNavTree(), 'management');
      const [mysqlBackups, atw] = management?.children || [];

      expect(mysqlBackups).toMatchObject({
        text: 'MySQL Backups',
        url: SEP_MYSQL_BACKUPS_PATH,
        matches: [SEP_MYSQL_BACKUPS_PATH],
      });
      expect(mysqlBackups?.icon).toBeDefined();

      expect(atw).toMatchObject({
        text: 'Support diagnostics',
        url: SEP_ATW_PATH,
        matches: [SEP_ATW_PATH],
      });
      expect(atw?.icon).toBeDefined();
    });

    it('sits right below Inventory for an admin, moving nothing else', () => {
      const ids = renderNavTree().map((item) => item.id);
      // Only the block the section joins is pinned: asserting the whole tree
      // would break on any unrelated nav addition without telling us anything
      // about where Management landed.
      const block = ids.slice(ids.indexOf('inventory-divider'));

      expect(block).toEqual([
        'inventory-divider',
        'inventory',
        'management',
        'backups',
        'backups-divider',
        'configuration',
        'users-and-access',
        'account',
        'help',
      ]);
    });

    it('is withheld from anonymous, which has no SEP session to exchange', () => {
      const ids = renderNavTree(TEST_USER_ANONYMOUS).map((item) => item.id);

      expect(ids).not.toContain('management');
    });

    it('is withheld when SEP is disabled in server settings', () => {
      const ids = renderNavTree(TEST_USER_ADMIN, undefined, {
        sepEnabled: false,
      }).map((item) => item.id);

      expect(ids).not.toContain('management');
    });

    it('opens the block for a viewer, who gets no Inventory of their own', () => {
      const ids = renderNavTree(TEST_USER_VIEWER).map((item) => item.id);

      expect(ids).not.toContain('inventory');
      expect(ids).not.toContain('backups');
      expect(ids).not.toContain('configuration');
      expect(ids.indexOf('management')).toBe(
        ids.indexOf('inventory-divider') + 1
      );
    });
  });

  describe('deep links into a SEP app', () => {
    it.each([
      ['sep-atw', SEP_ATW_PATH],
      ['sep-mysql-backups', SEP_MYSQL_BACKUPS_PATH],
    ])('marks %s active and keeps it inside Management', (childId, path) => {
      const navTree = renderNavTree(TEST_USER_VIEWER, {
        initialEntries: [path],
      });
      const management = findById(navTree, 'management');
      const active = findActiveNavItem(navTree, path);

      expect(active?.id).toBe(childId);
      // The sidebar expands a section when its active child is the very object
      // held in `children`, so identity — not just the id — has to match.
      expect(management?.children).toContain(active);
    });
  });
});
