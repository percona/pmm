import { RefObject } from 'react';

export interface GrafanaContextProps {
  frameRef?: RefObject<HTMLIFrameElement>;
  isOnGrafanaPage: boolean;
  isFrameLoaded: boolean;
  isFullScreen: boolean;
  /** Last Grafana document title from iframe (for ADRE chat context). */
  grafanaDocumentTitle: string | null;
}
