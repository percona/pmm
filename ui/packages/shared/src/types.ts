export type ColorMode = 'light' | 'dark';

export type HistoryAction = 'PUSH' | 'POP' | 'REPLACE';

export type MessageType =
  | 'MESSENGER_READY'
  | 'LOCATION_CHANGE'
  | 'DASHBOARD_VARIABLES'
  | 'GRAFANA_READY'
  | 'DOCUMENT_TITLE_CHANGE'
  | 'GRAFANA_THEME_CHANGED'
  | 'CHANGE_THEME'
  | 'SETTINGS_CHANGED'
  | 'FRONTEND_SETTINGS_CHANGED'
  | 'SERVICE_ADDED'
  | 'SERVICE_DELETED'
  | 'TIMEZONE_CHANGED'
  | 'OPEN_ALERT_THRESHOLDS_MODAL';

export type LocationState = { fromGrafana?: boolean } | null;

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
    theme: ColorMode;
  }
>;

export type SettingsChangedMessage = Message<'SETTINGS_CHANGED'>;

export type FrontendSettingsChangedMessage =
  Message<'FRONTEND_SETTINGS_CHANGED'>;

export type ServiceAddedMessage = Message<'SERVICE_ADDED'>;

export type OpenAlertThresholdsModalMessage = Message<
  'OPEN_ALERT_THRESHOLDS_MODAL',
  { nodeId: string; nodeName: string }
>;
