import { CrossFrameMessenger } from './messenger';
import { Message, MessageListener } from './types';

const testData = {
  id: '6462b0bf-41f3-4048-a6fd-1611ba377f9c',
  id2: 'cac6a99a-b6c3-49a6-a50b-d528e30abc0e',
};

const setup = () => {
  const iframe = document.createElement('iframe');
  document.body.appendChild(iframe);

  const messenger = new CrossFrameMessenger('document')
    .setTargetOrigin('*')
    .setTargetWindow(iframe.contentWindow!, 'iframe')
    .register();

  const iframeMessenger = new CrossFrameMessenger('iframe')
    .setWindow(iframe.contentWindow!)
    .setTargetOrigin('*')
    .setTargetWindow(window!)
    .register();

  return { messenger, iframeMessenger };
};

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

    expect(
      iframeMessenger.waitForMessage('MESSENGER_READY', 1000)
    ).resolves.toBe(msg);
  });

  it('throws if waiting exceeds timeout', async () => {
    const { iframeMessenger } = setup();

    expect(
      iframeMessenger.waitForMessage('MESSENGER_READY', 1000)
    ).rejects.toBeUndefined();
  });
});
