import { describe, expect, it, vi } from 'vitest';
import { getLinkProps } from './SidebarNavItem.utils';

describe('getLinkProps', () => {
  it('keeps a target link an anchor and runs its onClick', () => {
    // "Sign in" needs both: a top-level document load, plus the handler that records the return.
    const onClick = vi.fn();

    const props = getLinkProps(
      {
        id: 'sign-in',
        text: 'Sign in',
        url: '/graph/login',
        target: '_self',
        onClick,
      },
      '/graph/login'
    );

    expect(props).toEqual({
      component: 'a',
      target: '_self',
      href: '/graph/login',
      onClick,
    });
  });

  it('leaves a target link without a handler alone', () => {
    const props = getLinkProps(
      { id: 'ext', text: 'External', url: '/graph/logout', target: '_self' },
      '/graph/logout'
    );

    expect(props).toEqual({
      component: 'a',
      target: '_self',
      href: '/graph/logout',
    });
  });

  it('still returns a bare handler for an item with no url', () => {
    const onClick = vi.fn();

    expect(getLinkProps({ id: 'theme', text: 'Theme', onClick })).toEqual({
      onClick,
    });
  });
});
