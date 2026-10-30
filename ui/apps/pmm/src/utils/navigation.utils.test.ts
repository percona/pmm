import { NavItem } from 'types/navigation.types';
import { findActiveNavItem } from './navigation.utils';
import { NAV_INVENTORY } from 'contexts/navigation/navigation.constants';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

describe('findActiveNavItem', () => {
  it("returns root if first child doesn't match", () => {
    const child: NavItem = {
      id: 'child',
      url: '/parent/child',
    };
    const parent: NavItem = {
      id: 'parent',
      url: '/parent',
      children: [child],
    };
    const activeItem = findActiveNavItem(parent, '/parent');

    expect(activeItem?.id).not.toBe(child.id);
    expect(activeItem?.id).toMatch(parent.id);
  });

  it('returns first child if it has the same url as parent', () => {
    const child: NavItem = {
      id: 'child',
      url: '/parent/child',
    };
    const parent: NavItem = {
      id: 'parent',
      url: '/parent/child',
      children: [child],
    };
    const activeItem = findActiveNavItem(parent, '/parent/child');

    expect(activeItem?.id).not.toBe(parent.id);
    expect(activeItem?.id).toBe(child.id);
  });

  it('returns undefined if no match', () => {
    const child: NavItem = {
      id: 'child',
      url: '/parent/child',
    };
    const parent: NavItem = {
      id: 'parent',
      url: '/parent',
      children: [child],
    };
    const activeItem = findActiveNavItem(parent, '/no-match');

    expect(activeItem?.id).not.toBe(parent.id);
    expect(activeItem?.id).not.toBe(child.id);
    expect(activeItem).toBeUndefined();
  });

  it('returns item on subpath match if "*" match is used', () => {
    const child: NavItem = {
      id: 'child',
      url: '/parent/child',
    };
    const parent: NavItem = {
      id: 'parent',
      url: '/parent',
      children: [child],
      matches: ['*'],
    };
    const activeItem = findActiveNavItem(parent, '/parent/test');

    expect(activeItem?.id).toBe(parent.id);
  });

  it('returns item if matches on custom match path', () => {
    const child: NavItem = {
      id: 'child',
      url: '/parent/child',
      matches: ['/custom-path'],
    };
    const parent: NavItem = {
      id: 'parent',
      url: '/parent',
      children: [child],
    };
    const activeItem = findActiveNavItem(parent, '/custom-path');

    expect(activeItem?.id).toBe(child.id);
  });

  it('makes Services the active Inventory item on the Services list', () => {
    const activeItem = findActiveNavItem(
      NAV_INVENTORY,
      `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/services`
    );

    expect(activeItem?.id).toBe('inventory-services');
  });
});
