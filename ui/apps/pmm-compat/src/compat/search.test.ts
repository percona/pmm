jest.mock('lib/utils/shortcuts', () => ({ triggerShortcut: jest.fn() }));

import {
  CrossFrameMessenger,
  type MessageListener,
  type SearchConfigPayload,
} from '@pmm/shared';
import { triggerShortcut } from 'lib/utils/shortcuts';
import {
  initSearch,
  openSearch,
  resetSearchForTests,
  type SearchMessenger,
} from './search';

type ConfigListener = MessageListener<'SEARCH_CONFIG', SearchConfigPayload>;

const connect = () => {
  const listeners: ConfigListener[] = [];
  const sendMessage = jest.fn();
  const messenger: SearchMessenger = {
    addListener: (listener) => {
      listeners.push(listener as unknown as ConfigListener);
      return listener;
    },
    sendMessage,
  };

  initSearch(messenger);

  const configure = (payload?: SearchConfigPayload) =>
    listeners
      .filter((listener) => listener.type === 'SEARCH_CONFIG')
      .forEach((listener) =>
        listener.onMessage({ type: 'SEARCH_CONFIG', source: 'PMM', payload })
      );

  return { listeners, sendMessage, configure };
};

const pressShortcut = (
  target: EventTarget = document.body,
  init: KeyboardEventInit = {}
) => {
  const event = new KeyboardEvent('keydown', {
    key: 'k',
    code: 'KeyK',
    ctrlKey: true,
    bubbles: true,
    cancelable: true,
    ...init,
  });

  target.dispatchEvent(event);

  return event;
};

const append = <K extends keyof HTMLElementTagNameMap>(tag: K) => {
  const element = document.createElement(tag);
  document.body.append(element);
  return element;
};

describe('compat search', () => {
  const windowKeyDown = jest.fn();
  let frames: FrameRequestCallback[] = [];

  const runFrame = () => {
    const callbacks = frames;
    frames = [];
    callbacks.forEach((callback) => callback(performance.now()));
  };

  // Focus moves from the frame to PMM UI's search input and comes back on close.
  const leaveAndReturnToFrame = () => {
    (document.activeElement as HTMLElement | null)?.blur();
    window.dispatchEvent(new Event('focus'));
  };

  beforeEach(() => {
    jest.clearAllMocks();
    frames = [];
    jest.spyOn(window.navigator, 'platform', 'get').mockReturnValue('Win32');
    jest
      .spyOn(window, 'requestAnimationFrame')
      .mockImplementation((callback) => frames.push(callback));
    window.addEventListener('keydown', windowKeyDown);
  });

  afterEach(() => {
    resetSearchForTests();
    window.removeEventListener('keydown', windowKeyDown);
    jest.restoreAllMocks();
    document.body.innerHTML = '';
    window.history.replaceState(null, '', '/');
  });

  describe('off by default', () => {
    it('registers only a SEARCH_CONFIG listener and no DOM listener', () => {
      const documentAdd = jest.spyOn(document, 'addEventListener');
      const windowAdd = jest.spyOn(window, 'addEventListener');

      const { listeners } = connect();

      expect(listeners.map((listener) => listener.type)).toEqual([
        'SEARCH_CONFIG',
      ]);
      expect(documentAdd).not.toHaveBeenCalled();
      expect(windowAdd).not.toHaveBeenCalled();
    });

    it('leaves the shortcut to Grafana', () => {
      const { sendMessage } = connect();

      const event = pressShortcut();

      expect(event.defaultPrevented).toBe(false);
      expect(windowKeyDown).toHaveBeenCalledTimes(1);
      expect(sendMessage).not.toHaveBeenCalled();
    });

    it('stays off when the config does not enable search', () => {
      const documentAdd = jest.spyOn(document, 'addEventListener');
      const { configure, sendMessage } = connect();

      configure({ enabled: false });
      configure();
      pressShortcut();

      expect(documentAdd).not.toHaveBeenCalledWith(
        'keydown',
        expect.any(Function)
      );
      expect(windowKeyDown).toHaveBeenCalledTimes(1);
      expect(sendMessage).not.toHaveBeenCalled();
    });
  });

  describe('after SEARCH_CONFIG enables search', () => {
    it('takes over the shortcut and asks PMM UI to open search', () => {
      const { configure, sendMessage } = connect();
      configure({ enabled: true });

      const event = pressShortcut();

      expect(event.defaultPrevented).toBe(true);
      expect(windowKeyDown).not.toHaveBeenCalled();
      expect(sendMessage).toHaveBeenCalledTimes(1);
      expect(sendMessage).toHaveBeenCalledWith({ type: 'OPEN_SEARCH' });
    });

    it('works through the real messenger', () => {
      const topFrame = { postMessage: jest.fn() };
      const messenger = new CrossFrameMessenger('GRAFANA')
        .setTargetWindow(topFrame as unknown as Window)
        .register();
      initSearch(messenger);

      window.dispatchEvent(
        new MessageEvent('message', {
          data: { type: 'SEARCH_CONFIG', payload: { enabled: true } },
        })
      );
      pressShortcut();
      messenger.unregister();

      expect(topFrame.postMessage).toHaveBeenCalledTimes(1);
      expect(topFrame.postMessage).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'OPEN_SEARCH', source: 'GRAFANA' }),
        undefined
      );
    });

    it('uses Command K on Apple platforms', () => {
      jest
        .spyOn(window.navigator, 'platform', 'get')
        .mockReturnValue('MacIntel');
      const { configure, sendMessage } = connect();
      configure({ enabled: true });

      const ctrlEvent = pressShortcut();
      const metaEvent = pressShortcut(document.body, {
        ctrlKey: false,
        metaKey: true,
      });

      expect(ctrlEvent.defaultPrevented).toBe(false);
      expect(metaEvent.defaultPrevented).toBe(true);
      expect(sendMessage).toHaveBeenCalledTimes(1);
    });

    it('adds the keydown listener once however often it is enabled', () => {
      const documentAdd = jest.spyOn(document, 'addEventListener');
      const { configure, sendMessage } = connect();

      configure({ enabled: true });
      configure({ enabled: true });
      pressShortcut();

      expect(
        documentAdd.mock.calls.filter(([type]) => type === 'keydown')
      ).toHaveLength(1);
      expect(sendMessage).toHaveBeenCalledTimes(1);
    });
  });

  describe('pass-through cases', () => {
    it('swallows a held shortcut without opening search again', () => {
      const { configure, sendMessage } = connect();
      configure({ enabled: true });

      const event = pressShortcut(document.body, { repeat: true });

      expect(event.defaultPrevented).toBe(true);
      expect(windowKeyDown).not.toHaveBeenCalled();
      expect(sendMessage).not.toHaveBeenCalled();
    });

    it('leaves an event that a target listener already prevented', () => {
      const { configure, sendMessage } = connect();
      configure({ enabled: true });
      const input = append('input');
      input.addEventListener('keydown', (event) => event.preventDefault());

      pressShortcut(input);

      expect(windowKeyDown).toHaveBeenCalledTimes(1);
      expect(sendMessage).not.toHaveBeenCalled();
    });

    it('sends nothing when an editor stops the event at its target', () => {
      const { configure, sendMessage } = connect();
      configure({ enabled: true });
      const editor = append('textarea');
      editor.addEventListener('keydown', (event) => event.stopPropagation());

      const event = pressShortcut(editor);

      expect(event.defaultPrevented).toBe(false);
      expect(sendMessage).not.toHaveBeenCalled();
    });

    it("leaves the shortcut to Grafana's palette while it is open", () => {
      const { configure, sendMessage } = connect();
      configure({ enabled: true });
      append('div').id = 'kbar-listbox';

      const event = pressShortcut();

      expect(event.defaultPrevented).toBe(false);
      expect(windowKeyDown).toHaveBeenCalledTimes(1);
      expect(sendMessage).not.toHaveBeenCalled();
    });
  });

  describe('switching off and the toolbar button', () => {
    it('removes the listener and passes the shortcut through again', () => {
      const documentRemove = jest.spyOn(document, 'removeEventListener');
      const { configure, sendMessage } = connect();
      configure({ enabled: true });

      configure({ enabled: false });
      const event = pressShortcut();

      expect(documentRemove).toHaveBeenCalledWith(
        'keydown',
        expect.any(Function)
      );
      expect(event.defaultPrevented).toBe(false);
      expect(windowKeyDown).toHaveBeenCalledTimes(1);
      expect(sendMessage).not.toHaveBeenCalled();
    });

    it("opens Grafana's palette while search is off", () => {
      const { sendMessage } = connect();

      openSearch();

      expect(triggerShortcut).toHaveBeenCalledTimes(1);
      expect(triggerShortcut).toHaveBeenCalledWith('search');
      expect(sendMessage).not.toHaveBeenCalled();
    });

    it("opens Grafana's palette before the messenger exists", () => {
      openSearch();

      expect(triggerShortcut).toHaveBeenCalledWith('search');
    });

    it('asks PMM UI to open search while search is on', () => {
      const { configure, sendMessage } = connect();
      configure({ enabled: true });

      openSearch();

      expect(sendMessage).toHaveBeenCalledTimes(1);
      expect(sendMessage).toHaveBeenCalledWith({ type: 'OPEN_SEARCH' });
      expect(triggerShortcut).not.toHaveBeenCalled();
    });
  });

  describe('focus restore', () => {
    const openFromInput = () => {
      const { configure } = connect();
      configure({ enabled: true });
      const input = append('input');
      input.focus();
      pressShortcut(input);
      const focus = jest.spyOn(input, 'focus');

      return { configure, input, focus };
    };

    it('refocuses the recorded element one frame after the frame regains focus', () => {
      const { input } = openFromInput();

      leaveAndReturnToFrame();

      expect(document.activeElement).toBe(document.body);
      expect(frames).toHaveLength(1);

      runFrame();

      expect(document.activeElement).toBe(input);
    });

    it('refocuses the toolbar button that opened search', () => {
      const { configure } = connect();
      configure({ enabled: true });
      const button = append('button');
      button.focus();

      openSearch();
      leaveAndReturnToFrame();
      runFrame();

      expect(document.activeElement).toBe(button);
    });

    it('does not refocus after Grafana navigated to another page', () => {
      const { focus } = openFromInput();

      window.history.pushState(null, '', '/graph/d/another-dashboard');
      leaveAndReturnToFrame();
      runFrame();

      expect(focus).not.toHaveBeenCalled();
    });

    it('does not refocus an element that was removed', () => {
      const { input, focus } = openFromInput();

      input.remove();
      leaveAndReturnToFrame();
      runFrame();

      expect(focus).not.toHaveBeenCalled();
    });

    it('does not move focus that is already elsewhere', () => {
      const { focus } = openFromInput();
      const other = append('input');

      window.dispatchEvent(new Event('focus'));
      other.focus();
      runFrame();

      expect(focus).not.toHaveBeenCalled();
      expect(document.activeElement).toBe(other);
    });

    it('drops the record when the user clicks inside the frame first', () => {
      const { focus } = openFromInput();

      document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }));
      leaveAndReturnToFrame();
      runFrame();

      expect(window.requestAnimationFrame).not.toHaveBeenCalled();
      expect(focus).not.toHaveBeenCalled();
    });

    it('drops the record when the click lands before the next frame', () => {
      const { focus } = openFromInput();

      leaveAndReturnToFrame();
      document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }));
      runFrame();

      expect(focus).not.toHaveBeenCalled();
    });

    it('drops the record when search is switched off', () => {
      const { configure, focus } = openFromInput();

      configure({ enabled: false });
      leaveAndReturnToFrame();
      runFrame();

      expect(window.requestAnimationFrame).not.toHaveBeenCalled();
      expect(focus).not.toHaveBeenCalled();
    });

    it('uses the record only once', () => {
      const { input, focus } = openFromInput();

      leaveAndReturnToFrame();
      runFrame();
      input.blur();
      leaveAndReturnToFrame();
      runFrame();

      expect(focus).toHaveBeenCalledTimes(1);
      expect(document.activeElement).toBe(document.body);
    });
  });
});
