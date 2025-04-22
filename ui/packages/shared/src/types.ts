export type MessageType =
  | 'MESSENGER_READY'
  | 'LOCATION_CHANGE'
  | 'DASHBOARD_VARIABLES';

export interface Message<T extends MessageType, V> {
  id?: string;
  type: T;
  data?: V;
}

export interface MessageListener<T extends MessageType, V> {
  type: T;
  onMessage: (message: Message<T, V>) => void;
}

export type MessengerReadyMessage = Message<'MESSENGER_READY', undefined>;

export type LocationChangeMessage = Message<'LOCATION_CHANGE', Location>;

export type DashboardVariablesMessage = Message<
  'DASHBOARD_VARIABLES',
  { url: string }
>;

export type DashboardVariablesResult = { url: string };
