import { log as baseLog } from './utils';

const log = (...msg: string[]) => baseLog('messenger', ...msg);

export type MessageType =
  | 'MESSENGER_READY'
  | 'NAVIGATE_TO'
  | 'LOCATION_CHANGE'
  | 'LINK_VARIABLES'
  | 'LINK_VARIABLES_RESULT'
  | 'STARRED_DASHBOARDS';

export interface Message<T = any> {
  type: MessageType;
  data: T;
}

export interface MessageListener {
  type: MessageType;
  onMessage: (message: Message<any>) => void;
}

export class Messager {
  listeners: MessageListener[] = [];

  addListener(listener: MessageListener) {
    log('listener', listener.type);

    this.listeners.push(listener);
  }

  onMessageReceived(e: MessageEvent) {
    const message = e.data;

    if (!message) {
      return;
    }

    log('received', message.type);

    this.listeners.forEach((listener) => {
      if (message.type === listener.type) {
        listener.onMessage(e.data);
      }
    });
  }

  sendMessage<T = any>(message: Message<T>) {
    log('sending', message.type);

    window.top?.postMessage(message);
  }

  register() {
    log('register');

    window.addEventListener('message', (e) => this.onMessageReceived(e));

    const message: Message = {
      type: 'MESSENGER_READY',
      data: {},
    };

    this.sendMessage(message);
  }

  unregister() {
    log('unregister');

    this.listeners = [];
    window.removeEventListener('message', (e) => this.onMessageReceived(e));
  }
}

const messager = new Messager();

export default messager;
