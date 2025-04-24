import { Message, MessageListener, MessageType } from './types';

export class CrossFrameMessenger {
  private name?: string;
  private targetWindow?: Window;
  private fallbackSelector?: string;
  private eventListener = (e: MessageEvent) => this.onMessageReceived(e);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private listeners: MessageListener<any, any>[] = [];

  constructor(name?: string) {
    this.name = name;
  }

  setWindow(window: Window, fallbackSelector?: string) {
    this.targetWindow = window;
    this.fallbackSelector = fallbackSelector;
  }

  register() {
    window.addEventListener('message', this.eventListener);
  }

  unregister() {
    window.removeEventListener('message', this.eventListener);
  }

  addListener<T extends MessageType, V>(listener: MessageListener<T, V>) {
    this.listeners.push(listener);
  }

  removeListener<T extends MessageType, V>(listener: MessageListener<T, V>) {
    this.listeners = this.listeners.filter((l) => l !== listener);
  }

  onMessageReceived<T extends MessageType, V>(e: MessageEvent) {
    const message = e.data as Message<T, V>;

    if (!message) {
      return;
    }

    console.log(this.name, 'received', message);

    this.listeners.forEach((listener) => {
      if (listener.type === message.type) {
        listener.onMessage(message);
      }
    });
  }

  sendMessage<T extends MessageType, V>(message: Message<T, V>) {
    if (!message.id) {
      message.id = self.crypto.randomUUID();
    }
    console.log(this.name, 'sending', message);
    this.getWindow()?.postMessage(message);
  }

  sendMessageWithResult<U, T extends MessageType, V>(
    message: Message<T, V>,
    timeout = 10_000
  ): Promise<U> {
    return new Promise((resolve, reject) => {
      let resolved = false;
      message.id = self.crypto.randomUUID();

      const listener: MessageListener<T, U> = {
        type: message.type,
        onMessage: (received) => {
          if (received.id === message.id) {
            resolved = true;
            this.removeListener(listener);
            resolve(received.data!);
          }
        },
      };

      this.addListener(listener);

      this.sendMessage(message);

      setTimeout(() => {
        if (!resolved) {
          reject();
        }
      }, timeout);
    });
  }

  waitForMessage<T extends MessageType>(
    type: T,
    timeout = 10_000
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      let resolved = false;

      const listener: MessageListener<T, void> = {
        type,
        onMessage: () => {
          resolved = true;
          this.removeListener(listener);
          resolve();
        },
      };
      this.addListener(listener);

      setTimeout(() => {
        if (!resolved) {
          reject();
        }
      }, timeout);
    });
  }

  private getWindow() {
    if (!this.targetWindow && this.fallbackSelector) {
      const iframe = document.querySelector<HTMLIFrameElement>(
        this.fallbackSelector
      );
      return this.targetWindow || iframe?.contentWindow;
    }

    return this.targetWindow;
  }
}
