import { NavItem } from 'types/navigation.types';
import { findActiveNavItem } from './navigation.utils';

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

  it('never returns a section heading', () => {
    const section: NavItem = {
      id: 'section',
      type: 'menu-section',
      text: 'Browse',
      url: '/browse',
    };

    expect(findActiveNavItem([section], '/browse')).toBeUndefined();
  });

  it('gives an item with a query string its page only when the params are present', () => {
    const plain: NavItem = { id: 'plain', url: '/page' };
    const filtered: NavItem = {
      id: 'filtered',
      url: '/page?var-type=postgresql',
    };

    expect(findActiveNavItem([filtered, plain], '/page')?.id).toBe('plain');
    expect(
      findActiveNavItem(
        [filtered, plain],
        '/page',
        '?var-type=postgresql&from=now'
      )?.id
    ).toBe('filtered');
  });

  it('keeps custom match paths path-only for an item with a query string', () => {
    const item: NavItem = {
      id: 'item',
      url: '/page?var-namespace=pmm',
      matches: ['/page'],
    };

    expect(findActiveNavItem([item], '/page')?.id).toBe('item');
  });
});
