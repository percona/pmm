import { NavItem } from 'types/navigation.types';
import { findActiveNavItem } from './navigation.utils';
import { NAV_INVENTORY } from 'contexts/navigation/navigation.constants';
import { PMM_INVENTORY_SERVICES_PATH } from 'lib/constants';

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
      PMM_INVENTORY_SERVICES_PATH
    );

    expect(activeItem?.id).toBe('inventory-services');
  });

  it('never returns a section heading', () => {
    const section: NavItem = {
      id: 'section',
      type: 'menu-section',
      text: 'Browse',
      url: '/browse',
    };

    expect(findActiveNavItem([section], '/browse')).toBeUndefined();
  });

  it('matches an item whose link carries a query string by its path', () => {
    const item: NavItem = { id: 'item', url: '/page?var-namespace=pmm' };

    expect(findActiveNavItem([item], '/page')?.id).toBe('item');
  });
});
