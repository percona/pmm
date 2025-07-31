export const triggerShortcut = (shortcut: 'view-shortcuts' | 'toggle-theme' | 'search') => {
  if (shortcut === 'search') {
    triggerKeypress({
      key: 'k',
      code: 'KeyK',
      keyCode: 75,
      metaKey: true,
    });
  } else if (shortcut === 'view-shortcuts') {
    triggerKeypress({
      key: '?',
      code: 'Slash',
      keyCode: 191,
      shiftKey: true,
    });
  } else if (shortcut === 'toggle-theme') {
    triggerKeypress({
      key: 'c',
      code: 'KeyC',
      keyCode: 67,
    });
    triggerKeypress({
      key: 't',
      code: 'KeyT',
      keyCode: 84,
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
