import {
  getModifierKeyCap,
  getSearchShortcut,
  isApplePlatform,
  isSearchShortcut,
  type ShortcutEventLike,
} from './shortcuts';

const keyEvent = (
  overrides: Partial<ShortcutEventLike>
): ShortcutEventLike => ({
  key: 'k',
  code: 'KeyK',
  metaKey: false,
  ctrlKey: false,
  altKey: false,
  shiftKey: false,
  ...overrides,
});

describe('isApplePlatform', () => {
  it.each(['MacIntel', 'iPhone', 'iPad'])(
    'is true for platform %s',
    (platform) => {
      expect(isApplePlatform({ platform })).toBe(true);
    }
  );

  it.each(['Win32', 'Linux x86_64'])('is false for platform %s', (platform) => {
    expect(isApplePlatform({ platform })).toBe(false);
  });

  it('falls back to UA-CH when the platform is empty', () => {
    expect(
      isApplePlatform({ platform: '', userAgentData: { platform: 'macOS' } })
    ).toBe(true);
    expect(
      isApplePlatform({ platform: '', userAgentData: { platform: 'Windows' } })
    ).toBe(false);
  });

  it('falls back to the user agent when the platform and UA-CH are empty', () => {
    expect(
      isApplePlatform({
        platform: '',
        userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
      })
    ).toBe(true);
    expect(
      isApplePlatform({
        platform: '',
        userAgent: 'Mozilla/5.0 (X11; Linux x86_64)',
      })
    ).toBe(false);
  });

  it('prefers the platform over UA-CH', () => {
    expect(
      isApplePlatform({
        platform: 'Win32',
        userAgentData: { platform: 'macOS' },
      })
    ).toBe(false);
  });

  it('is false without a navigator', () => {
    expect(isApplePlatform(undefined)).toBe(isApplePlatform());
    expect(isApplePlatform({})).toBe(false);
  });
});

describe('isSearchShortcut', () => {
  it('accepts ⌘K on Apple platforms', () => {
    expect(isSearchShortcut(keyEvent({ metaKey: true }), true)).toBe(true);
  });

  it('accepts Ctrl+K elsewhere', () => {
    expect(isSearchShortcut(keyEvent({ ctrlKey: true }), false)).toBe(true);
  });

  it('accepts an uppercase K from caps lock', () => {
    expect(isSearchShortcut(keyEvent({ key: 'K', ctrlKey: true }), false)).toBe(
      true
    );
  });

  it('accepts the physical K key on a non-Latin layout', () => {
    expect(isSearchShortcut(keyEvent({ key: 'л', ctrlKey: true }), false)).toBe(
      true
    );
  });

  it('accepts the K letter on a remapped Latin layout', () => {
    expect(
      isSearchShortcut(
        keyEvent({ key: 'k', code: 'KeyV', metaKey: true }),
        true
      )
    ).toBe(true);
  });

  it.each([
    ['Ctrl+K on Apple platforms', keyEvent({ ctrlKey: true }), true],
    ['⌘K elsewhere', keyEvent({ metaKey: true }), false],
    ['with Shift', keyEvent({ metaKey: true, shiftKey: true }), true],
    ['with Alt', keyEvent({ ctrlKey: true, altKey: true }), false],
    ['⌘+Ctrl+K', keyEvent({ metaKey: true, ctrlKey: true }), true],
    [
      'another Latin letter on the K key',
      keyEvent({ key: 't', ctrlKey: true }),
      false,
    ],
    ['while composing', keyEvent({ ctrlKey: true, isComposing: true }), false],
    ['an IME key event', keyEvent({ ctrlKey: true, keyCode: 229 }), false],
    ['without a modifier', keyEvent({}), false],
  ])('rejects %s', (_, event, apple) => {
    expect(isSearchShortcut(event, apple)).toBe(false);
  });

  it('returns false for an event without a key', () => {
    expect(
      isSearchShortcut(
        keyEvent({ key: undefined, code: undefined, ctrlKey: true }),
        false
      )
    ).toBe(false);
  });
});

describe('getSearchShortcut', () => {
  it('describes ⌘K on Apple platforms', () => {
    expect(getSearchShortcut(true)).toEqual({
      keys: ['⌘', 'K'],
      text: '⌘K',
      spoken: 'Command K',
      ariaKeyShortcuts: 'Meta+K',
    });
  });

  it('describes Ctrl+K elsewhere', () => {
    expect(getSearchShortcut(false)).toEqual({
      keys: ['Ctrl', 'K'],
      text: 'Ctrl+K',
      spoken: 'Control K',
      ariaKeyShortcuts: 'Control+K',
    });
  });
});

describe('getModifierKeyCap', () => {
  it('returns the platform modifier', () => {
    expect(getModifierKeyCap(true)).toBe('⌘');
    expect(getModifierKeyCap(false)).toBe('Ctrl');
  });
});
