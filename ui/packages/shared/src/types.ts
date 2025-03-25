export type MessageType = 'MESSENGER_READY';

export interface Message<T extends MessageType, V> {
  type: T;
  data?: V;
}

export interface MessageListener<T extends MessageType, V> {
  type: T;
  onMessage: (message: Message<T, V>) => void;
}

export type MessengerReadyMessage = Message<'MESSENGER_READY', undefined>;
