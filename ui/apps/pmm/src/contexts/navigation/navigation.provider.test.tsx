import { renderHook } from '@testing-library/react';
import { ReactElement } from 'react';
import { MemoryRouterProps } from 'react-router-dom';
import { NavItem } from 'types/navigation.types';
import { User } from 'types/user.types';
import {
  EXTENSIONS_ATW_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  OM_PATH,
} from 'lib/constants';
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
  settings?: {
    extensionsEnabled?: boolean;
    backupManagementEnabled?: boolean;
    omEnabled?: boolean;
  }
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
              extensionsEnabled: true,
              omEnabled: true,
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
        'extensions-mysql-backups',
        'extensions-atw',
      ]);
    });

    it('does not leave the PMM Extensions apps as top-level entries', () => {
      const ids = renderNavTree().map((item) => item.id);

      expect(ids).not.toContain('extensions-atw');
      expect(ids).not.toContain('extensions-mysql-backups');
    });

    it('preserves each child url, matches and icon', () => {
      const management = findById(renderNavTree(), 'management');
      const [mysqlBackups, atw] = management?.children || [];

      expect(mysqlBackups).toMatchObject({
        text: 'MySQL Backups',
        url: EXTENSIONS_MYSQL_BACKUPS_PATH,
        matches: ['*'],
      });
      expect(mysqlBackups?.icon).toBeDefined();

      expect(atw).toMatchObject({
        text: 'Support diagnostics',
        url: EXTENSIONS_ATW_PATH,
        matches: ['*'],
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

    it('is withheld from anonymous, which has no side-car session to exchange', () => {
      const ids = renderNavTree(TEST_USER_ANONYMOUS).map((item) => item.id);

      expect(ids).not.toContain('management');
    });

    it('is withheld when PMM Extensions is disabled in server settings', () => {
      const ids = renderNavTree(TEST_USER_ADMIN, undefined, {
        extensionsEnabled: false,
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

  describe('Operations (OpenManager)', () => {
    const mongoChildIds = (navTree: NavItem[]) =>
      findById(navTree, 'mongo')?.children?.map((child) => child.id);

    it('is nested in the MongoDB menu, not a top-level entry', () => {
      const navTree = renderNavTree();

      expect(navTree.map((item) => item.id)).not.toContain('om');
      // No MongoDB service is monitored here, so the menu exists for OM alone.
      expect(mongoChildIds(navTree)).toEqual(['om']);
    });

    it('is withheld when the switch is off', () => {
      // Gated on its own settings flag rather than on the PMM Extensions group:
      // it is served by pmm-managed (PMM-15360).
      const navTree = renderNavTree(TEST_USER_ADMIN, undefined, {
        omEnabled: false,
      });

      expect(findById(navTree, 'mongo')).toBeUndefined();
    });

    it('is withheld from a non-admin', () => {
      expect(
        findById(renderNavTree(TEST_USER_EDITOR), 'mongo')
      ).toBeUndefined();
    });

    it.each([
      ['om-fleet', OM_PATH],
      ['om-nodes', `${OM_PATH}/nodes`],
      ['om-automations', `${OM_PATH}/automations`],
      ['om-settings', `${OM_PATH}/settings`],
    ])('marks %s active inside the MongoDB menu', (childId, path) => {
      const navTree = renderNavTree(TEST_USER_ADMIN, {
        initialEntries: [path],
      });
      const om = findById(findById(navTree, 'mongo')?.children ?? [], 'om');
      const active = findActiveNavItem(navTree, path);

      expect(active?.id).toBe(childId);
      expect(om?.children).toContain(active);
    });
  });

  describe('deep links into a PMM Extensions app', () => {
    it.each([
      ['extensions-atw', EXTENSIONS_ATW_PATH],
      ['extensions-atw', `${EXTENSIONS_ATW_PATH}/runs/abc`],
      ['extensions-mysql-backups', EXTENSIONS_MYSQL_BACKUPS_PATH],
      [
        'extensions-mysql-backups',
        `${EXTENSIONS_MYSQL_BACKUPS_PATH}/backups/123`,
      ],
    ])(
      'marks %s active and keeps it inside Management for %s',
      (childId, path) => {
        const navTree = renderNavTree(TEST_USER_VIEWER, {
          initialEntries: [path],
        });
        const management = findById(navTree, 'management');
        const active = findActiveNavItem(navTree, path);

        expect(active?.id).toBe(childId);
        // The sidebar expands a section when its active child is the very object
        // held in `children`, so identity — not just the id — has to match.
        expect(management?.children).toContain(active);
      }
    );
  });
});
