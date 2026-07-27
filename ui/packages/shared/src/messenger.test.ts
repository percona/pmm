import { CrossFrameMessenger } from './messenger';
import { Message, MessageListener } from './types';

const testData = {
  id: '6462b0bf-41f3-4048-a6fd-1611ba377f9c',
  id2: 'cac6a99a-b6c3-49a6-a50b-d528e30abc0e',
};

const created: CrossFrameMessenger[] = [];

const track = <T extends CrossFrameMessenger>(messenger: T) => {
  created.push(messenger);
  return messenger;
};

const setup = () => {
  const iframe = document.createElement('iframe');
  document.body.appendChild(iframe);

  const messenger = track(
    new CrossFrameMessenger('document')
      .setTargetOrigin('*')
      .setTargetWindow(iframe.contentWindow!, 'iframe')
      .register()
  );

  const iframeMessenger = track(
    new CrossFrameMessenger('iframe')
      .setWindow(iframe.contentWindow!)
      .setTargetOrigin('*')
      .setTargetWindow(window!)
      .register()
  );

  return { messenger, iframeMessenger };
};

/** A stand-in for a target frame, so we can assert on what gets posted. */
const fakeFrame = () => {
  const postMessage = jest.fn();
  const frame = { postMessage, closed: false };
  return { ...frame, postMessage, window: frame as unknown as Window };
};

const ready = (messenger: CrossFrameMessenger) =>
  messenger.onMessageReceived({
    data: { type: 'GRAFANA_READY' },
  } as MessageEvent);

afterEach(() => {
  created.splice(0).forEach((messenger) => messenger.destroy());
  document.body.innerHTML = '';
});

describe('CrossFrameMessenger', () => {
  it('sends a message', (done) => {
    const { messenger, iframeMessenger } = setup();

    const msg: Message = {
      type: 'MESSENGER_READY',
    };

    iframeMessenger.addListener({
      type: 'MESSENGER_READY',
      onMessage: (received) => {
        expect(received).toEqual(expect.objectContaining(msg));
        done();
      },
    });

    messenger.sendMessage(msg);
  });

  it('receives message', async () => {
    const { messenger, iframeMessenger } = setup();

    const msg: Message = {
      type: 'MESSENGER_READY',
    };
    const listener: MessageListener = {
      type: 'MESSENGER_READY',
      onMessage: jest.fn(),
    };

    messenger.addListener(listener);

    iframeMessenger.sendMessage(msg);

    await messenger.waitForMessage('MESSENGER_READY');

    expect(listener.onMessage).toHaveBeenCalledWith(
      expect.objectContaining(msg)
    );
  });

  it("doesn't change id if provided", (done) => {
    const { messenger, iframeMessenger } = setup();

    const msg: Message = {
      id: testData.id,
      type: 'MESSENGER_READY',
    };

    iframeMessenger.addListener({
      type: 'MESSENGER_READY',
      onMessage: (received) => {
        expect(received.id).toBe(testData.id);
        done();
      },
    });

    messenger.sendMessage(msg);
  });

  it('assigns an id if not provided', (done) => {
    const { messenger, iframeMessenger } = setup();

    const msg: Message = {
      type: 'MESSENGER_READY',
    };

    iframeMessenger.addListener({
      type: 'MESSENGER_READY',
      onMessage: (received) => {
        expect(received.id).not.toBe(msg.id);
        expect(received.id).not.toBeUndefined();
        done();
      },
    });

    messenger.sendMessage(msg);
  });

  it('waits for correct result from a message', async () => {
    const { messenger, iframeMessenger } = setup();

    iframeMessenger.addListener({
      type: 'DASHBOARD_VARIABLES',
      onMessage: (msg) => {
        // same id but different type
        iframeMessenger.sendMessage({
          type: 'GRAFANA_READY',
        });

        // same type different id
        iframeMessenger.sendMessage({
          id: testData.id2,
          type: msg.type,
        });

        // same id and type
        iframeMessenger.sendMessage(msg);
      },
    });

    const result = await messenger.sendMessageWithResult({
      id: testData.id,
      type: 'DASHBOARD_VARIABLES',
    });

    expect(result).toBe(result);
  });

  it('waits for message to be received', async () => {
    const { messenger, iframeMessenger } = setup();
    const msg: Message = {
      id: testData.id,
      type: 'MESSENGER_READY',
    };

    setTimeout(() => {
      messenger.sendMessage(msg);
    }, 500);

    await expect(
      iframeMessenger.waitForMessage('MESSENGER_READY', 1000)
    ).resolves.toBeUndefined();
  });

  it('throws if waiting exceeds timeout', async () => {
    const { iframeMessenger } = setup();

    await expect(
      iframeMessenger.waitForMessage('MESSENGER_READY', 100)
    ).rejects.toThrow('Timed out waiting for "MESSENGER_READY"');
  });

  describe('listeners', () => {
    it('unsubscribes only the listener it belongs to', () => {
      const messenger = track(new CrossFrameMessenger('document'));
      const first = jest.fn();
      const second = jest.fn();

      const unsubscribe = messenger.addListener({
        type: 'SETTINGS_CHANGED',
        onMessage: first,
      });
      messenger.addListener({
        type: 'SETTINGS_CHANGED',
        onMessage: second,
      });

      unsubscribe();
      messenger.onMessageReceived({
        data: { type: 'SETTINGS_CHANGED' },
      } as MessageEvent);

      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledTimes(1);
    });

    it('delivers once when registered more than once', () => {
      const messenger = track(
        new CrossFrameMessenger('document').register().register()
      );
      const onMessage = jest.fn();
      messenger.addListener({ type: 'SERVICE_ADDED', onMessage });

      window.dispatchEvent(
        new MessageEvent('message', { data: { type: 'SERVICE_ADDED' } })
      );

      expect(onMessage).toHaveBeenCalledTimes(1);
    });

    it('stops delivering after destroy', () => {
      const messenger = track(new CrossFrameMessenger('document').register());
      const onMessage = jest.fn();
      messenger.addListener({ type: 'SERVICE_ADDED', onMessage });

      messenger.destroy();
      window.dispatchEvent(
        new MessageEvent('message', { data: { type: 'SERVICE_ADDED' } })
      );

      expect(onMessage).not.toHaveBeenCalled();
    });

    it('ignores messages without a type', () => {
      const messenger = track(new CrossFrameMessenger('document'));
      const onMessage = jest.fn();
      messenger.addListener({ type: 'SERVICE_ADDED', onMessage });

      messenger.onMessageReceived({ data: 'webpackHotUpdate' } as MessageEvent);

      expect(onMessage).not.toHaveBeenCalled();
    });
  });

  describe('target resolution', () => {
    it('resolves the target on every send', () => {
      const first = fakeFrame();
      const second = fakeFrame();
      let current = first.window;

      const messenger = track(
        new CrossFrameMessenger('PMM')
          .setTargetOrigin('*')
          .setTargetResolver(() => current)
      );

      messenger.sendMessage({ type: 'MESSENGER_READY' });
      current = second.window;
      messenger.sendMessage({ type: 'MESSENGER_READY' });

      expect(first.postMessage).toHaveBeenCalledTimes(1);
      expect(second.postMessage).toHaveBeenCalledTimes(1);
    });

    it('drops a closed target and falls back to the resolver', () => {
      const stale = fakeFrame();
      const fresh = fakeFrame();

      const messenger = track(
        new CrossFrameMessenger('PMM')
          .setTargetOrigin('*')
          .setTargetWindow(stale.window)
          .setTargetResolver(() => fresh.window)
      );

      stale.window.closed = true;
      messenger.sendMessage({ type: 'MESSENGER_READY' });

      expect(stale.postMessage).not.toHaveBeenCalled();
      expect(fresh.postMessage).toHaveBeenCalledTimes(1);
    });
  });

  describe('outbox', () => {
    it('queues state-sync messages and replays only the latest per type', () => {
      const frame = fakeFrame();
      const messenger = track(
        new CrossFrameMessenger('PMM').setTargetOrigin('*')
      );

      expect(
        messenger.sendMessage({
          type: 'CHANGE_THEME',
          payload: { theme: 'light' },
        })
      ).toBe(false);
      expect(
        messenger.sendMessage({
          type: 'CHANGE_THEME',
          payload: { theme: 'dark' },
        })
      ).toBe(false);

      messenger.setTargetResolver(() => frame.window);
      ready(messenger);

      expect(frame.postMessage).toHaveBeenCalledTimes(1);
      expect(frame.postMessage).toHaveBeenCalledWith(
        expect.objectContaining({
          type: 'CHANGE_THEME',
          payload: { theme: 'dark' },
        }),
        '*'
      );
    });

    it('drops messages that are meaningless once stale', () => {
      const frame = fakeFrame();
      const messenger = track(
        new CrossFrameMessenger('PMM').setTargetOrigin('*')
      );

      expect(
        messenger.sendMessage({
          type: 'DASHBOARD_VARIABLES',
          payload: { url: '/d/pmm-home' },
        })
      ).toBe(false);

      messenger.setTargetResolver(() => frame.window);
      ready(messenger);

      expect(frame.postMessage).not.toHaveBeenCalled();
    });

    it('replays messages sent before the peer announced itself', () => {
      const frame = fakeFrame();
      const messenger = track(
        new CrossFrameMessenger('PMM')
          .setTargetOrigin('*')
          .setTargetResolver(() => frame.window)
      );

      // Posted right away, but the peer may still be booting.
      expect(messenger.sendMessage({ type: 'SETTINGS_CHANGED' })).toBe(true);
      expect(messenger.isTargetReady()).toBe(false);

      ready(messenger);

      expect(frame.postMessage).toHaveBeenCalledTimes(2);
      expect(messenger.isTargetReady()).toBe(true);

      // Once the handshake is done nothing is retained any more.
      messenger.sendMessage({ type: 'SETTINGS_CHANGED' });
      ready(messenger);
      expect(frame.postMessage).toHaveBeenCalledTimes(3);
    });

    it('rejects a request when there is no target instead of hanging', async () => {
      const messenger = track(
        new CrossFrameMessenger('PMM').setTargetOrigin('*')
      );

      await expect(
        messenger.sendMessageWithResult({
          type: 'DASHBOARD_VARIABLES',
          payload: { url: '/d/pmm-home' },
        })
      ).rejects.toThrow('No target frame to handle "DASHBOARD_VARIABLES"');
    });
  });
});
