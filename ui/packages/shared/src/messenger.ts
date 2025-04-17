import { Message, MessageListener, MessageType } from './types';

export class CrossFrameMessenger {
  private targetWindow?: Window;
  private eventListener = (e: MessageEvent) => this.onMessageReceived(e);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  private listeners: MessageListener<any, any>[] = [];

  setWindow(window: Window) {
    this.targetWindow = window;
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

    this.listeners.forEach((listener) => {
      if (listener.type === message.type) {
        listener.onMessage(message);
      }
    });
  }

  sendMessage<T extends MessageType, V>(message: Message<T, V>) {
    this.targetWindow?.postMessage(message);
  }
}
