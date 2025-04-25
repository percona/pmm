import { Message, MessageListener, MessageType } from './types';
import { v4 as uuidv4 } from 'uuid';

export class CrossFrameMessenger {
  private source?: string;
  private window: Window = window;
  private targetOrigin?: string;
  private targetWindow?: Window;
  private fallbackSelector?: string;
  private eventListener = (e: MessageEvent) => this.onMessageReceived(e);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private listeners: MessageListener<any, any>[] = [];

  constructor(source?: string) {
    this.source = source;
  }

  setTargetOrigin(origin: string) {
    this.targetOrigin = origin;
    return this;
  }

  setWindow(window: Window) {
    this.window = window;
    return this;
  }

  setTargetWindow(window: Window, fallbackSelector?: string) {
    this.targetWindow = window;
    this.fallbackSelector = fallbackSelector;
    return this;
  }

  register() {
    this.window.addEventListener('message', this.eventListener);
    return this;
  }

  unregister() {
    this.window.removeEventListener('message', this.eventListener);
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

    if (process.env.NODE_ENV === 'development') {
      console.log(this.source, 'received', message);
    }

    this.listeners.forEach((listener) => {
      if (listener.type === message.type) {
        listener.onMessage(message);
      }
    });
  }

  sendMessage<T extends MessageType, V>(message: Message<T, V>) {
    // provide source and id if not present
    const msg = {
      ...message,
      source: this.source,
      id: message.id || uuidv4(),
    };

    if (process.env.NODE_ENV === 'development') {
      console.log(this.source, 'sending', msg);
    }

    this.getWindow()?.postMessage(msg, this.targetOrigin!);
  }

  sendMessageWithResult = <U, T extends MessageType, V>(
    message: Message<T, V>,
    timeoutMs = 10_000
  ): Promise<U> =>
    new Promise((resolve, reject) => {
      const timeoutId = setTimeout(reject, timeoutMs);
      const id = uuidv4();

      const listener: MessageListener<T, U> = {
        type: message.type,
        onMessage: (received) => {
          if (received.id === id) {
            clearTimeout(timeoutId);
            this.removeListener(listener);
            resolve(received.payload!);
          }
        },
      };

      this.addListener(listener);

      this.sendMessage({
        ...message,
        id,
      });
    });

  waitForMessage = <T extends MessageType>(
    type: T,
    timeoutMs = 10_000
  ): Promise<void> =>
    new Promise((resolve, reject) => {
      const timeoutId = setTimeout(reject, timeoutMs);

      const listener: MessageListener<T, void> = {
        type,
        onMessage: () => {
          clearTimeout(timeoutId);
          this.removeListener(listener);
          resolve();
        },
      };
      this.addListener(listener);
    });

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
