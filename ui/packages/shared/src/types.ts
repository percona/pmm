export type Theme = 'light' | 'dark';

export type HistoryAction = 'PUSH' | 'POP' | 'REPLACE';

export type MessageType =
  | 'MESSENGER_READY'
  | 'LOCATION_CHANGE'
  | 'DASHBOARD_VARIABLES'
  | 'GRAFANA_READY'
  | 'DOCUMENT_TITLE_CHANGE'
  | 'CHANGE_THEME';

export interface Message<T extends MessageType = MessageType, V = undefined> {
  id?: string;
  type: T;
  source?: string;
  payload?: V;
}

export interface MessageListener<
  T extends MessageType = MessageType,
  V = undefined,
> {
  type: T;
  onMessage: (message: Message<T, V>) => void;
}

export type MessengerReadyMessage = Message<'MESSENGER_READY', undefined>;

export type LocationChangeMessage = Message<
  'LOCATION_CHANGE',
  Location & { title: string; action: HistoryAction }
>;

export type DashboardVariablesMessage = Message<
  'DASHBOARD_VARIABLES',
  { url: string }
>;

export type DashboardVariablesResult = { url: string };

export type DocumentTitleUpdateMessage = Message<
  'DOCUMENT_TITLE_CHANGE',
  {
    title: string;
  }
>;

export type ChangeThemeMessage = Message<
  'CHANGE_THEME',
  {
    theme: Theme;
  }
>;
