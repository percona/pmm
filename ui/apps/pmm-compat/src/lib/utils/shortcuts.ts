import { isUserAgentApple } from './navigator';

export const triggerShortcut = (shortcut: 'view-shortcuts' | 'toggle-theme' | 'search') => {
  const isApple = isUserAgentApple();

  if (shortcut === 'search') {
    triggerKeypress({
      key: 'k',
      code: 'KeyK',
      keyCode: 75,
      which: 75,
      metaKey: isApple,
      ctrlKey: !isApple,
    });
  } else if (shortcut === 'view-shortcuts') {
    triggerKeypress({
      key: '?',
      code: 'Slash',
      keyCode: 191,
      which: 191,
      shiftKey: true,
    });
  } else if (shortcut === 'toggle-theme') {
    triggerKeypress({
      key: 'c',
      code: 'KeyC',
      keyCode: 67,
      which: 67,
    });
    triggerKeypress({
      key: 't',
      code: 'KeyT',
      keyCode: 84,
      which: 84,
    });
  }
};

const triggerKeypress = (init: KeyboardEventInit) => {
  const event = new KeyboardEvent('keydown', {
    bubbles: true,
    cancelable: true,
    ...init,
  });

  document.dispatchEvent(event);
};
