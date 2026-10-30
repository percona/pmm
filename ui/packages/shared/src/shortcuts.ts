export interface NavigatorLike {
  platform?: string;
  userAgent?: string;
  userAgentData?: { platform?: string };
}

export interface ShortcutEventLike {
  // Chrome autofill fires keydown events without a key.
  key?: string;
  code?: string;
  metaKey: boolean;
  ctrlKey: boolean;
  altKey: boolean;
  shiftKey: boolean;
  isComposing?: boolean;
  keyCode?: number;
}

/** A platform-resolved shortcut, structurally identical to Peak UI's `KeyboardShortcut`. */
export interface SearchShortcut {
  keys: readonly string[];
  text: string;
  spoken: string;
  ariaKeyShortcuts: string;
}

const APPLE_PATTERN = /Mac|iPhone|iPad|iPod/;

const getNavigator = (): NavigatorLike | undefined =>
  typeof navigator === 'undefined' ? undefined : (navigator as NavigatorLike);

/** Reads `navigator.platform` first, the same source tinykeys uses for `$mod`, so both palettes agree. */
export const isApplePlatform = (
  nav: NavigatorLike | undefined = getNavigator()
): boolean => {
  if (!nav) {
    return false;
  }

  if (nav.platform) {
    return APPLE_PATTERN.test(nav.platform);
  }

  const uaPlatform = nav.userAgentData?.platform;

  if (uaPlatform) {
    return uaPlatform === 'macOS' || uaPlatform === 'iOS';
  }

  return APPLE_PATTERN.test(nav.userAgent ?? '');
};

/** ⌘K on Apple platforms and Ctrl+K elsewhere, on any keyboard layout. */
export const isSearchShortcut = (
  event: ShortcutEventLike,
  apple: boolean = isApplePlatform()
): boolean => {
  if (
    event.isComposing ||
    event.keyCode === 229 ||
    event.altKey ||
    event.shiftKey
  ) {
    return false;
  }

  const modifier = apple
    ? event.metaKey && !event.ctrlKey
    : event.ctrlKey && !event.metaKey;

  if (!modifier) {
    return false;
  }

  const key = event.key ?? '';

  // A non-Latin layout reports another letter in `key`, so fall back to the physical key.
  return (
    key.toLowerCase() === 'k' ||
    (event.code === 'KeyK' && !/^[a-z]$/i.test(key))
  );
};

export const getSearchShortcut = (
  apple: boolean = isApplePlatform()
): SearchShortcut =>
  apple
    ? {
        keys: ['⌘', 'K'],
        text: '⌘K',
        spoken: 'Command K',
        ariaKeyShortcuts: 'Meta+K',
      }
    : {
        keys: ['Ctrl', 'K'],
        text: 'Ctrl+K',
        spoken: 'Control K',
        ariaKeyShortcuts: 'Control+K',
      };

export const getModifierKeyCap = (
  apple: boolean = isApplePlatform()
): '⌘' | 'Ctrl' => (apple ? '⌘' : 'Ctrl');
